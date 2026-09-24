package browser_test

import (
	"strings"
	"testing"

	"gotiktokdownloader/backend/browser"
)

func TestManagerProfilesIsolated(t *testing.T) {
	m := browser.NewManager()
	if m.Main.ProfileDir() == m.Search.ProfileDir() {
		t.Fatal("profile dirs must differ")
	}
	if !strings.Contains(m.Main.ProfileDir(), "tiktok-main") {
		t.Fatalf("main profile: %q", m.Main.ProfileDir())
	}
	if !strings.Contains(m.Search.ProfileDir(), "tiktok-search") {
		t.Fatalf("search profile: %q", m.Search.ProfileDir())
	}
}

func TestErrorCodes(t *testing.T) {
	err := &browser.BrowserError{Code: browser.ErrBadURL, Message: "empty url"}
	if err.Error() == "" || !strings.Contains(err.Error(), "BAD_URL") {
		t.Fatalf("bad error string: %q", err.Error())
	}
	withCause := &browser.BrowserError{Code: browser.ErrNavFailed, Message: "x", Cause: "y"}
	if !strings.Contains(withCause.Error(), "y") {
		t.Fatalf("cause missing: %q", withCause.Error())
	}
}

func TestInterfaceConformance(t *testing.T) {
	var _ browser.BrowserHost = browser.NewManager().Main
	var _ browser.BrowserHost = browser.NewManager().Search
}
