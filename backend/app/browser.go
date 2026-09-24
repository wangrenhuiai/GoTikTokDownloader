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

// --- Single TikTok browser ---

// OpenTikTokBrowser starts the one and only TikTok WebView2 instance.
func (a *App) OpenTikTokBrowser() (BrowserState, error) {
	a.ensureBrowsers()
	a.log.Infof("TikTokBrowserHost starting (single embedded WebView2)")
	if err := a.browsers.TikTok.Start(); err != nil {
		if be, ok := err.(*browser.BrowserError); ok && be.Code == browser.ErrAlreadyRunning {
			return snap(a.browsers.TikTok), nil
		}
		a.log.Errorf("TikTokBrowserHost start failed: %v", err)
		return snap(a.browsers.TikTok), err
	}
	a.Emit(EvtSystemLog, "TikTokBrowserHost started")
	a.Emit(EvtSystemStatus, "tiktok browser open")
	return snap(a.browsers.TikTok), nil
}

func (a *App) CloseTikTokBrowser() error {
	a.ensureBrowsers()
	a.log.Infof("TikTokBrowserHost stopped")
	return a.browsers.TikTok.Stop()
}

func (a *App) GetTikTokBrowserState() BrowserState {
	a.ensureBrowsers()
	return snap(a.browsers.TikTok)
}

// OpenTikTokSearch reuses the single browser to show a keyword search page.
func (a *App) OpenTikTokSearch(keyword string) (BrowserState, error) {
	a.ensureBrowsers()
	st, err := a.OpenTikTokBrowser()
	if err != nil {
		return st, err
	}
	if keyword != "" {
		if err := a.browsers.TikTok.Navigate("https://www.tiktok.com/search?q=" + keyword); err != nil {
			a.log.Errorf("TikTokBrowserHost search navigate failed: %v", err)
			return snap(a.browsers.TikTok), err
		}
		a.Emit(EvtSystemLog, "TikTokBrowserHost navigated to search")
	}
	return snap(a.browsers.TikTok), nil
}

// LegacyProfiles reports pre-convergence profiles (tiktok-main/tiktok-search).
func (a *App) LegacyProfiles() map[string]bool {
	return browser.LegacyProfiles()
}

// --- Script / page ---

// RunScript executes JS in the single embedded browser.
func (a *App) RunScript(script string) (string, error) {
	a.ensureBrowsers()
	res, err := a.browsers.TikTok.ExecuteScript(script)
	if err != nil {
		a.log.Errorf("RunScript failed: %v", err)
		return "", err
	}
	return res, nil
}

// PageInfo bundles title+url+body length via three real script round-trips.
func (a *App) PageInfo() (map[string]string, error) {
	a.ensureBrowsers()
	h := a.browsers.TikTok
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

func (a *App) WaitBrowserReady(timeoutMs int) error {
	a.ensureBrowsers()
	return a.browsers.TikTok.WaitReady(timeoutMs)
}

// WaitPageReady blocks until page content is really rendered (blank-page guard).
func (a *App) WaitPageReady(timeoutMs int) (string, error) {
	a.ensureBrowsers()
	if w, ok := a.browsers.TikTok.(interface {
		WaitPageReady(int) (string, error)
	}); ok {
		return w.WaitPageReady(timeoutMs)
	}
	return "", errStr("WaitPageReady unsupported on this platform")
}

// --- Login ---

func (a *App) GetLoginURL() string {
	a.ensureBrowsers()
	return a.login.LoginURL()
}

func (a *App) OpenLoginWindow() (BrowserState, error) {
	a.ensureBrowsers()
	a.log.Infof("TikTok login page opening in single embedded browser")
	st, err := a.OpenTikTokBrowser()
	if err != nil {
		return st, err
	}
	if err := a.browsers.TikTok.Navigate(a.login.LoginURL()); err != nil {
		return snap(a.browsers.TikTok), err
	}
	a.Emit(EvtSystemLog, "TikTok navigation: login page (embedded)")
	return snap(a.browsers.TikTok), nil
}

// GetLoginState: Layer1 local profile + Layer2 live session probe.
func (a *App) GetLoginState() string {
	a.ensureBrowsers()
	if !a.login.SessionExists() {
		return "not-logged-in"
	}
	if a.browsers != nil && a.browsers.TikTok.IsReady() {
		if probe, err := a.browsers.TikTok.LoginProbe(); err == nil && probe != "unknown" {
			return probe
		}
	}
	if a.login.CheckLoginState() == "logged-in" {
		return "logged-in (unverified)"
	}
	return "not-logged-in"
}

// ProbeLoginState forces a live WebView2 session check.
func (a *App) ProbeLoginState() string {
	a.ensureBrowsers()
	if a.browsers == nil || !a.browsers.TikTok.IsReady() {
		return "unknown (browser not open)"
	}
	probe, err := a.browsers.TikTok.LoginProbe()
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
		return "Phase 3A-R: single browser ready; auto-crawl not implemented", err
	}
	return "ok", nil
}

// --- Diagnostics (additive only) ---

func (a *App) DiagRuntime() map[string]any {
	a.ensureBrowsers()
	h := a.browsers.TikTok
	if s, ok := h.(interface{ DiagSnapshot() map[string]any }); ok {
		return s.DiagSnapshot()
	}
	return map[string]any{"name": h.Name(), "note": "no native snapshot on this platform"}
}

// DiagDOM runs the exact DOM snapshot script from the diag spec.
func (a *App) DiagDOM() (string, error) {
	const js = `({href: location.href, title: document.title, readyState: document.readyState, bodyExists: !!document.body, bodyLength: document.body ? document.body.innerHTML.length : -1, htmlLength: document.documentElement ? document.documentElement.outerHTML.length : -1, textLength: document.body ? document.body.innerText.length : -1})`
	return a.RunScript(js)
}

// DiagUA returns userAgent + navigator details (record only, never modified).
func (a *App) DiagUA() (string, error) {
	const js = `({ua: navigator.userAgent, language: navigator.language, platform: navigator.platform, webdriver: navigator.webdriver})`
	return a.RunScript(js)
}

// DiagNetwork summarizes resource response statuses via Performance API.
func (a *App) DiagNetwork() (string, error) {
	const js = `(function(){var out={nav:null,hist:{},failSample:[]};try{var n=performance.getEntriesByType('navigation')[0];if(n){out.nav={url:n.name,redirects:n.redirectCount,status:n.responseStatus||null,domContent:n.domContentLoadedEventEnd-n.startTime,load:n.loadEventEnd-n.startTime}}}catch(e){out.nav='ERR:'+e}try{var rs=performance.getEntriesByType('resource');for(var i=0;i<rs.length;i++){var s=rs[i].responseStatus||0;var k=String(s);out.hist[k]=(out.hist[k]||0)+1;if((s>=400||s===0)&&out.failSample.length<5){out.failSample.push({status:s,url:String(rs[i].name).slice(0,160)})}}}catch(e){out.histErr=String(e)}return out})()`
	return a.RunScript(js)
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
