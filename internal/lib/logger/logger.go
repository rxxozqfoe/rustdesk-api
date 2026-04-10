package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	DebugMode   = "debug"
	ReleaseMode = "release"
)

type Config struct {
	Path         string
	Level        string
	ReportCaller bool
}

type Logger struct {
	sl    *slog.Logger
	level *slog.LevelVar // dynamic level holder (unused after init, kept for future)
	out   io.Writer
}

// New builds a Logger from a Config. If Path is non-empty, output is teed
// to the file and stdout. The level string accepts debug/info/warn/error
// (case-insensitive); anything else falls back to debug.
func New(c *Config) *Logger {
	var out io.Writer = os.Stdout
	if c.Path != "" {
		f, err := os.OpenFile(c.Path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			panic("open log file fail: " + err.Error())
		}
		out = io.MultiWriter(f, os.Stdout)
	}

	lvl := new(slog.LevelVar)
	lvl.Set(parseLevel(c.Level))

	handler := slog.NewTextHandler(out, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: c.ReportCaller,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) != 0 {
				return a
			}
			switch a.Key {
			case slog.TimeKey:
				// Readable timestamp that still parses cleanly as key=value.
				t := a.Value.Time()
				return slog.String(slog.TimeKey, t.Format("2006-01-02T15:04:05.000Z07:00"))
			case slog.SourceKey:
				// Shorten "/long/abs/path/pkg/file.go" to "pkg/file.go:line"
				// so log lines stay readable.
				if src, ok := a.Value.Any().(*slog.Source); ok && src != nil {
					dir, file := filepath.Split(src.File)
					pkg := filepath.Base(strings.TrimRight(dir, "/\\"))
					short := file
					if pkg != "" && pkg != "." && pkg != "/" {
						short = pkg + "/" + file
					}
					return slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", short, src.Line))
				}
			}
			return a
		},
	})

	return &Logger{
		sl:    slog.New(handler),
		level: lvl,
		out:   out,
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug", "":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

// Slog returns the underlying *slog.Logger for code that wants to log with
// native structured attributes (preferred for new code).
func (l *Logger) Slog() *slog.Logger { return l.sl }

// Writer returns an io.Writer that forwards every line written to it to the
// underlying slog logger at the given level. Useful for bridging libraries
// that expect an io.Writer (e.g. gin.DefaultErrorWriter).
func (l *Logger) Writer(level slog.Level) io.Writer {
	return &levelWriter{l: l.sl, level: level}
}

// levelWriter buffers partial writes until a newline is seen, then logs the
// accumulated line at a fixed level. It is safe for concurrent writers.
type levelWriter struct {
	l     *slog.Logger
	level slog.Level
	mu    sync.Mutex
	buf   bytes.Buffer
}

func (w *levelWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, _ := w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// no newline yet; stash the remainder back
			if line != "" {
				w.buf.WriteString(line)
			}
			break
		}
		msg := strings.TrimRight(line, "\r\n")
		if msg == "" {
			continue
		}
		w.l.Log(context.Background(), w.level, msg)
	}
	return n, nil
}

// logSkip logs a message with the correct caller at the given skip depth.
// skip=2 means: logSkip's caller's caller (i.e. the public method's caller).
func (l *Logger) logSkip(skip int, level slog.Level, msg string) {
	ctx := context.Background()
	if !l.sl.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	_ = l.sl.Handler().Handle(ctx, r)
}

func (l *Logger) log(level slog.Level, args ...any) {
	l.logSkip(4, level, fmt.Sprint(args...))
}

func (l *Logger) logf(level slog.Level, format string, args ...any) {
	l.logSkip(4, level, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(args ...any) { l.log(slog.LevelDebug, args...) }
func (l *Logger) Info(args ...any)  { l.log(slog.LevelInfo, args...) }
func (l *Logger) Warn(args ...any)  { l.log(slog.LevelWarn, args...) }
func (l *Logger) Error(args ...any) { l.log(slog.LevelError, args...) }

func (l *Logger) Debugf(format string, args ...any) { l.logf(slog.LevelDebug, format, args...) }
func (l *Logger) Infof(format string, args ...any)  { l.logf(slog.LevelInfo, format, args...) }
func (l *Logger) Warnf(format string, args ...any)  { l.logf(slog.LevelWarn, format, args...) }
func (l *Logger) Errorf(format string, args ...any) { l.logf(slog.LevelError, format, args...) }

func (l *Logger) Fatal(args ...any) {
	l.log(slog.LevelError, args...)
	os.Exit(1)
}

// Fatalf logs a formatted message at Error level then exits with status 1.
func (l *Logger) Fatalf(format string, args ...any) {
	l.logf(slog.LevelError, format, args...)
	os.Exit(1)
}

// Printf satisfies gorm.io/gorm/logger.Writer (and anything else expecting a
// classic Printf sink). The message is emitted at Info level.
func (l *Logger) Printf(format string, args ...any) {
	l.logf(slog.LevelInfo, format, args...)
}

// Println is provided for completeness; emits at Info level.
func (l *Logger) Println(args ...any) { l.log(slog.LevelInfo, args...) }
