package tiktok

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"gotiktokdownloader/backend/browser"
	"gotiktokdownloader/backend/logging"
)

// Search states — one string field, no competing booleans.
const (
	SearchIdle      = "Idle"
	SearchSearching = "Searching"
	SearchStopping  = "Stopping"
	SearchCompleted = "Completed"
	SearchError     = "Error"
)

// SearchOptions configures one search run.
type SearchOptions struct {
	MaxResults          int `json:"maxResults"`
	MaxNoNewRounds      int `json:"maxNoNewRounds"`
	ScrollDelayMs       int `json:"scrollDelayMs"`
	PageReadyTimeoutMs  int `json:"pageReadyTimeoutMs"`
}

func DefaultSearchOptions() SearchOptions {
	return SearchOptions{MaxResults: 100, MaxNoNewRounds: 5, ScrollDelayMs: 500, PageReadyTimeoutMs: 30000}
}

func (o SearchOptions) normalized() SearchOptions {
	if o.MaxResults <= 0 {
		o.MaxResults = 100
	}
	if o.MaxNoNewRounds <= 0 {
		o.MaxNoNewRounds = 5
	}
	if o.ScrollDelayMs < 200 {
		o.ScrollDelayMs = 200
	}
	if o.ScrollDelayMs > 3000 {
		o.ScrollDelayMs = 3000
	}
	if o.PageReadyTimeoutMs <= 0 {
		o.PageReadyTimeoutMs = 30000
	}
	return o
}

// SearchEvents carries search lifecycle to the UI layer.
type SearchEvents struct {
	OnStarted   func(keyword string)
	OnCandidate func(c VideoCandidate)
	OnProgress  func(found, round int)
	OnCompleted func(keyword string, total int)
	OnError     func(keyword, message string)
}

// SearchRunner executes keyword search on the single TikTok browser.
// Browser (Navigate/ExecuteScript/WaitPageReady) and Collector (DOM parse)
// stay separated; Runner only orchestrates.
type SearchRunner struct {
	mu      sync.Mutex
	state   string
	stopCh  chan struct{}
	results map[string]VideoCandidate
	order   []string
	keyword string
	rounds  int
	log     *logging.Logger
}

func NewSearchRunner(log *logging.Logger) *SearchRunner {
	return &SearchRunner{state: SearchIdle, log: log}
}

func (r *SearchRunner) State() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

func (r *SearchRunner) setState(s string) {
	r.mu.Lock()
	r.state = s
	r.mu.Unlock()
}

func (r *SearchRunner) Results() []VideoCandidate {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]VideoCandidate, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.results[id])
	}
	return out
}

func (r *SearchRunner) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == SearchSearching || r.state == SearchStopping {
		return
	}
	r.results = map[string]VideoCandidate{}
	r.order = nil
	r.keyword = ""
	r.rounds = 0
	r.state = SearchIdle
}

// Stop signals the loop to halt after the current round.
// Already-collected results are kept; the browser stays open.
func (r *SearchRunner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == SearchSearching && r.stopCh != nil {
		r.state = SearchStopping
		close(r.stopCh)
	}
}

