package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents the log level.
type Level int

const (
	// DebugLevel is the debug log level.
	DebugLevel Level = iota
	// InfoLevel is the info log level.
	InfoLevel
	// WarnLevel is the warn log level.
	WarnLevel
	// ErrorLevel is the error log level.
	ErrorLevel
	// FatalLevel is the fatal log level.
	FatalLevel
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Color returns the ANSI color code for the log level.
func (l Level) Color() string {
	switch l {
	case DebugLevel:
		return "\033[37m" // White
	case InfoLevel:
		return "\033[32m" // Green
	case WarnLevel:
		return "\033[33m" // Yellow
	case ErrorLevel:
		return "\033[31m" // Red
	case FatalLevel:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Reset
	}
}

// Field represents a log field.
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new log field.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger is the interface for logging.
type Logger interface {
	// Debug logs a debug message.
	Debug(args ...interface{})
	// Debugf logs a formatted debug message.
	Debugf(format string, args ...interface{})
	// Info logs an info message.
	Info(args ...interface{})
	// Infof logs a formatted info message.
	Infof(format string, args ...interface{})
	// Warn logs a warning message.
	Warn(args ...interface{})
	// Warnf logs a formatted warning message.
	Warnf(format string, args ...interface{})
	// Error logs an error message.
	Error(args ...interface{})
	// Errorf logs a formatted error message.
	Errorf(format string, args ...interface{})
	// Fatal logs a fatal message and exits.
	Fatal(args ...interface{})
	// Fatalf logs a formatted fatal message and exits.
	Fatalf(format string, args ...interface{})

	// WithFields returns a new logger with the given fields.
	WithFields(fields ...Field) Logger
	// WithContext returns a new logger with the given context.
	WithContext(ctx context.Context) Logger
	// WithLevel returns a new logger with the given level.
	WithLevel(level Level) Logger
	// WithOutput returns a new logger with the given output.
	WithOutput(output io.Writer) Logger
	// WithCaller returns a new logger with caller information.
	WithCaller(enabled bool) Logger
	// WithTime returns a new logger with time information.
	WithTime(enabled bool) Logger
	// WithColor returns a new logger with color output.
	WithColor(enabled bool) Logger
	// WithTrace returns a new logger with trace information.
	WithTrace(enabled bool) Logger
	// WithServiceName returns a new logger with the given service name.
	WithServiceName(serviceName string) Logger
	// WithEnvironment returns a new logger with the given environment.
	WithEnvironment(environment string) Logger
	// WithTraceInfo returns a new logger with the given trace information.
	WithTraceInfo(traceInfo *TraceInfo) Logger
}

// Config is the configuration for the logger.
type Config struct {
	// Level is the log level.
	Level Level
	// Output is the log output.
	Output io.Writer
	// Fields are the default fields.
	Fields []Field
	// EnableCaller enables caller information.
	EnableCaller bool
	// EnableTime enables time information.
	EnableTime bool
	// EnableColor enables color output.
	EnableColor bool
	// EnableTrace enables trace information.
	EnableTrace bool
	// TimeFormat is the time format.
	TimeFormat string
	// CallerSkip is the number of stack frames to skip when getting caller information.
	CallerSkip int
	// ServiceName is the name of the service.
	ServiceName string
	// Environment is the environment (e.g., production, staging, development).
	Environment string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Level:        InfoLevel,
		Output:       os.Stdout,
		Fields:       []Field{},
		EnableCaller: true,
		EnableTime:   true,
		EnableColor:  true,
		EnableTrace:  true,
		TimeFormat:   time.RFC3339,
		CallerSkip:   2,
		ServiceName:  "unknown",
		Environment:  "development",
	}
}

// logger is the default implementation of Logger.
type logger struct {
	config    *Config
	mu        sync.Mutex
	ctx       context.Context
	traceInfo *TraceInfo
}

// New creates a new logger.
func New(config *Config) Logger {
	if config == nil {
		config = DefaultConfig()
	}

	// 创建跟踪信息
	traceInfo := NewTraceInfo()
	if config.ServiceName != "" {
		traceInfo.WithServiceName(config.ServiceName)
	}
	if config.Environment != "" {
		traceInfo.WithEnvironment(config.Environment)
	}

	return &logger{
		config:    config,
		ctx:       context.Background(),
		traceInfo: traceInfo,
	}
}

