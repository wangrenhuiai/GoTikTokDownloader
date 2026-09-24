package runtime_test

import (
	"testing"

	gort "gotiktokdownloader/backend/runtime"
)

func TestPathsDetectable(t *testing.T) {
	if p := gort.GetYtDlpPath(); p == "" {
		t.Fatal("yt-dlp path must be detectable (runtime/yt-dlp/yt-dlp.exe)")
	}
	if p := gort.GetFFmpegPath(); p == "" {
		t.Fatal("ffmpeg path must be detectable (runtime/ffmpeg/ffmpeg.exe)")
	}
	if v := gort.GetYtDlpVersion(); v == "" || v == "not installed" {
		t.Fatalf("yt-dlp version must resolve, got %q", v)
	}
	if v := gort.GetFFmpegVersion(); v == "" || v == "not installed" {
		t.Fatalf("ffmpeg version must resolve, got %q", v)
	}
}
