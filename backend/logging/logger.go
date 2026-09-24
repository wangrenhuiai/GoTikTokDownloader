package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	l     *log.Logger
	level Level
}

func New(level Level) *Logger {
	var w io.Writer = os.Stdout
	if dir, err := os.UserConfigDir(); err == nil {
		logFile := filepath.Join(dir, "GoTikTokDownloader", "logs", "app.log")
		_ = os.MkdirAll(filepath.Dir(logFile), 0755)
		if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			w = io.MultiWriter(os.Stdout, f)
		}
	}
	return &Logger{l: log.New(w, "", log.LstdFlags|log.Lshortfile), level: level}
}

func (l *Logger) logf(lv Level, format string, args ...any) {
	if lv < l.level {
		return
	}
	prefix := map[Level]string{DEBUG: "[DEBUG] ", INFO: "[INFO] ", WARN: "[WARN] ", ERROR: "[ERROR] "}[lv]
	l.l.Printf(prefix+format, args...)
}

func (l *Logger) Debugf(f string, a ...any) { l.logf(DEBUG, f, a...) }
func (l *Logger) Infof(f string, a ...any)  { l.logf(INFO, f, a...) }
func (l *Logger) Warnf(f string, a ...any)  { l.logf(WARN, f, a...) }
func (l *Logger) Errorf(f string, a ...any) { l.logf(ERROR, f, a...) }
