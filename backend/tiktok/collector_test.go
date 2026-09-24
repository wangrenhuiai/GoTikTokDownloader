package tiktok_test

import (
	"testing"

	"gotiktokdownloader/backend/tiktok"
)

func TestParseVideoURL(t *testing.T) {
	canon, vid, author, ok := tiktok.ParseVideoURL("https://www.tiktok.com/@abc/video/123456789?lang=en&_r=1")
	if !ok || vid != "123456789" || author != "abc" || canon != "https://www.tiktok.com/@abc/video/123456789" {
		t.Fatalf("got %q %q %q %v", canon, vid, author, ok)
	}
	// Query params must not affect identity.
	c2, _, _, ok := tiktok.ParseVideoURL("https://www.tiktok.com/@abc/video/123456789?foo=bar")
	if !ok || c2 != canon {
		t.Fatalf("canonical mismatch: %q", c2)
	}
	for _, bad := range []string{"", "https://www.youtube.com/watch?v=x", "https://www.tiktok.com/@abc", "https://www.tiktok.com/search?q=cooking"} {
		if _, _, _, ok := tiktok.ParseVideoURL(bad); ok {
			t.Fatalf("should reject %q", bad)
		}
	}
}

func TestMergeDedup(t *testing.T) {
	acc := map[string]tiktok.VideoCandidate{}
	mk := func(id string) tiktok.VideoCandidate { return tiktok.VideoCandidate{VideoID: id, URL: "https://www.tiktok.com/@a/video/" + id} }
	added, dups := tiktok.MergeDedup(acc, []tiktok.VideoCandidate{mk("1"), mk("2")})
	if added != 2 || dups != 0 {
		t.Fatalf("got %d %d", added, dups)
	}
	added, dups = tiktok.MergeDedup(acc, []tiktok.VideoCandidate{mk("2"), mk("3")})
	if added != 1 || dups != 1 || len(acc) != 3 {
		t.Fatalf("got %d %d len=%d", added, dups, len(acc))
	}
}

func TestParseScanResult(t *testing.T) {
	js := `[{"url":"https://www.tiktok.com/@abc/video/111","author":"@abc","title":"t1"},{"url":"https://www.tiktok.com/@xyz/video/222?lang=en","author":"","title":""},{"url":"https://www.tiktok.com/@abc/video/111","author":"@abc","title":"dup"},{"url":"https://www.tiktok.com/search?q=x","author":"","title":""}]`
	cands, err := tiktok.ParseScanResult(js, "cooking")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("expected 2 deduped, got %d", len(cands))
	}
	if cands[0].VideoID != "111" || cands[0].AuthorName != "@abc" || cands[0].Keyword != "cooking" {
		t.Fatalf("bad candidate: %+v", cands[0])
	}
}
