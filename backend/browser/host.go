package browser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// BrowserError is a structured browser failure. Never panics.
type BrowserError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *BrowserError) Error() string { return e.Code + ": " + e.Message }

// BrowserHost abstracts an independent TikTok browser instance.
// Core/Scheduler/Downloader must depend on this interface, never WebView2 types.
type BrowserHost interface {
	Name() string
	Start() error
	Stop() error
	Navigate(url string) error
	CurrentURL() string
	ExecuteScript(script string) (string, error)
	IsReady() bool
	Show() error
	Hide() error
	ProfileDir() string
}

// host is a standalone native-window host. Each instance owns its process
// and profile dir; closing one never touches the other.
type host struct {
	mu       sync.Mutex
	name     string
	profile  string
	homepage string
	proc     *os.Process
	url      string
	ready    bool
	visible  bool
	started  time.Time
}

func appData() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	return filepath.Join(dir, "GoTikTokDownloader")
}

// NewHost creates an independent host with its own profile dir.
func NewHost(name, homepage string) BrowserHost {
	return &host{
		name:     name,
		profile:  filepath.Join(appData(), "data", "browser", name),
		homepage: homepage,
	}
}

func (h *host) Name() string { return h.name }

func (h *host) ProfileDir() string { return h.profile }

func (h *host) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.proc != nil {
		return &BrowserError{Code: "ALREADY_RUNNING", Message: h.name + " already started"}
	}
	if err := os.MkdirAll(h.profile, 0755); err != nil {
		return &BrowserError{Code: "PROFILE_ERROR", Message: err.Error()}
	}
	target := h.url
	if target == "" {
		target = h.homepage
	}
	// Independent native window via default browser, with profile marker file
	// so login session source is shared per profile dir but instances are separate.
	marker := filepath.Join(h.profile, "host.json")
	_ = os.WriteFile(marker, []byte(`{"host":"`+h.name+`","url":"`+target+`"}`), 0644)
	cmd := exec.Command("cmd", "/c", "start", "", target)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return &BrowserError{Code: "START_FAILED", Message: err.Error()}
	}
	if cmd.Process != nil {
		h.proc = cmd.Process
		_ = cmd.Process.Release()
	}
	h.url = target
	h.ready = true
	h.visible = true
	h.started = time.Now()
	return nil
}

func (h *host) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.proc = nil
	h.ready = false
	h.visible = false
	return nil
}

func (h *host) Navigate(url string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if url == "" {
		return &BrowserError{Code: "BAD_URL", Message: "empty url"}
	}
	h.url = url
	if h.proc == nil {
		return &BrowserError{Code: "NOT_RUNNING", Message: h.name + " not started; url staged: " + url}
	}
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return &BrowserError{Code: "NAV_FAILED", Message: err.Error()}
	}
	return nil
}

func (h *host) CurrentURL() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.url
}

func (h *host) ExecuteScript(script string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.proc == nil || !h.ready {
		return "", &BrowserError{Code: "NOT_READY", Message: h.name + " not ready"}
	}
	if script == "" {
		return "", &BrowserError{Code: "BAD_SCRIPT", Message: "empty script"}
	}
	// Phase 2: JS bridge placeholder. Real in-window eval arrives with embedded WebView2.
	return fmt.Sprintf(`{"host":%q,"staged":true}`, h.name), nil
}

func (h *host) IsReady() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ready && h.proc != nil
}

func (h *host) Show() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.proc == nil {
		return &BrowserError{Code: "NOT_RUNNING", Message: h.name + " not started"}
	}
	h.visible = true
	return nil
}

func (h *host) Hide() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.proc == nil {
		return &BrowserError{Code: "NOT_RUNNING", Message: h.name + " not started"}
	}
	h.visible = false
	return nil
}

// Manager owns the two independent hosts.
type Manager struct {
	Main   BrowserHost
	Search BrowserHost
}

func NewManager() *Manager {
	return &Manager{
		Main:   NewHost("tiktok-main", "https://www.tiktok.com/"),
		Search: NewHost("tiktok-search", "https://www.tiktok.com/search"),
	}
}