func (r *SearchRunner) stopped() bool {
	r.mu.Lock()
	ch := r.stopCh
	r.mu.Unlock()
	if ch == nil {
		return true
	}
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// Start runs a search synchronously (caller decides goroutine).
// Never touches Downloader/yt-dlp/FFmpeg.
func (r *SearchRunner) Start(host browser.BrowserHost, keyword string, opts SearchOptions, ev SearchEvents) error {
	opts = opts.normalized()
	r.mu.Lock()
	if r.state == SearchSearching || r.state == SearchStopping {
		r.mu.Unlock()
		return errors.New("search already running")
	}
	r.state = SearchSearching
	r.stopCh = make(chan struct{})
	r.results = map[string]VideoCandidate{}
	r.order = nil
	r.keyword = keyword
	r.rounds = 0
	r.mu.Unlock()

	r.log.Infof("search start: keyword=%s", keyword)
	searchURL := "https://www.tiktok.com/search?q=" + url.QueryEscape(keyword)
	r.log.Infof("search url: %s", searchURL)
	if ev.OnStarted != nil {
		ev.OnStarted(keyword)
	}

	if err := host.Navigate(searchURL); err != nil {
		return r.fail(keyword, ev, fmt.Sprintf("navigate failed: %v", err))
	}
	if w, ok := host.(interface{ WaitPageReady(int) (string, error) }); ok {
		if _, err := w.WaitPageReady(opts.PageReadyTimeoutMs); err != nil {
			return r.fail(keyword, ev, fmt.Sprintf("page not ready: %v", err))
		}
		r.log.Infof("search page ready: true")
	} else if err := host.WaitReady(opts.PageReadyTimeoutMs); err != nil {
		return r.fail(keyword, ev, fmt.Sprintf("page not ready: %v", err))
	}
	// Fallback: /search sometimes serves a server-error page to anonymous
	// sessions (data-e2e="search-error-title"). Single-token keywords can
	// fall back to the public tag page in the SAME browser instance.
	if r.searchErrorPage(host) {
		if !singleToken(keyword) {
			return r.fail(keyword, ev, "search page returned server error and keyword is not a single tag token")
		}
		tagURL := "https://www.tiktok.com/tag/" + url.PathEscape(keyword)
		r.log.Infof("search fallback: search page error, navigating %s", tagURL)
		if err := host.Navigate(tagURL); err != nil {
			return r.fail(keyword, ev, fmt.Sprintf("fallback navigate failed: %v", err))
		}
		if w, ok := host.(interface{ WaitPageReady(int) (string, error) }); ok {
			if _, err := w.WaitPageReady(opts.PageReadyTimeoutMs); err != nil {
				return r.fail(keyword, ev, fmt.Sprintf("fallback page not ready: %v", err))
			}
		}
	}
	noNewRounds := 0
	for {
		if r.stopped() {
			r.setState(SearchCompleted)
			r.log.Infof("search stopped by user: keyword=%s found=%d", keyword, len(r.order))
			if ev.OnCompleted != nil {
				ev.OnCompleted(keyword, len(r.order))
			}
			return nil
		}
		r.mu.Lock()
		r.rounds++
		round := r.rounds
		r.mu.Unlock()

		scanJSON, err := host.ExecuteScript(ScanJS)
		if err != nil {
			return r.fail(keyword, ev, fmt.Sprintf("scan failed: %v", err))
		}
		batch, err := ParseScanResult(scanJSON, keyword)
		if err != nil {
			return r.fail(keyword, ev, fmt.Sprintf("parse failed: %v", err))
		}
		r.mu.Lock()
		added, dups := 0, 0
		for _, c := range batch {
			if _, ok := r.results[c.VideoID]; ok {
				dups++
				continue
			}
			r.results[c.VideoID] = c
			r.order = append(r.order, c.VideoID)
			added++
		}
		total := len(r.order)
		r.mu.Unlock()
		r.log.Infof("search scan: round=%d found=%d new=%d duplicate=%d total=%d", round, len(batch), added, dups, total)
		if ev.OnProgress != nil {
			ev.OnProgress(total, round)
		}
		if added > 0 && ev.OnCandidate != nil {
			r.mu.Lock()
			tail := r.order[total-added:]
			r.mu.Unlock()
			for _, id := range tail {
				r.mu.Lock()
				c := r.results[id]
				r.mu.Unlock()
				ev.OnCandidate(c)
			}
		}
		if total >= opts.MaxResults {
			r.setState(SearchCompleted)
			r.log.Infof("search completed: keyword=%s total=%d (maxResults reached)", keyword, total)
			if ev.OnCompleted != nil {
				ev.OnCompleted(keyword, total)
			}
			return nil
		}
		if added == 0 {
			noNewRounds++
		} else {
			noNewRounds = 0
		}
		if noNewRounds >= opts.MaxNoNewRounds {
			r.setState(SearchCompleted)
			r.log.Infof("search completed: keyword=%s total=%d (no new content)", keyword, total)
			if ev.OnCompleted != nil {
				ev.OnCompleted(keyword, total)
			}
			return nil
		}
		if _, err := host.ExecuteScript(ScrollJS); err != nil {
			return r.fail(keyword, ev, fmt.Sprintf("scroll failed: %v", err))
		}
		r.log.Infof("search scroll: round=%d", round)
		time.Sleep(time.Duration(opts.ScrollDelayMs) * time.Millisecond)
	}
}

func (r *SearchRunner) fail(keyword string, ev SearchEvents, msg string) error {
	r.setState(SearchError)
	r.log.Errorf("search error: keyword=%s %s", keyword, msg)
	if ev.OnError != nil {
		ev.OnError(keyword, msg)
	}
	return errors.New(msg)
}

// searchErrorPage detects TikTok's server-error page marker.
func (r *SearchRunner) searchErrorPage(host browser.BrowserHost) bool {
	res, err := host.ExecuteScript(`!!document.querySelector('[data-e2e="search-error-title"]')`)
	if err != nil {
		return false
	}
	return res == "true"
}

func singleToken(keyword string) bool {
	if keyword == "" {
		return false
	}
	for _, c := range keyword {
		if c == ' ' || c == '\t' || c == '\n' {
			return false
		}
	}
	return true
}
