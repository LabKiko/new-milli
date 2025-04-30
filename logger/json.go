package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"
)

// JSONLogger is a logger that outputs JSON.
type JSONLogger struct {
	config *JSONConfig
	mu     sync.Mutex
	ctx    context.Context
}

// JSONConfig is the configuration for the JSON logger.
type JSONConfig struct {
	// Level is the log level.
	Level Level
	// Output is the log output.
	Output io.Writer
	// Fields are the default fields.
	Fields map[string]interface{}
	// EnableCaller enables caller information.
	EnableCaller bool
	// EnableTime enables time information.
	EnableTime bool
	// TimeFormat is the time format.
	TimeFormat string
	// CallerSkip is the number of stack frames to skip when getting caller information.
	CallerSkip int
	// TimeKey is the key for the time field.
	TimeKey string
	// LevelKey is the key for the level field.
	LevelKey string
	// MessageKey is the key for the message field.
	MessageKey string
	// CallerKey is the key for the caller field.
	CallerKey string
	// StacktraceKey is the key for the stacktrace field.
	StacktraceKey string
	// PrettyPrint enables pretty printing.
	PrettyPrint bool
}

// DefaultJSONConfig returns the default JSON configuration.
func DefaultJSONConfig() *JSONConfig {
	return &JSONConfig{
		Level:         InfoLevel,
		Output:        nil,
		Fields:        make(map[string]interface{}),
		EnableCaller:  true,
		EnableTime:    true,
		TimeFormat:    time.RFC3339,
		CallerSkip:    2,
		TimeKey:       "time",
		LevelKey:      "level",
		MessageKey:    "message",
		CallerKey:     "caller",
		StacktraceKey: "stacktrace",
		PrettyPrint:   false,
	}
}

// NewJSONLogger creates a new JSON logger.
func NewJSONLogger(config *JSONConfig) Logger {
	if config == nil {
		config = DefaultJSONConfig()
	}
	if config.Output == nil {
		config.Output = DefaultConfig().Output
	}
	return &JSONLogger{
		config: config,
		ctx:    context.Background(),
	}
}

// Debug logs a debug message.
func (l *JSONLogger) Debug(args ...interface{}) {
	l.log(DebugLevel, fmt.Sprint(args...))
}

// Debugf logs a formatted debug message.
func (l *JSONLogger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

// Info logs an info message.
func (l *JSONLogger) Info(args ...interface{}) {
	l.log(InfoLevel, fmt.Sprint(args...))
}

// Infof logs a formatted info message.
func (l *JSONLogger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func (l *JSONLogger) Warn(args ...interface{}) {
	l.log(WarnLevel, fmt.Sprint(args...))
}

// Warnf logs a formatted warning message.
func (l *JSONLogger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

// Error logs an error message.
func (l *JSONLogger) Error(args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprint(args...))
}

// Errorf logs a formatted error message.
func (l *JSONLogger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits.
func (l *JSONLogger) Fatal(args ...interface{}) {
	l.log(FatalLevel, fmt.Sprint(args...))
	panic(fmt.Sprint(args...))
}

// Fatalf logs a formatted fatal message and exits.
func (l *JSONLogger) Fatalf(format string, args ...interface{}) {
	l.log(FatalLevel, fmt.Sprintf(format, args...))
	panic(fmt.Sprintf(format, args...))
}

// WithFields returns a new logger with the given fields.
func (l *JSONLogger) WithFields(fields ...Field) Logger {
	config := *l.config
	newFields := make(map[string]interface{}, len(config.Fields)+len(fields))
	for k, v := range config.Fields {
		newFields[k] = v
	}
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}
	config.Fields = newFields
	return &JSONLogger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithContext returns a new logger with the given context.
func (l *JSONLogger) WithContext(ctx context.Context) Logger {
	return &JSONLogger{
		config: l.config,
		ctx:    ctx,
	}
}

// WithLevel returns a new logger with the given level.
func (l *JSONLogger) WithLevel(level Level) Logger {
	config := *l.config
	config.Level = level
	return &JSONLogger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithOutput returns a new logger with the given output.
func (l *JSONLogger) WithOutput(output io.Writer) Logger {
	config := *l.config
	config.Output = output
	return &JSONLogger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithCaller returns a new logger with caller information.
func (l *JSONLogger) WithCaller(enabled bool) Logger {
	config := *l.config
	config.EnableCaller = enabled
	return &JSONLogger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithTime returns a new logger with time information.
func (l *JSONLogger) WithTime(enabled bool) Logger {
	config := *l.config
	config.EnableTime = enabled
	return &JSONLogger{
		config: &config,
		ctx:    l.ctx,
	}
}

// WithColor returns a new logger with color output.
// This is a no-op for JSON logger.
func (l *JSONLogger) WithColor(enabled bool) Logger {
	return l
}

// log logs a message with the given level.
func (l *JSONLogger) log(level Level, message string) {
	if level < l.config.Level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Create the log entry
	entry := make(map[string]interface{}, len(l.config.Fields)+3)

	// Add time
	if l.config.EnableTime {
		entry[l.config.TimeKey] = time.Now().Format(l.config.TimeFormat)
	}

	// Add level
	entry[l.config.LevelKey] = level.String()

	// Add message
	entry[l.config.MessageKey] = message

	// Add caller
	if l.config.EnableCaller {
		_, file, line, ok := runtime.Caller(l.config.CallerSkip)
		if ok {
			entry[l.config.CallerKey] = fmt.Sprintf("%s:%d", file, line)
		}
	}

	// Add fields
	for k, v := range l.config.Fields {
		entry[k] = v
	}

	// Marshal to JSON
	var data []byte
	var err error
	if l.config.PrettyPrint {
		data, err = json.MarshalIndent(entry, "", "  ")
	} else {
		data, err = json.Marshal(entry)
	}
	if err != nil {
		return
	}

	// Add newline
	data = append(data, '\n')

	// Write to output
	l.config.Output.Write(data)
}
