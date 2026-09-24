package runtime

import (
	"os"
	"path/filepath"
)

func AppDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	p := filepath.Join(dir, "GoTikTokDownloader")
	_ = os.MkdirAll(p, 0755)
	return p
}

func DataDBPath() string {
	p, err := databasePath()
	if err != nil {
		return "app.db"
	}
	return p
}

func databasePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "GoTikTokDownloader", "data", "app.db"), nil
}
