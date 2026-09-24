package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type DownloadConfig struct {
	MaxConcurrency    int    `json:"maxConcurrency"`
	RetryCount        int    `json:"retryCount"`
	DownloadDirectory string `json:"downloadDirectory"`
}

type NetworkConfig struct {
	Proxy     string `json:"proxy"`
	CookieFile string `json:"cookieFile"`
	UserAgent string `json:"userAgent"`
}

type RuntimeConfig struct {
	YtDlpPath   string `json:"ytDlpPath"`
	FFmpegPath  string `json:"ffmpegPath"`
	PythonPath  string `json:"pythonPath"`
}

type Config struct {
	Download DownloadConfig `json:"download"`
	Network  NetworkConfig  `json:"network"`
	Runtime  RuntimeConfig  `json:"runtime"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	dl := filepath.Join(home, "Downloads", "TikTokDownloader")
	return Config{
		Download: DownloadConfig{MaxConcurrency: 4, RetryCount: 3, DownloadDirectory: dl},
		Network:  NetworkConfig{},
		Runtime:  RuntimeConfig{},
	}
}

func (c *Config) Validate() error {
	if c.Download.MaxConcurrency < 1 || c.Download.MaxConcurrency > 100 {
		return fmt.Errorf("maxConcurrency must be 1..100, got %d", c.Download.MaxConcurrency)
	}
	if c.Download.RetryCount < 0 || c.Download.RetryCount > 10 {
		return fmt.Errorf("retryCount must be 0..10, got %d", c.Download.RetryCount)
	}
	if c.Download.DownloadDirectory == "" {
		return fmt.Errorf("downloadDirectory must not be empty")
	}
	return nil
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "GoTikTokDownloader", "config.json"), nil
}

func Load() (Config, error) {
	cfg := Default()
	p, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}

func Save(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(p, data, 0644)
}
