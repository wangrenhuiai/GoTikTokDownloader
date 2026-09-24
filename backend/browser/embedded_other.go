//go:build !windows

package browser

import "os"
import "path/filepath"

type stubHost struct{ name, profile string }

func appData() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	return filepath.Join(dir, "GoTikTokDownloader")
}

func newEmbeddedHost(name, title, homepage string) BrowserHost { //nolint:revive
	return &stubHost{name: name, profile: filepath.Join(appData(), "data", "browser", name)}
}

func unsupported() error { return &BrowserError{Code: ErrUnsupported, Message: "embedded browser requires Windows WebView2"} }

func (s *stubHost) Name() string                         { return s.name }
func (s *stubHost) Start() error                         { return unsupported() }
func (s *stubHost) Stop() error                          { return nil }
func (s *stubHost) Navigate(string) error                { return unsupported() }
func (s *stubHost) CurrentURL() string                   { return "" }
func (s *stubHost) ExecuteScript(string) (string, error) { return "", unsupported() }
func (s *stubHost) IsReady() bool                        { return false }
func (s *stubHost) WaitReady(int) error                  { return unsupported() }
func (s *stubHost) SetTitle(string) error                { return nil }
func (s *stubHost) Show() error                          { return unsupported() }
func (s *stubHost) Hide() error                          { return unsupported() }
func (s *stubHost) ProfileDir() string                   { return s.profile }
func (s *stubHost) LoginProbe() (string, error)          { return "unknown", nil }
