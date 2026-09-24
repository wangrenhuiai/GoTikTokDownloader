package database_test

import (
	"path/filepath"
	"testing"

	"gotiktokdownloader/backend/database"
)

func TestOpenMigrateWALRoundtrip(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("expected wal, got %q", mode)
	}
	for _, tbl := range []string{"settings", "authors", "videos", "download_tasks", "download_attempts"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&n); err != nil || n != 1 {
			t.Fatalf("table %s missing: %v", tbl, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO settings(key,value) VALUES('phase2-smoke','ok') ON CONFLICT(key) DO UPDATE SET value='ok'`); err != nil {
		t.Fatal(err)
	}
	var v string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key='phase2-smoke'`).Scan(&v); err != nil || v != "ok" {
		t.Fatalf("settings roundtrip failed: %v %q", err, v)
	}
}
