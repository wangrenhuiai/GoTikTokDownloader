//go:build windows

package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

var (
	modUser32               = windows.NewLazySystemDLL("user32.dll")
	modOle32                = windows.NewLazySystemDLL("ole32.dll")
	procRegisterClassExW    = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW      = modUser32.NewProc("DefWindowProcW")
	procDestroyWindow       = modUser32.NewProc("DestroyWindow")
	procShowWindow          = modUser32.NewProc("ShowWindow")
	procUpdateWindow        = modUser32.NewProc("UpdateWindow")
	procGetMessageW         = modUser32.NewProc("GetMessageW")
	procTranslateMessage    = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW    = modUser32.NewProc("DispatchMessageW")
	procPostMessageW        = modUser32.NewProc("PostMessageW")
	procSetWindowTextW      = modUser32.NewProc("SetWindowTextW")
	procGetClientRect       = modUser32.NewProc("GetClientRect")
	procOleInitialize       = modOle32.NewProc("OleInitialize")
	modKernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procGetModuleHandleW    = modKernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId  = modKernel32.NewProc("GetCurrentThreadId")
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	swShow             = 5
	wmClose            = 0x0010
	wmDestroy          = 0x0002
	wmSize             = 0x0005
	cwUsedefault       = 0x80000000
)

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

type wndMsg struct {
	hwnd    windows.HWND
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	ptX     int32
	ptY     int32
	lPrivate uint32
}

type winHost struct {
	mu       sync.Mutex
	name     string
	title    string
	profile  string
	homepage string

	hwnd      uintptr
	threadID  uint32
	chromium  *edge.Chromium
	url       string
	ready     atomic.Bool
	visible   atomic.Bool
	running   atomic.Bool
	navDone   chan struct{}
	navOnce   sync.Once
	seq       atomic.Uint64
	pending   map[string]chan string
	pendingMu sync.Mutex
	stopCh    chan struct{}
	callCh    chan func()
	diag      diagState
}

const wmApp = 0x8000

func appData() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	return filepath.Join(dir, "GoTikTokDownloader")
}

func newEmbeddedHost(name, title, homepage string) BrowserHost {
	return &winHost{
		name:     name,
		title:    title,
		profile:  filepath.Join(appData(), "data", "browser", name),
		homepage: homepage,
		pending:  map[string]chan string{},
	}
}

func (h *winHost) Name() string      { return h.name }
func (h *winHost) ProfileDir() string { return h.profile }

func (h *winHost) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.running.Load() {
		return &BrowserError{Code: ErrAlreadyRunning, Message: h.name + " already started"}
	}
	if err := os.MkdirAll(h.profile, 0755); err != nil {
		return &BrowserError{Code: ErrProfileLocked, Message: "cannot create profile dir", Cause: err.Error()}
	}
	target := h.url
	if target == "" {
		target = h.homepage
	}
	h.navDone = make(chan struct{})
	h.navOnce = sync.Once{}
	h.stopCh = make(chan struct{})
	h.callCh = make(chan func(), 64)
	started := make(chan error, 1)
	go h.windowThread(target, started)
	select {
	case err := <-started:
		if err != nil {
			return err
		}
	case <-time.After(30 * time.Second):
		return &BrowserError{Code: ErrWebViewInit, Message: h.name + " webview init timed out"}
	}
	h.url = target
	h.running.Store(true)
	h.visible.Store(true)
	return nil
}

