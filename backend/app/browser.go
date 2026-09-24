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
	a.log.Infof("MainBrowserHost started")
	if err := a.browsers.Main.Start(); err != nil {
		if be, ok := err.(*browser.BrowserError); ok && be.Code == "ALREADY_RUNNING" {
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
	a.log.Infof("SearchBrowserHost started (keyword=%q)", keyword)
	if err := a.browsers.Search.Start(); err != nil {
		if be, ok := err.(*browser.BrowserError); ok && be.Code == "ALREADY_RUNNING" {
			return snap(a.browsers.Search), nil
		}
		a.log.Errorf("SearchBrowserHost start failed: %v", err)
		return snap(a.browsers.Search), err
	}
	if keyword != "" {
		_ = a.browsers.Search.Navigate("https://www.tiktok.com/search?q=" + keyword)
		a.Emit(EvtSystemLog, "Phase 2: search browser infrastructure started")
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

// --- Login ---

func (a *App) GetLoginURL() string {
	a.ensureBrowsers()
	return a.login.LoginURL()
}

func (a *App) OpenLoginWindow() (BrowserState, error) {
	a.ensureBrowsers()
	a.log.Infof("TikTok login browser starting")
	st, err := a.OpenMainBrowser()
	if err != nil {
		return st, err
	}
	if err := a.browsers.Main.Navigate(a.login.LoginURL()); err != nil {
		if be, ok := err.(*browser.BrowserError); ok && be.Code == "NOT_RUNNING" {
			return snap(a.browsers.Main), nil
		}
		return snap(a.browsers.Main), err
	}
	a.Emit(EvtSystemLog, "TikTok navigation: login page")
	return snap(a.browsers.Main), nil
}

func (a *App) GetLoginState() string {
	a.ensureBrowsers()
	return a.login.CheckLoginState()
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
		return "Phase 2: search browser infrastructure ready; auto-crawl not implemented", err
	}
	return "ok", nil
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
