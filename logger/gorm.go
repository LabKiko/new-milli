package logger

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger is an adapter for GORM logger.
type GormLogger struct {
	logger                    Logger
	slowThreshold             time.Duration
	logLevel                  gormlogger.LogLevel
	ignoreRecordNotFoundError bool
}

// NewGormLogger creates a new GORM logger adapter.
func NewGormLogger(logger Logger) *GormLogger {
	return &GormLogger{
		logger:                    logger,
		slowThreshold:             200 * time.Millisecond,
		logLevel:                  gormlogger.Info,
		ignoreRecordNotFoundError: true,
	}
}

// WithSlowThreshold sets the slow threshold.
func (l *GormLogger) WithSlowThreshold(threshold time.Duration) *GormLogger {
	l.slowThreshold = threshold
	return l
}

// WithLogLevel sets the log level.
func (l *GormLogger) WithLogLevel(level gormlogger.LogLevel) *GormLogger {
	l.logLevel = level
	return l
}

// WithIgnoreRecordNotFoundError sets whether to ignore record not found error.
func (l *GormLogger) WithIgnoreRecordNotFoundError(ignore bool) *GormLogger {
	l.ignoreRecordNotFoundError = ignore
	return l
}

// LogMode implements gormlogger.Interface.
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info implements gormlogger.Interface.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Info {
		l.logger.WithContext(ctx).Infof(msg, data...)
	}
}

// Warn implements gormlogger.Interface.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Warn {
		l.logger.WithContext(ctx).Warnf(msg, data...)
	}
}

// Error implements gormlogger.Interface.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Error {
		l.logger.WithContext(ctx).Errorf(msg, data...)
	}
}

// Trace implements gormlogger.Interface.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Create fields for structured logging
	fields := []Field{
		F("elapsed", elapsed),
		F("rows", rows),
	}

	// Add SQL statement as a field
	if sql != "" {
		fields = append(fields, F("sql", sql))
	}

	// Log based on error and elapsed time
	switch {
	case err != nil && l.logLevel >= gormlogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.ignoreRecordNotFoundError):
		l.logger.WithContext(ctx).WithFields(fields...).WithFields(F("error", err)).Error("GORM error")
	case elapsed > l.slowThreshold && l.slowThreshold > 0 && l.logLevel >= gormlogger.Warn:
		l.logger.WithContext(ctx).WithFields(fields...).Warn("GORM slow query")
	case l.logLevel >= gormlogger.Info:
		l.logger.WithContext(ctx).WithFields(fields...).Debug("GORM query")
	}
}

// GormConfig creates a GORM config with the logger.
func GormConfig(logger Logger) *gorm.Config {
	return &gorm.Config{
		Logger: NewGormLogger(logger),
	}
}

// GormConfigWithOptions creates a GORM config with the logger and options.
func GormConfigWithOptions(logger Logger, slowThreshold time.Duration, logLevel gormlogger.LogLevel, ignoreRecordNotFoundError bool) *gorm.Config {
	return &gorm.Config{
		Logger: NewGormLogger(logger).
			WithSlowThreshold(slowThreshold).
			WithLogLevel(logLevel).
			WithIgnoreRecordNotFoundError(ignoreRecordNotFoundError),
	}
}

// ConvertLevel converts logger.Level to gormlogger.LogLevel.
func ConvertLevel(level Level) gormlogger.LogLevel {
	switch level {
	case DebugLevel:
		return gormlogger.Info // GORM's Info level is used for all queries
	case InfoLevel:
		return gormlogger.Info
	case WarnLevel:
		return gormlogger.Warn
	case ErrorLevel:
		return gormlogger.Error
	case FatalLevel:
		return gormlogger.Error // GORM doesn't have a Fatal level
	default:
		return gormlogger.Info
	}
}

// ConvertGormLevel converts gormlogger.LogLevel to logger.Level.
func ConvertGormLevel(level gormlogger.LogLevel) Level {
	switch level {
	case gormlogger.Silent:
		return FatalLevel // Only log fatal errors
	case gormlogger.Error:
		return ErrorLevel
	case gormlogger.Warn:
		return WarnLevel
	case gormlogger.Info:
		return DebugLevel // GORM's Info level logs all queries, which is more like our Debug level
	default:
		return InfoLevel
	}
}