func (h *winHost) windowThread(target string, started chan error) {
	runtime.LockOSThread()
	procOleInitialize.Call(0)

	hInstPtr, _, _ := procGetModuleHandleW.Call(0)
	hInst := windows.Handle(hInstPtr)
	className, _ := windows.UTF16PtrFromString("GoTikTokDownloader.Browser." + h.name)
	wndProc := windows.NewCallback(h.wndProc)
	var wc wndClassExW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = wndProc
	wc.hInstance = hInst
	wc.lpszClassName = className
	wc.hbrBackground = windows.Handle(6) // COLOR_WINDOW+1
	ret, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		if errno, ok := err.(windows.Errno); ok && errno != 1410 { // CLASS_ALREADY_EXISTS ok
			started <- &BrowserError{Code: ErrWindowCreate, Message: "register class failed", Cause: err.Error()}
			return
		}
	}
	title, _ := windows.UTF16PtrFromString(h.title)
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow|wsVisible,
		cwUsedefault, cwUsedefault, 1100, 750,
		0, 0, uintptr(hInst), 0,
	)
	if hwnd == 0 {
		started <- &BrowserError{Code: ErrWindowCreate, Message: "create window failed", Cause: err.Error()}
		return
	}
	h.hwnd = hwnd
	tid, _, _ := procGetCurrentThreadId.Call()
	h.threadID = uint32(tid)
	h.diag.mu.Lock()
	h.diag.ownerTID = uint32(tid)
	h.diag.mu.Unlock()

	ch := edge.NewChromium()
	ch.DataPath = h.profile
	ch.SetErrorCallback(func(e error) { fmt.Println("WEBVIEW2-ERR:", e) }) // never os.Exit; surface via started/errors
	ch.MessageCallback = h.onWebMessage
	ch.NavigationCompletedCallback = func(_ *edge.ICoreWebView2, _ *edge.ICoreWebView2NavigationCompletedEventArgs) {
		h.ready.Store(true)
		h.diag.addCompleted()
		h.navOnce.Do(func() { close(h.navDone) })
	}
	// Embed pumps messages internally until init completes.
	if !ch.Embed(hwnd) {
		started <- &BrowserError{Code: ErrWebViewInit, Message: "embed webview failed"}
		return
	}
	h.chromium = ch
	h.ready.Store(true)
	h.navOnce.Do(func() { close(h.navDone) })
	started <- nil

	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	if target != "" {
		h.diag.addNav(target, h.threadID, false)
		ch.Navigate(target)
	}
	// Main message loop for this window thread.
	// NOTE: queued cross-thread calls are drained HERE (not inside wndProc):
	// COM calls made from inside a NewCallback wndproc crash (AV), because
	// the Go runtime is in a C-callback context there.
	var msg wndMsg
	drain := func() {
		for {
			select {
			case fn := <-h.callCh:
				fn()
			default:
				return
			}
		}
	}
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		drain()
		select {
		case <-h.stopCh:
			procDestroyWindow.Call(hwnd)
		default:
		}
	}
	h.running.Store(false)
	h.ready.Store(false)
	h.chromium = nil
	h.hwnd = 0
}

func (h *winHost) wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case wmApp:
		return 0 // drained in the message loop (see windowThread)
	case wmSize:
		if h.chromium != nil {
			h.chromium.Resize()
		}
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		h.running.Store(false)
		h.ready.Store(false)
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

const msgPrefix = "__gtd:"

// DiagSnapshot returns Go-side diagnostics (no DOM, no cookies).
func (h *winHost) DiagSnapshot() map[string]any {
	h.diag.mu.Lock()
	defer h.diag.mu.Unlock()
	h.mu.Lock()
	hasController := h.chromium != nil
	hasWindow := h.hwnd != 0
	url := h.url
	profile := h.profile
	h.mu.Unlock()
	navs := make([]navRecord, len(h.diag.navigations))
	copy(navs, h.diag.navigations)
	done := make([]string, len(h.diag.completedAt))
	copy(done, h.diag.completedAt)
	return map[string]any{
		"name":             h.name,
		"profile":          profile,
		"hasWindow":        hasWindow,
		"hasController":    hasController,
		"ownerTid":         h.diag.ownerTID,
		"currentUrl":       url,
		"navigations":      navs,
		"navCompletedAt":   done,
		"webview2Runtime":  RuntimeVersion(),
	}
}

// onThread runs fn on the window (WebView2 STA) thread and waits for it.
func (h *winHost) onThread(fn func()) {
	tid, _, _ := procGetCurrentThreadId.Call()
	h.mu.Lock()
	hwnd := h.hwnd
	running := h.running.Load()
	owner := h.threadID
	h.mu.Unlock()
	if !running || hwnd == 0 || uint32(tid) == owner {
		fn()
		return
	}
	done := make(chan struct{})
	h.callCh <- func() { fn(); close(done) }
	procPostMessageW.Call(hwnd, wmApp, 0, 0)
	<-done
}

func (h *winHost) onWebMessage(message string) {
	if !strings.HasPrefix(message, msgPrefix) {
		return
	}
	rest := strings.TrimPrefix(message, msgPrefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return
	}
	h.pendingMu.Lock()
	ch, ok := h.pending[parts[0]]
	if ok {
		delete(h.pending, parts[0])
	}
	h.pendingMu.Unlock()
	if ok {
		select {
		case ch <- parts[1]:
		default:
		}
	}
}

func (h *winHost) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running.Load() {
		h.ready.Store(false)
		return nil
	}
	select {
	case <-h.stopCh:
	default:
		close(h.stopCh)
	}
	if h.hwnd != 0 {
		procPostMessageW.Call(h.hwnd, wmClose, 0, 0)
	}
	h.running.Store(false)
	h.ready.Store(false)
	h.visible.Store(false)
	return nil
}

func (h *winHost) Navigate(url string) error {
	if url == "" {
		return &BrowserError{Code: ErrBadURL, Message: "empty url"}
	}
	h.mu.Lock()
	h.url = url
	ch := h.chromium
	running := h.running.Load()
	h.mu.Unlock()
	if !running || ch == nil {
		return &BrowserError{Code: ErrNotRunning, Message: h.name + " not started; url staged: " + url}
	}
	ctid, _, _ := procGetCurrentThreadId.Call()
	h.onThread(func() {
		h.diag.addNav(url, uint32(ctid), uint32(ctid) != h.threadID)
		ch.Navigate(url)
	})
	return nil
}

