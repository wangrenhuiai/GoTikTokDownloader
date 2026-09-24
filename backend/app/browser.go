package app

import (
	"gotiktokdownloader/backend/browser"
	"gotiktokdownloader/backend/tiktok"
)

// BrowserState is a JSON-safe snapshot for the UI.
type BrowserState struct {
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	URL     string `json:"url"`
	Profile string `json:"profile"`
}

func snap(h browser.BrowserHost) BrowserState {
	return BrowserState{Name: h.Name(), Ready: h.IsReady(), URL: h.CurrentURL(), Profile: h.ProfileDir()}
}

func (a *App) ensureBrowsers() {
	if a.browsers == nil {
		a.browsers = browser.NewManager()
	}
	if a.login == nil {
		a.login = tiktok.NewLoginService()
	}
	if a.search == nil {
		a.search = tiktok.NewSearchService()
	}
}

// --- Main browser ---

func (a *App) OpenMainBrowser() (BrowserState, error) {
	a.ensureBrowsers()
	a.log.Infof("MainBrowserHost starting (embedded WebView2)")
	if err := a.browsers.Main.Start(); err != nil {
		if be, ok := err.(*browser.BrowserError); ok && be.Code == browser.ErrAlreadyRunning {
			return snap(a.browsers.Main), nil
		}
		a.log.Errorf("MainBrowserHost start failed: %v", err)
		return snap(a.browsers.Main), err
	}
	a.Emit(EvtSystemLog, "MainBrowserHost started")
	a.Emit(EvtSystemStatus, "main browser open")
	return snap(a.browsers.Main), nil
}

func (a *App) CloseMainBrowser() error {
	a.ensureBrowsers()
	a.log.Infof("MainBrowserHost stopped")
	return a.browsers.Main.Stop()
}

func (a *App) GetMainBrowserState() BrowserState {
	a.ensureBrowsers()
	return snap(a.browsers.Main)
}

// --- Search browser ---

func (a *App) OpenSearchBrowser(keyword string) (BrowserState, error) {
	a.ensureBrowsers()
	a.log.Infof("SearchBrowserHost starting (keyword len=%d)", len(keyword))
	if err := a.browsers.Search.Start(); err != nil {
		if be, ok := err.(*browser.BrowserError); ok && be.Code == browser.ErrAlreadyRunning {
			return snap(a.browsers.Search), nil
		}
		a.log.Errorf("SearchBrowserHost start failed: %v", err)
		return snap(a.browsers.Search), err
	}
	if keyword != "" {
		if err := a.browsers.Search.Navigate("https://www.tiktok.com/search?q=" + keyword); err != nil {
			a.log.Errorf("SearchBrowserHost navigate failed: %v", err)
		} else {
			a.Emit(EvtSystemLog, "Phase 3A: embedded search browser navigated")
		}
	}
	a.Emit(EvtSystemStatus, "search browser open")
	return snap(a.browsers.Search), nil
}

func (a *App) CloseSearchBrowser() error {
	a.ensureBrowsers()
	a.log.Infof("SearchBrowserHost stopped")
	return a.browsers.Search.Stop()
}

func (a *App) GetSearchBrowserState() BrowserState {
	a.ensureBrowsers()
	return snap(a.browsers.Search)
}

// --- Script / page ---

// RunScript executes JS in the chosen embedded browser ("main"|"search").
func (a *App) RunScript(which, script string) (string, error) {
	a.ensureBrowsers()
	h := a.browsers.Main
	if which == "search" {
		h = a.browsers.Search
	}
	res, err := h.ExecuteScript(script)
	if err != nil {
		a.log.Errorf("RunScript(%s) failed: %v", which, err)
		return "", err
	}
	return res, nil
}

// PageInfo bundles title+url+body length via three real script round-trips.
func (a *App) PageInfo(which string) (map[string]string, error) {
	a.ensureBrowsers()
	h := a.browsers.Main
	if which == "search" {
		h = a.browsers.Search
	}
	out := map[string]string{}
	for k, js := range map[string]string{
		"title": "document.title",
		"url":   "location.href",
		"body":  "document.body ? document.body.innerText.length : -1",
	} {
		v, err := h.ExecuteScript(js)
		if err != nil {
			return out, err
		}
		out[k] = v
	}
	return out, nil
}

func (a *App) WaitBrowserReady(which string, timeoutMs int) error {
	a.ensureBrowsers()
	if which == "search" {
		return a.browsers.Search.WaitReady(timeoutMs)
	}
	return a.browsers.Main.WaitReady(timeoutMs)
}

// --- Login ---

func (a *App) GetLoginURL() string {
	a.ensureBrowsers()
	return a.login.LoginURL()
}

func (a *App) OpenLoginWindow() (BrowserState, error) {
	a.ensureBrowsers()
	a.log.Infof("TikTok login browser starting (embedded)")
	st, err := a.OpenMainBrowser()
	if err != nil {
		return st, err
	}
	if err := a.browsers.Main.Navigate(a.login.LoginURL()); err != nil {
		return snap(a.browsers.Main), err
	}
	a.Emit(EvtSystemLog, "TikTok navigation: login page (embedded)")
	return snap(a.browsers.Main), nil
}

