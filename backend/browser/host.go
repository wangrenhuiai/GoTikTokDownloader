package browser

import (
	"os"
	"path/filepath"
)

// BrowserError is a structured browser failure. Never panics, never carries cookies.
type BrowserError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   string `json:"cause,omitempty"`
}

func (e *BrowserError) Error() string {
	if e.Cause != "" {
		return e.Code + ": " + e.Message + " (" + e.Cause + ")"
	}
	return e.Code + ": " + e.Message
}

// Error codes.
const (
	ErrAlreadyRunning = "ALREADY_RUNNING"
	ErrNotRunning     = "NOT_RUNNING"
	ErrNotReady       = "NOT_READY"
	ErrBadURL         = "BAD_URL"
	ErrBadScript      = "BAD_SCRIPT"
	ErrWebViewInit    = "WEBVIEW2_INIT_FAILED"
	ErrWindowCreate   = "WINDOW_CREATE_FAILED"
	ErrProfileLocked  = "PROFILE_LOCKED"
	ErrNavFailed      = "NAVIGATION_FAILED"
	ErrScriptFailed   = "SCRIPT_FAILED"
	ErrPageTimeout    = "PAGE_TIMEOUT"
	ErrUnsupported    = "UNSUPPORTED_PLATFORM"
)

// BrowserHost abstracts an independent embedded TikTok browser instance.
// Business layers (TikTokService/SearchService/Scheduler/Downloader) must
// depend on this interface, never on WebView2/COM types.
type BrowserHost interface {
	Name() string
	Start() error
	Stop() error
	Navigate(url string) error
	CurrentURL() string
	ExecuteScript(script string) (string, error)
	IsReady() bool
	WaitReady(timeoutMs int) error
	SetTitle(title string) error
	Show() error
	Hide() error
	ProfileDir() string
	// LoginProbe reports "logged-in" | "not-logged-in" | "unknown"
	// using cookie NAMES only (never values).
	LoginProbe() (string, error)
}

// Manager owns the single TikTok browser instance.
// One software = one TikTok WebView2. Login, search and video pages all
// reuse this instance via Navigate.
type Manager struct {
	TikTok BrowserHost
}

// NewManager builds the single TikTok host.
func NewManager() *Manager {
	return &Manager{
		TikTok: newEmbeddedHost("tiktok", "TikTok 浏览器", "https://www.tiktok.com/"),
	}
}

// LegacyProfiles reports whether pre-convergence profiles still exist.
// They are NOT used anymore; kept only so a future migration can inspect them.
func LegacyProfiles() map[string]bool {
	out := map[string]bool{}
	for _, n := range []string{"tiktok-main", "tiktok-search"} {
		st, err := os.Stat(filepath.Join(appData(), "data", "browser", n))
		out[n] = err == nil && st.IsDir()
	}
	return out
}
