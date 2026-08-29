// Package logger provides a logger interface and its slog-based implementation.
package logger

import "log/slog"

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

type slogWrapper struct {
	l *slog.Logger
}

func NewSlog(l *slog.Logger) Logger {
	return &slogWrapper{l: l}
}

func (s *slogWrapper) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogWrapper) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogWrapper) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogWrapper) Error(msg string, args ...any) { s.l.Error(msg, args...) }
func (s *slogWrapper) With(args ...any) Logger       { return &slogWrapper{l: s.l.With(args...)} }

// NopLogger is a no-op logger for tests.
type NopLogger struct{}

func NewNoop() Logger {
	return &NopLogger{}
}

func (n *NopLogger) Debug(_ string, _ ...any) {}
func (n *NopLogger) Info(_ string, _ ...any)  {}
func (n *NopLogger) Warn(_ string, _ ...any)  {}
func (n *NopLogger) Error(_ string, _ ...any) {}
func (n *NopLogger) With(_ ...any) Logger     { return n }
