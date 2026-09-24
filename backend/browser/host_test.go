package browser_test

import (
	"testing"

	"gotiktokdownloader/backend/browser"
)

func TestIndependentLifecycle(t *testing.T) {
	m := browser.NewManager()
	if err := m.Main.Start(); err != nil {
		t.Fatal(err)
	}
	if err := m.Search.Start(); err != nil {
		t.Fatal(err)
	}
	if !m.Main.IsReady() || !m.Search.IsReady() {
		t.Fatal("both hosts should be ready")
	}
	if m.Main.CurrentURL() == "" || m.Search.CurrentURL() == "" {
		t.Fatal("urls should be set")
	}
	// Closing search must not affect main.
	if err := m.Search.Stop(); err != nil {
		t.Fatal(err)
	}
	if m.Search.IsReady() {
		t.Fatal("search should be stopped")
	}
	if !m.Main.IsReady() {
		t.Fatal("main must be unaffected by search stop")
	}
	// Reopen search, then navigate to keyword page.
	if err := m.Search.Start(); err != nil {
		t.Fatal(err)
	}
	if err := m.Search.Navigate("https://www.tiktok.com/search?q=football"); err != nil {
		t.Fatal(err)
	}
	if got := m.Search.CurrentURL(); got != "https://www.tiktok.com/search?q=football" {
		t.Fatalf("unexpected staged url %q", got)
	}
	if err := m.Search.Stop(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Search.ExecuteScript("1+1"); err == nil {
		t.Fatal("stopped host must reject script")
	}
	if err := m.Main.Stop(); err != nil {
		t.Fatal(err)
	}
	if m.Main.ProfileDir() == m.Search.ProfileDir() {
		t.Fatal("profile dirs must differ")
	}
}

func TestDoubleStartRejected(t *testing.T) {
	m := browser.NewManager()
	if err := m.Main.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Main.Stop()
	if err := m.Main.Start(); err == nil {
		t.Fatal("double start should fail")
	}
}
