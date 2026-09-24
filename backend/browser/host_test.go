package browser_test

import (
	"strings"
	"testing"

	"gotiktokdownloader/backend/browser"
)

func TestSingleHostProfile(t *testing.T) {
	m := browser.NewManager()
	if m.TikTok == nil {
		t.Fatal("single TikTok host must exist")
	}
	if !strings.HasSuffix(m.TikTok.ProfileDir(), "tiktok") {
		t.Fatalf("canonical profile must be .../tiktok, got %q", m.TikTok.ProfileDir())
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
	var _ browser.BrowserHost = browser.NewManager().TikTok
}

func TestLegacyProfilesCallable(t *testing.T) {
	_ = browser.LegacyProfiles()
}
