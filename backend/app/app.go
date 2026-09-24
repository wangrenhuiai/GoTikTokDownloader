package app

import (
	"context"
	"database/sql"
	"runtime"

	"gotiktokdownloader/backend/config"
	"gotiktokdownloader/backend/database"
	gort "gotiktokdownloader/backend/runtime"
	"gotiktokdownloader/backend/logging"
)

type AppInfo struct {
	AppName   string `json:"appName"`
	Version   string `json:"version"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

type RuntimeStatus struct {
	PythonPath   string `json:"pythonPath"`
	YtDlpPath    string `json:"ytDlpPath"`
	YtDlpVersion string `json:"ytDlpVersion"`
	FFmpegPath   string `json:"ffmpegPath"`
	FFmpegOK     bool   `json:"ffmpegOK"`
	DBPath       string `json:"dbPath"`
	DBOK         bool   `json:"dbOK"`
}

type App struct {
	ctx context.Context
	cfg config.Config
	db  *sql.DB
	log *logging.Logger
	emit func(event string, data any)
}

func New(log *logging.Logger, emit func(event string, data any)) *App {
	cfg, _ := config.Load()
	return &App{cfg: cfg, log: log, emit: emit}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	dbPath, _ := database.DefaultPath()
	db, err := database.Open(dbPath)
	if err != nil {
		a.log.Errorf("open db: %v", err)
	} else {
		a.db = db
	}
	a.Emit("system:status", "backend ready")
	a.Emit("system:log", "GoTikTokDownloader backend started")
}

func (a *App) Shutdown(ctx context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

func (a *App) Emit(event string, data any) {
	if a.emit != nil {
		a.emit(event, data)
	}
}

func (a *App) GetAppInfo() AppInfo {
	return AppInfo{AppName: "GoTikTokDownloader", Version: "0.1.0-phase1", GoVersion: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH}
}

func (a *App) GetRuntimeStatus() RuntimeStatus {
	ffPath := gort.GetFFmpegPath()
	_, ffErr := gort.CheckFFmpeg()
	dbPath, _ := database.DefaultPath()
	return RuntimeStatus{
		PythonPath:   gort.GetPythonPath(),
		YtDlpPath:    gort.GetYtDlpPath(),
		YtDlpVersion: gort.GetYtDlpVersion(),
		FFmpegPath:   ffPath,
		FFmpegOK:     ffErr == nil,
		DBPath:       dbPath,
		DBOK:         a.db != nil,
	}
}

func (a *App) GetSettings() config.Config { return a.cfg }

func (a *App) SetSettings(c config.Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := config.Save(c); err != nil {
		return err
	}
	a.cfg = c
	a.Emit("system:status", "settings saved")
	return nil
}

// PingEvent proves Go -> Svelte event chain in Phase 1.
func (a *App) PingEvent(msg string) string {
	a.Emit("system:log", "pong: "+msg)
	a.Emit("task:progress", map[string]any{"id": "phase1-demo", "progress": 42.5})
	return "pong: " + msg
}
