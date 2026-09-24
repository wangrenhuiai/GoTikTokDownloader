//go:build windows

package browser

import (
	"sync"
	"time"

	"github.com/wailsapp/go-webview2/webviewloader"
)

// navRecord tracks one navigation request (Go side only, no DOM guessing).
type navRecord struct {
	URL        string `json:"url"`
	CallerTID  uint32 `json:"callerTid"`
	Dispatched bool   `json:"dispatched"`
	At         string `json:"at"`
}

// diagState is additive instrumentation only; it never changes browser behavior.
type diagState struct {
	mu          sync.Mutex
	ownerTID    uint32
	navigations []navRecord
	completedAt []string
}

func (d *diagState) addNav(url string, tid uint32, dispatched bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.navigations = append(d.navigations, navRecord{URL: url, CallerTID: tid, Dispatched: dispatched, At: time.Now().Format(time.RFC3339)})
}

func (d *diagState) addCompleted() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.completedAt = append(d.completedAt, time.Now().Format(time.RFC3339))
}

// RuntimeVersion returns the installed WebView2 version string.
func RuntimeVersion() string {
	v, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString("")
	if err != nil {
		return "error: " + err.Error()
	}
	if v == "" {
		return "not found"
	}
	return v
}
