package logger

import (
	"context"
)

// contextKey is the key type for storing values in context.
type contextKey int

const (
	// loggerKey is the key for the logger in the context.
	loggerKey contextKey = iota
	// fieldsKey is the key for the fields in the context.
	fieldsKey
)

// FromContext returns the logger from the context.
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return global
	}

	// 获取上下文中的日志器
	var logger Logger = global
	if ctxLogger, ok := ctx.Value(loggerKey).(Logger); ok {
		logger = ctxLogger
	}

	// 获取上下文中的字段
	if fields, ok := ctx.Value(fieldsKey).([]Field); ok && len(fields) > 0 {
		logger = logger.WithFields(fields...)
	}

	// 获取上下文中的跟踪信息
	if traceInfo, ok := ctx.Value(traceKey).(*TraceInfo); ok {
		logger = logger.WithFields(traceInfo.ToFields()...)
	}

	return logger
}

// WithLogger returns a new context with the given logger.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// WithContextFields returns a new context with the given fields.
func WithContextFields(ctx context.Context, fields ...Field) context.Context {
	if existingFields, ok := ctx.Value(fieldsKey).([]Field); ok {
		fields = append(existingFields, fields...)
	}
	return context.WithValue(ctx, fieldsKey, fields)
}

// DebugContext logs a debug message with context.
func DebugContext(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Debug(args...)
}

// DebugfContext logs a formatted debug message with context.
func DebugfContext(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Debugf(format, args...)
}

// InfoContext logs an info message with context.
func InfoContext(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Info(args...)
}

// InfofContext logs a formatted info message with context.
func InfofContext(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Infof(format, args...)
}

// WarnContext logs a warning message with context.
func WarnContext(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Warn(args...)
}

// WarnfContext logs a formatted warning message with context.
func WarnfContext(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Warnf(format, args...)
}

// ErrorContext logs an error message with context.
func ErrorContext(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Error(args...)
}

// ErrorfContext logs a formatted error message with context.
func ErrorfContext(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Errorf(format, args...)
}

// FatalContext logs a fatal message with context and exits.
func FatalContext(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Fatal(args...)
}

// FatalfContext logs a formatted fatal message with context and exits.
func FatalfContext(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Fatalf(format, args...)
}
