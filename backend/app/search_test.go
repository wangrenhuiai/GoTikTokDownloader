package app

import "testing"

func TestIsTikTokURL(t *testing.T) {
	for _, u := range []string{
		"https://www.tiktok.com/@a/video/123",
		"https://www.tiktok.com/search?q=x",
	} {
		if !isTikTokURL(u) {
			t.Fatalf("should allow %q", u)
		}
	}
	for _, u := range []string{"", "https://evil.com/@a/video/123", "https://tiktok.com.evil.com/x", "https://v16.tiktokcdn.com/abc.mp4", "not a url %%"} {
		if isTikTokURL(u) {
			t.Fatalf("should reject %q", u)
		}
	}
}