func (h *winHost) CurrentURL() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.url
}

// ExecuteScript evaluates JS and returns the JSON-encoded result via the
// postMessage bridge (real WebView2 round-trip, nothing hard-coded).
func (h *winHost) ExecuteScript(script string) (string, error) {
	if script == "" {
		return "", &BrowserError{Code: ErrBadScript, Message: "empty script"}
	}
	h.mu.Lock()
	ch := h.chromium
	running := h.running.Load()
	h.mu.Unlock()
	if !running || ch == nil {
		return "", &BrowserError{Code: ErrNotReady, Message: h.name + " not ready"}
	}
	id := fmt.Sprintf("%d", h.seq.Add(1))
	reply := make(chan string, 1)
	h.pendingMu.Lock()
	h.pending[id] = reply
	h.pendingMu.Unlock()
	bridge := `(function(){try{var r=(` + script + `);if(r&&r.then){r.then(function(v){window.chrome.webview.postMessage("__gtd:` + id + `:"+JSON.stringify(v))},function(e){window.chrome.webview.postMessage("__gtd:` + id + `:"+JSON.stringify("ERR:"+e))})}else{window.chrome.webview.postMessage("__gtd:` + id + `:"+JSON.stringify(r))}}catch(e){window.chrome.webview.postMessage("__gtd:` + id + `:"+JSON.stringify("ERR:"+e))}})()`
	h.onThread(func() { ch.Eval(bridge) })
	select {
	case res := <-reply:
		var out any
		if err := json.Unmarshal([]byte(res), &out); err != nil {
			return res, nil
		}
		pretty, _ := json.Marshal(out)
		return string(pretty), nil
	case <-time.After(15 * time.Second):
		h.pendingMu.Lock()
		delete(h.pending, id)
		h.pendingMu.Unlock()
		return "", &BrowserError{Code: ErrScriptFailed, Message: "script timed out"}
	}
}

func (h *winHost) IsReady() bool { return h.ready.Load() && h.running.Load() }

func (h *winHost) WaitReady(timeoutMs int) error {
	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	h.mu.Lock()
	done := h.navDone
	h.mu.Unlock()
	if done == nil {
		return &BrowserError{Code: ErrNotRunning, Message: h.name + " not started"}
	}
	select {
	case <-done:
		return nil
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return &BrowserError{Code: ErrPageTimeout, Message: h.name + " wait ready timed out"}
	}
}

func (h *winHost) SetTitle(title string) error {
	h.mu.Lock()
	h.title = title
	hwnd := h.hwnd
	h.mu.Unlock()
	if hwnd != 0 {
		t, _ := windows.UTF16PtrFromString(title)
		procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(t)))
	}
	return nil
}

func (h *winHost) Show() error {
	h.mu.Lock()
	ch := h.chromium
	hwnd := h.hwnd
	h.mu.Unlock()
	if ch == nil {
		return &BrowserError{Code: ErrNotRunning, Message: h.name + " not started"}
	}
	if hwnd != 0 {
		procShowWindow.Call(hwnd, swShow)
	}
	h.visible.Store(true)
	return nil
}

func (h *winHost) Hide() error {
	h.mu.Lock()
	ch := h.chromium
	h.mu.Unlock()
	if ch == nil {
		return &BrowserError{Code: ErrNotRunning, Message: h.name + " not started"}
	}
	_ = ch.Hide()
	h.visible.Store(false)
	return nil
}

// sessionCookies documents the login-indicating cookie names for reference.
// Values are never read. NOTE: go-webview2 v1.0.19's GetCookies slot faults
// (AV) in our linkage, so Layer-2 probing uses an auth-gated redirect check
// below instead of cookie enumeration. No fixed CSS selectors are assumed.
var sessionCookies = []string{"sessionid", "sessionid_ss", "sid_tt", "sid_guard", "uid_tt"}

// loginProbeScript same-origin fetches an auth-gated page with manual
// redirect: logged-out sessions get redirected to /login.
const loginProbeScript = `(fetch("https://www.tiktok.com/upload",{method:"HEAD",redirect:"manual",credentials:"include"}).then(function(r){return (r.type==="opaqueredirect"||r.status===301||r.status===302||r.status===307||r.status===308)?"not-logged-in":"logged-in"}).catch(function(e){return "unknown"}))`

func (h *winHost) LoginProbe() (string, error) {
	if !h.IsReady() {
		return "unknown", &BrowserError{Code: ErrNotRunning, Message: h.name + " not started"}
	}
	res, err := h.ExecuteScript(loginProbeScript)
	if err != nil {
		return "unknown", err
	}
	s := strings.Trim(strings.ToLower(strings.Trim(res, `"`)), " ")
	switch s {
	case "logged-in", "not-logged-in":
		return s, nil
	default:
		return "unknown", nil
	}
}
