package sentry

import (
	"context"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/dugble/dugble/server/internal/platform/config"
)

// Init initializes Sentry error monitoring when a DSN is configured.
func Init(configuration config.SentryConfig, environment string) error {
	configuration.DSN = strings.TrimSpace(configuration.DSN)
	if configuration.DSN == "" {
		return nil
	}
	return sentry.Init(sentry.ClientOptions{
		Dsn:              configuration.DSN,
		Environment:      strings.TrimSpace(environment),
		Release:          strings.TrimSpace(configuration.Release),
		SampleRate:       configuration.ErrorSampleRate,
		Debug:            configuration.Debug,
		AttachStacktrace: true,
		EnableTracing:    false,
	})
}

func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}

// Debug emits a Sentry debug log without trace correlation.
func Debug(message string, args ...any) {
	DebugContext(context.Background(), message, args...)
}

// DebugContext emits a Sentry debug log correlated with ctx.
func DebugContext(ctx context.Context, message string, args ...any) {
	newLogger(ctx).Debug().Emit(append([]any{message}, args...)...)
}

// Info emits a Sentry info log without trace correlation.
func Info(message string, args ...any) {
	InfoContext(context.Background(), message, args...)
}

// InfoContext emits a Sentry info log correlated with ctx.
func InfoContext(ctx context.Context, message string, args ...any) {
	newLogger(ctx).Info().Emit(append([]any{message}, args...)...)
}

// Warn emits a Sentry warning log without trace correlation.
func Warn(message string, args ...any) {
	WarnContext(context.Background(), message, args...)
}

// WarnContext emits a Sentry warning log correlated with ctx.
func WarnContext(ctx context.Context, message string, args ...any) {
	newLogger(ctx).Warn().Emit(append([]any{message}, args...)...)
}

// Error emits a Sentry error log without trace correlation.
func Error(message string, args ...any) {
	ErrorContext(context.Background(), message, args...)
}

// ErrorContext emits a Sentry error log correlated with ctx.
func ErrorContext(ctx context.Context, message string, args ...any) {
	newLogger(ctx).Error().Emit(append([]any{message}, args...)...)
}

func newLogger(ctx context.Context) sentry.Logger {
	if ctx == nil {
		ctx = context.Background()
	}
	return sentry.NewLogger(ctx)
}