// GetLoginState: Layer1 local profile + Layer2 live session probe.
func (a *App) GetLoginState() string {
	a.ensureBrowsers()
	if !a.login.SessionExists() {
		return "not-logged-in"
	}
	if a.browsers != nil && a.browsers.Main.IsReady() {
		if probe, err := a.browsers.Main.LoginProbe(); err == nil && probe != "unknown" {
			return probe
		}
	}
	if a.login.CheckLoginState() == "logged-in" {
		return "logged-in (unverified)"
	}
	return "not-logged-in"
}

// ProbeLoginState forces a live WebView2 session check (cookie names only).
func (a *App) ProbeLoginState() string {
	a.ensureBrowsers()
	if a.browsers == nil || !a.browsers.Main.IsReady() {
		return "unknown (browser not open)"
	}
	probe, err := a.browsers.Main.LoginProbe()
	if err != nil {
		return "unknown (" + err.Error() + ")"
	}
	return probe
}

func (a *App) ConfirmLoggedIn() string {
	a.ensureBrowsers()
	if err := a.login.MarkLoggedIn(); err != nil {
		return "error: " + err.Error()
	}
	a.log.Infof("TikTok login state changed: logged-in")
	a.Emit(EvtSystemStatus, "tiktok logged-in")
	return "logged-in"
}

// --- Future reservations ---

func (a *App) SearchKeyword(keyword string) (string, error) {
	a.ensureBrowsers()
	if err := a.search.Search(keyword); err != nil {
		return "Phase 3A: embedded search browser ready; auto-crawl not implemented", err
	}
	return "ok", nil
}

// --- Diagnostics (Phase 3A-diag, additive only) ---

func (a *App) diagHost(which string) browser.BrowserHost {
	a.ensureBrowsers()
	if which == "search" {
		return a.browsers.Search
	}
	return a.browsers.Main
}

// DiagRuntime returns Go-side WebView2 diagnostics.
func (a *App) DiagRuntime(which string) map[string]any {
	h := a.diagHost(which)
	if s, ok := h.(interface{ DiagSnapshot() map[string]any }); ok {
		return s.DiagSnapshot()
	}
	return map[string]any{"name": h.Name(), "note": "no native snapshot on this platform"}
}

// DiagDOM runs the exact DOM snapshot script from the diag spec.
func (a *App) DiagDOM(which string) (string, error) {
	const js = `({href: location.href, title: document.title, readyState: document.readyState, bodyExists: !!document.body, bodyLength: document.body ? document.body.innerHTML.length : -1, htmlLength: document.documentElement ? document.documentElement.outerHTML.length : -1, textLength: document.body ? document.body.innerText.length : -1})`
	return a.RunScript(which, js)
}

// DiagUA returns userAgent + navigator details (record only, never modified).
func (a *App) DiagUA(which string) (string, error) {
	const js = `({ua: navigator.userAgent, language: navigator.language, platform: navigator.platform, webdriver: navigator.webdriver})`
	return a.RunScript(which, js)
}

// DiagNetwork summarizes resource response statuses via Performance API
// (responseStatus needs Chromium 109+; failures sampled, max 5 URLs).
func (a *App) DiagNetwork(which string) (string, error) {
	const js = `(function(){var out={nav:null,hist:{},failSample:[]};try{var n=performance.getEntriesByType('navigation')[0];if(n){out.nav={url:n.name,redirects:n.redirectCount,status:n.responseStatus||null,domContent:n.domContentLoadedEventEnd-n.startTime,load:n.loadEventEnd-n.startTime}}}catch(e){out.nav='ERR:'+e}try{var rs=performance.getEntriesByType('resource');for(var i=0;i<rs.length;i++){var s=rs[i].responseStatus||0;var k=String(s);out.hist[k]=(out.hist[k]||0)+1;if((s>=400||s===0)&&out.failSample.length<5){out.failSample.push({status:s,url:String(rs[i].name).slice(0,160)})}}}catch(e){out.histErr=String(e)}return out})()`
	return a.RunScript(which, js)
}

// --- SQLite smoke test (no demo residue) ---

func (a *App) DBSmokeTest() (string, error) {
	if a.db == nil {
		return "", errStr("database not initialized")
	}
	const k = "__phase2_smoke__"
	if _, err := a.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, k, "ok"); err != nil {
		return "", err
	}
	var v string
	if err := a.db.QueryRow(`SELECT value FROM settings WHERE key=?`, k).Scan(&v); err != nil {
		return "", err
	}
	if _, err := a.db.Exec(`DELETE FROM settings WHERE key=?`, k); err != nil {
		return "", err
	}
	return "sqlite ok (wal, migration, write/read/delete)", nil
}

type errString2 string

func (e errString2) Error() string { return string(e) }
func errStr(s string) error        { return errString2(s) }