// Debug logs a debug message.
func (l *logger) Debug(args ...interface{}) {
	l.log(DebugLevel, fmt.Sprint(args...))
}

// Debugf logs a formatted debug message.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

// Info logs an info message.
func (l *logger) Info(args ...interface{}) {
	l.log(InfoLevel, fmt.Sprint(args...))
}

// Infof logs a formatted info message.
func (l *logger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func (l *logger) Warn(args ...interface{}) {
	l.log(WarnLevel, fmt.Sprint(args...))
}

// Warnf logs a formatted warning message.
func (l *logger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

// Error logs an error message.
func (l *logger) Error(args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprint(args...))
}

// Errorf logs a formatted error message.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits.
func (l *logger) Fatal(args ...interface{}) {
	l.log(FatalLevel, fmt.Sprint(args...))
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits.
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.log(FatalLevel, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// WithFields returns a new logger with the given fields.
func (l *logger) WithFields(fields ...Field) Logger {
	config := *l.config
	config.Fields = append(append([]Field{}, config.Fields...), fields...)
	return &logger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithContext returns a new logger with the given context.
func (l *logger) WithContext(ctx context.Context) Logger {
	newLogger := &logger{
		config:    l.config,
		ctx:       ctx,
		traceInfo: l.traceInfo,
	}

	// 从上下文中获取跟踪信息
	if traceInfo, ok := ctx.Value(traceKey).(*TraceInfo); ok && traceInfo != nil {
		newLogger.traceInfo = traceInfo
	}

	return newLogger
}

// WithLevel returns a new logger with the given level.
func (l *logger) WithLevel(level Level) Logger {
	config := *l.config
	config.Level = level
	return &logger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithOutput returns a new logger with the given output.
func (l *logger) WithOutput(output io.Writer) Logger {
	config := *l.config
	config.Output = output
	return &logger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithCaller returns a new logger with caller information.
func (l *logger) WithCaller(enabled bool) Logger {
	config := *l.config
	config.EnableCaller = enabled
	return &logger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithTime returns a new logger with time information.
func (l *logger) WithTime(enabled bool) Logger {
	config := *l.config
	config.EnableTime = enabled
	return &logger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithColor returns a new logger with color output.
func (l *logger) WithColor(enabled bool) Logger {
	config := *l.config
	config.EnableColor = enabled
	return &logger{
		config:    &config,
		ctx:       l.ctx,
		traceInfo: l.traceInfo,
	}
}

// WithTrace returns a new logger with trace information.
func (l *logger) WithTrace(enabled bool) Logger {
	config := *l.config
	config.EnableTrace = enabled
	return &logger{
		config:    &config,
		ctx:       l.ctx,
		traceInfo: l.traceInfo,
	}
}

// WithServiceName returns a new logger with the given service name.
func (l *logger) WithServiceName(serviceName string) Logger {
	config := *l.config
	config.ServiceName = serviceName

	// 更新跟踪信息
	newTraceInfo := *l.traceInfo
	newTraceInfo.WithServiceName(serviceName)

	return &logger{
		config:    &config,
		ctx:       l.ctx,
		traceInfo: &newTraceInfo,
	}
}

// WithEnvironment returns a new logger with the given environment.
func (l *logger) WithEnvironment(environment string) Logger {
	config := *l.config
	config.Environment = environment

	// 更新跟踪信息
	newTraceInfo := *l.traceInfo
	newTraceInfo.WithEnvironment(environment)

	return &logger{
		config:    &config,
		ctx:       l.ctx,
		traceInfo: &newTraceInfo,
	}
}

// WithTraceInfo returns a new logger with the given trace information.
func (l *logger) WithTraceInfo(traceInfo *TraceInfo) Logger {
	return &logger{
		config:    l.config,
		ctx:       l.ctx,
		traceInfo: traceInfo,
	}
}

// log logs a message with the given level.
func (l *logger) log(level Level, message string) {
	if level < l.config.Level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var builder strings.Builder

	// Add time
	if l.config.EnableTime {
		timeStr := time.Now().Format(l.config.TimeFormat)
		if l.config.EnableColor {
			builder.WriteString("\033[90m")
		}
		builder.WriteString(timeStr)
		if l.config.EnableColor {
			builder.WriteString("\033[0m")
		}
		builder.WriteString(" ")
	}

	// Add level
	if l.config.EnableColor {
		builder.WriteString(level.Color())
	}
	builder.WriteString("[")
	builder.WriteString(level.String())
	builder.WriteString("]")
	if l.config.EnableColor {
		builder.WriteString("\033[0m")
	}
	builder.WriteString(" ")

	// Add caller
	if l.config.EnableCaller {
		_, file, line, ok := runtime.Caller(l.config.CallerSkip)
		if ok {
			file = filepath.Base(file)
			if l.config.EnableColor {
				builder.WriteString("\033[90m")
			}
			builder.WriteString(file)
			builder.WriteString(":")
			builder.WriteString(fmt.Sprintf("%d", line))
			if l.config.EnableColor {
				builder.WriteString("\033[0m")
			}
			builder.WriteString(" ")
		}
	}

	// Add message
	builder.WriteString(message)

	// Add fields
	fields := l.config.Fields

	// Add trace fields if enabled
	if l.config.EnableTrace && l.traceInfo != nil {
		traceFields := l.traceInfo.ToFields()
		fields = append(fields, traceFields...)
	}

	if len(fields) > 0 {
		builder.WriteString(" ")
		for i, field := range fields {
			if i > 0 {
				builder.WriteString(" ")
			}
			if l.config.EnableColor {
				builder.WriteString("\033[36m")
			}
			builder.WriteString(field.Key)
			builder.WriteString("=")
			if l.config.EnableColor {
				builder.WriteString("\033[0m")
			}
			builder.WriteString(fmt.Sprintf("%v", field.Value))
		}
	}

	// Add newline
	builder.WriteString("\n")

	// Write to output
	l.config.Output.Write([]byte(builder.String()))
}

// global is the global logger.
var global = New(DefaultConfig())

// SetGlobal sets the global logger.
func SetGlobal(logger Logger) {
	global = logger
}

// Debug logs a debug message.
func Debug(args ...interface{}) {
	global.Debug(args...)
}

// Debugf logs a formatted debug message.
func Debugf(format string, args ...interface{}) {
	global.Debugf(format, args...)
}

// Info logs an info message.
func Info(args ...interface{}) {
	global.Info(args...)
}

// Infof logs a formatted info message.
func Infof(format string, args ...interface{}) {
	global.Infof(format, args...)
}

// Warn logs a warning message.
func Warn(args ...interface{}) {
	global.Warn(args...)
}

// Warnf logs a formatted warning message.
func Warnf(format string, args ...interface{}) {
	global.Warnf(format, args...)
}

// Error logs an error message.
func Error(args ...interface{}) {
	global.Error(args...)
}

// Errorf logs a formatted error message.
func Errorf(format string, args ...interface{}) {
	global.Errorf(format, args...)
}

// Fatal logs a fatal message and exits.
func Fatal(args ...interface{}) {
	global.Fatal(args...)
}

// Fatalf logs a formatted fatal message and exits.
func Fatalf(format string, args ...interface{}) {
	global.Fatalf(format, args...)
}

// WithFields returns a new logger with the given fields.
func WithFields(fields ...Field) Logger {
	return global.WithFields(fields...)
}

// WithContext returns a new logger with the given context.
func WithContext(ctx context.Context) Logger {
	return global.WithContext(ctx)
}

// WithLevel returns a new logger with the given level.
func WithLevel(level Level) Logger {
	return global.WithLevel(level)
}

// WithOutput returns a new logger with the given output.
func WithOutput(output io.Writer) Logger {
	return global.WithOutput(output)
}

// WithCaller returns a new logger with caller information.
func WithCaller(enabled bool) Logger {
	return global.WithCaller(enabled)
}

// WithTime returns a new logger with time information.
func WithTime(enabled bool) Logger {
	return global.WithTime(enabled)
}

// WithColor returns a new logger with color output.
func WithColor(enabled bool) Logger {
	return global.WithColor(enabled)
}

// WithTrace returns a new logger with trace information.
func WithTrace(enabled bool) Logger {
	return global.WithTrace(enabled)
}

// WithServiceName returns a new logger with the given service name.
func WithServiceName(serviceName string) Logger {
	return global.WithServiceName(serviceName)
}

// WithEnvironment returns a new logger with the given environment.
func WithEnvironment(environment string) Logger {
	return global.WithEnvironment(environment)
}

// WithTraceInfo returns a new logger with the given trace information.
func WithTraceInfo(traceInfo *TraceInfo) Logger {
	return global.WithTraceInfo(traceInfo)
}
