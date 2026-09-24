package runtime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func exeDir() string {
	e, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(e)
}

// candidateRoots returns directories to search for bundled deps:
// exe dir, exe dir/runtime, cwd, cwd/runtime.
func candidateRoots() []string {
	cwd, _ := os.Getwd()
	roots := []string{exeDir(), filepath.Join(exeDir(), "runtime"), cwd, filepath.Join(cwd, "runtime")}
	// go test runs with cwd=<pkg>; walk up to repo root.
	d := cwd
	for i := 0; i < 4; i++ {
		d = filepath.Dir(d)
		roots = append(roots, d, filepath.Join(d, "runtime"))
	}
	return roots
}

func findFile(names ...string) string {
	for _, r := range candidateRoots() {
		for _, n := range names {
			p := filepath.Join(r, n)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}

func GetPythonPath() string {
	if p := findFile(filepath.Join("python", "python.exe"), "python.exe"); p != "" {
		return p
	}
	if p, err := exec.LookPath("python"); err == nil {
		return p
	}
	return ""
}

func GetYtDlpPath() string {
	return findFile(filepath.Join("yt-dlp", "yt-dlp.pyz"), "yt-dlp.pyz", filepath.Join("yt-dlp", "yt-dlp.exe"), "yt-dlp.exe")
}

func GetFFmpegPath() string {
	if p := findFile(filepath.Join("ffmpeg", "ffmpeg.exe"), "ffmpeg.exe"); p != "" {
		return p
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return ""
}

func runVersion(bin string, args ...string) string {
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return "not found: " + err.Error()
	}
	line := strings.TrimSpace(string(out))
	if i := strings.Index(line, "\n"); i >= 0 {
		line = line[:i]
	}
	return line
}

func CheckYtDlp() (string, error) {
	ydlp := GetYtDlpPath()
	if ydlp == "" {
		return "", errf("yt-dlp not found under runtime/yt-dlp/")
	}
	// Standalone exe (no python needed)
	if strings.HasSuffix(strings.ToLower(ydlp), ".exe") {
		out, err := exec.Command(ydlp, "--version").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	py := GetPythonPath()
	if py == "" {
		return "", errf("bundled python not found")
	}
	out, err := exec.Command(py, ydlp, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GetYtDlpVersion() string {
	v, err := CheckYtDlp()
	if err != nil {
		return "not installed (" + err.Error() + ")"
	}
	return v
}

func FindFFmpeg() string { return GetFFmpegPath() }

func CheckFFmpeg() (string, error) {
	p := GetFFmpegPath()
	if p == "" {
		return "", errf("ffmpeg not found (bundled runtime/ffmpeg/ffmpeg.exe or PATH)")
	}
	return runVersion(p, "-version"), nil
}

func GetFFmpegVersion() string {
	v, err := CheckFFmpeg()
	if err != nil {
		return "not installed"
	}
	return v
}

type errString string

func (e errString) Error() string { return string(e) }
func errf(s string) error         { return errString(s) }
