package tiktok

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ValidateURL accepts TikTok video/user/share URLs.
func ValidateURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return errors.New("invalid url")
	}
	h := strings.ToLower(u.Host)
	for _, ok := range []string{"tiktok.com", "vm.tiktok.com", "vt.tiktok.com", "vm.tiktok.com"} {
		if strings.Contains(h, ok) {
			return nil
		}
	}
	return errors.New("not a tiktok url: " + h)
}

// LoginService tracks login session via persisted browser profile.
// Never logs cookie values.
type LoginService struct {
	profileDir string
}

func NewLoginService() *LoginService {
	dir, _ := os.UserConfigDir()
	return &LoginService{profileDir: filepath.Join(dir, "GoTikTokDownloader", "data", "browser", "tiktok")}
}

func (l *LoginService) LoginURL() string { return "https://www.tiktok.com/login" }

// SessionExists reports whether a persisted profile exists (login source reusable).
func (l *LoginService) SessionExists() bool {
	st, err := os.Stat(l.profileDir)
	return err == nil && st.IsDir()
}

// MarkLoggedIn writes a non-sensitive login marker (no cookies).
func (l *LoginService) MarkLoggedIn() error {
	if err := os.MkdirAll(l.profileDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(l.profileDir, "login.marker"), []byte("logged-in"), 0644)
}

// CheckLoginState returns "logged-in" | "not-logged-in".
func (l *LoginService) CheckLoginState() string {
	if _, err := os.Stat(filepath.Join(l.profileDir, "login.marker")); err == nil {
		return "logged-in"
	}
	return "not-logged-in"
}

func (l *LoginService) ProfileDir() string { return l.profileDir }
