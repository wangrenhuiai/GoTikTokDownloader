package tiktok_test

import (
	"testing"

	"gotiktokdownloader/backend/tiktok"
)

func TestValidateURL(t *testing.T) {
	for _, u := range []string{
		"https://www.tiktok.com/@user/video/123",
		"https://vm.tiktok.com/abc/",
		"https://www.tiktok.com/search?q=football",
	} {
		if err := tiktok.ValidateURL(u); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}
	if err := tiktok.ValidateURL("https://www.youtube.com/watch?v=x"); err == nil {
		t.Fatal("non-tiktok must fail")
	}
}

func TestSearchReserved(t *testing.T) {
	s := tiktok.NewSearchService()
	if err := s.Search("football"); err == nil || err.Error() != "not implemented (Phase 2 reservation)" {
		t.Fatalf("expected reservation error, got %v", err)
	}
}
