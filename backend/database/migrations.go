package database

import "database/sql"

func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS authors (id TEXT PRIMARY KEY, nickname TEXT DEFAULT '', updated_at INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS videos (
			id TEXT PRIMARY KEY, author_id TEXT DEFAULT '', url TEXT UNIQUE NOT NULL,
			title TEXT DEFAULT '', duration REAL DEFAULT 0,
			play_count INTEGER DEFAULT 0, like_count INTEGER DEFAULT 0,
			comment_count INTEGER DEFAULT 0, share_count INTEGER DEFAULT 0,
			published_at INTEGER DEFAULT 0, file_path TEXT DEFAULT '',
			file_size INTEGER DEFAULT 0, status TEXT DEFAULT 'new',
			error TEXT DEFAULT '', updated_at INTEGER NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_author ON videos(author_id)`,
		`CREATE INDEX IF NOT EXISTS idx_videos_status ON videos(status)`,
		`CREATE TABLE IF NOT EXISTS download_tasks (
			id TEXT PRIMARY KEY, video_id TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'queued',
			progress REAL DEFAULT 0, speed REAL DEFAULT 0, eta INTEGER DEFAULT 0,
			retry_count INTEGER DEFAULT 0, created_at INTEGER NOT NULL, finished_at INTEGER DEFAULT 0)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON download_tasks(status)`,
		`CREATE TABLE IF NOT EXISTS download_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id TEXT NOT NULL,
			started_at INTEGER NOT NULL, error TEXT DEFAULT '')`,
		`CREATE INDEX IF NOT EXISTS idx_attempts_task ON download_attempts(task_id)`,
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}
