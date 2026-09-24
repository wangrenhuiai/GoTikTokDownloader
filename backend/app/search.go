package app

import (
	"net/url"
	"strings"
	"time"

	"gotiktokdownloader/backend/tiktok"
)

// --- Keyword search (Phase 3B) ---

func (a *App) ensureRunner() *tiktok.SearchRunner {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	if a.runner == nil {
		a.runner = tiktok.NewSearchRunner(a.log)
	}
	return a.runner
}

// StartSearch launches a keyword search on the single TikTok browser.
// Runs in background; progress arrives via search:* events.
func (a *App) StartSearch(keyword string, maxResults, maxNoNewRounds, scrollDelayMs int) (string, error) {
	a.ensureBrowsers()
	r := a.ensureRunner()
	if r.State() == tiktok.SearchSearching || r.State() == tiktok.SearchStopping {
		return r.State(), nil
	}
	if keyword == "" {
		return r.State(), errStr("empty keyword")
	}
	if _, err := a.OpenTikTokBrowser(); err != nil {
		return r.State(), err
	}
	opts := tiktok.DefaultSearchOptions()
	if maxResults > 0 {
		opts.MaxResults = maxResults
	}
	if maxNoNewRounds > 0 {
		opts.MaxNoNewRounds = maxNoNewRounds
	}
	if scrollDelayMs > 0 {
		opts.ScrollDelayMs = scrollDelayMs
	}
	ev := tiktok.SearchEvents{
		OnStarted: func(kw string) {
			a.Emit("search:started", map[string]any{"keyword": kw})
		},
		OnCandidate: func(c tiktok.VideoCandidate) {
			a.saveCandidate(c)
			a.Emit("search:candidate", c)
		},
		OnProgress: func(found, round int) {
			a.Emit("search:progress", map[string]any{"keyword": keyword, "found": found, "round": round})
		},
		OnCompleted: func(kw string, total int) {
			a.Emit("search:completed", map[string]any{"keyword": kw, "found": total})
			a.Emit("system:status", "search completed")
		},
		OnError: func(kw, msg string) {
			a.Emit("search:error", map[string]any{"keyword": kw, "message": msg})
		},
	}
	go func() {
		if err := r.Start(a.browsers.TikTok, keyword, opts, ev); err != nil {
			a.log.Errorf("search run ended: %v", err)
		}
	}()
	return tiktok.SearchSearching, nil
}

func (a *App) StopSearch() string {
	r := a.ensureRunner()
	r.Stop()
	return r.State()
}

func (a *App) SearchState() string {
	return a.ensureRunner().State()
}

func (a *App) SearchResults() []tiktok.VideoCandidate {
	return a.ensureRunner().Results()
}

func (a *App) ClearSearch() string {
	r := a.ensureRunner()
	r.Clear()
	return r.State()
}

// SearchOptions returns the effective defaults (for UI display).
func (a *App) SearchOptions() tiktok.SearchOptions {
	return tiktok.DefaultSearchOptions()
}

// NavigateTo opens a TikTok video URL in the single embedded browser.
// It never creates a second browser instance.
func (a *App) NavigateTo(rawURL string) (BrowserState, error) {
	a.ensureBrowsers()
	canon, _, _, ok := tiktok.ParseVideoURL(rawURL)
	if !ok {
		// Allow plain search/tag navigation as well, but stay on TikTok.
		if !isTikTokURL(rawURL) {
			return snap(a.browsers.TikTok), errStr("only tiktok.com URLs are allowed")
		}
		canon = rawURL
	}
	st, err := a.OpenTikTokBrowser()
	if err != nil {
		return st, err
	}
	if err := a.browsers.TikTok.Navigate(canon); err != nil {
		return snap(a.browsers.TikTok), err
	}
	return snap(a.browsers.TikTok), nil
}

func isTikTokURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "tiktok.com" || strings.HasSuffix(host, ".tiktok.com")
}

// saveCandidate persists to the existing videos table (no schema change):
// id=VideoID, url=canonical, author_id=AuthorName, title, status=discovered.
func (a *App) saveCandidate(c tiktok.VideoCandidate) {
	if a.db == nil {
		return
	}
	_, err := a.db.Exec(
		`INSERT INTO videos(id, author_id, url, title, status, updated_at) VALUES(?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET title=excluded.title, updated_at=excluded.updated_at`,
		c.VideoID, c.AuthorName, c.URL, c.Title, "discovered", time.Now().Unix(),
	)
	if err != nil {
		a.log.Errorf("save candidate %s: %v", c.VideoID, err)
	}
}
