package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"couple-mini/backend/configs"
)

var (
	mu           sync.RWMutex
	appLogger    = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	accessLogger = appLogger
	closers      []io.Closer
)

func Init(cfg configs.LogConfig) error {
	mu.Lock()
	defer mu.Unlock()

	closeLocked()

	appWriter, appCloser, err := buildWriter(cfg.Directory, cfg.AppFile, cfg.AlsoStdout)
	if err != nil {
		return err
	}
	accessWriter, accessCloser, err := buildWriter(cfg.Directory, cfg.AccessFile, cfg.AlsoStdout)
	if err != nil {
		if appCloser != nil {
			_ = appCloser.Close()
		}
		return err
	}

	if appCloser != nil {
		closers = append(closers, appCloser)
	}
	if accessCloser != nil {
		closers = append(closers, accessCloser)
	}

	appLogger = slog.New(newHandler(appWriter, cfg, parseLevel(cfg.Level))).With("stream", "app")
	accessLogger = slog.New(newHandler(accessWriter, cfg, slog.LevelInfo)).With("stream", "access")
	return nil
}

func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return appLogger
}

func Access() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return accessLogger
}

func Close() error {
	mu.Lock()
	defer mu.Unlock()
	closeLocked()
	return nil
}

func closeLocked() {
	for _, closer := range closers {
		_ = closer.Close()
	}
	closers = nil
}

func buildWriter(dir, file string, alsoStdout bool) (io.Writer, io.Closer, error) {
	writers := make([]io.Writer, 0, 2)
	var closer io.Closer

	if file != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, err
		}
		f, err := os.OpenFile(filepath.Join(dir, file), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		writers = append(writers, f)
		closer = f
	}
	if alsoStdout || len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}
	if len(writers) == 1 {
		return writers[0], closer, nil
	}
	return io.MultiWriter(writers...), closer, nil
}

func newHandler(writer io.Writer, cfg configs.LogConfig, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.IncludeSource,
	}
	if strings.EqualFold(cfg.Format, "text") {
		return slog.NewTextHandler(writer, opts)
	}
	return slog.NewJSONHandler(writer, opts)
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
