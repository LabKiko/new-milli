package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TraceKey 定义了链路追踪相关的键
type TraceKey string

const (
	// RequestIDKey 是请求ID的键
	RequestIDKey TraceKey = "request_id"
	// TraceIDKey 是跟踪ID的键
	TraceIDKey TraceKey = "trace_id"
	// SpanIDKey 是跨度ID的键
	SpanIDKey TraceKey = "span_id"
	// ParentSpanIDKey 是父跨度ID的键
	ParentSpanIDKey TraceKey = "parent_span_id"
	// ServiceNameKey 是服务名称的键
	ServiceNameKey TraceKey = "service"
	// EnvironmentKey 是环境的键
	EnvironmentKey TraceKey = "env"
)

// traceContextKey 是上下文中存储跟踪信息的键
type traceContextKey int

const (
	// traceKey 是上下文中存储跟踪信息的键
	traceKey traceContextKey = iota
)

// TraceInfo 包含链路追踪的信息
type TraceInfo struct {
	RequestID    string
	TraceID      string
	SpanID       string
	ParentSpanID string
	ServiceName  string
	Environment  string
	CustomFields map[string]string
	mu           sync.RWMutex
}

// NewTraceInfo 创建一个新的跟踪信息
func NewTraceInfo() *TraceInfo {
	return &TraceInfo{
		RequestID:    generateID(),
		TraceID:      generateID(),
		SpanID:       generateID(),
		ParentSpanID: "",
		CustomFields: make(map[string]string),
	}
}

// WithRequestID 设置请求ID
func (t *TraceInfo) WithRequestID(requestID string) *TraceInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.RequestID = requestID
	return t
}

// WithTraceID 设置跟踪ID
func (t *TraceInfo) WithTraceID(traceID string) *TraceInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.TraceID = traceID
	return t
}

// WithSpanID 设置跨度ID
func (t *TraceInfo) WithSpanID(spanID string) *TraceInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.SpanID = spanID
	return t
}

// WithParentSpanID 设置父跨度ID
func (t *TraceInfo) WithParentSpanID(parentSpanID string) *TraceInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ParentSpanID = parentSpanID
	return t
}

// WithServiceName 设置服务名称
func (t *TraceInfo) WithServiceName(serviceName string) *TraceInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ServiceName = serviceName
	return t
}

// WithEnvironment 设置环境
func (t *TraceInfo) WithEnvironment(env string) *TraceInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Environment = env
	return t
}

// WithCustomField 设置自定义字段
func (t *TraceInfo) WithCustomField(key, value string) *TraceInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.CustomFields[key] = value
	return t
}

// NewChildSpan 创建一个子跨度
func (t *TraceInfo) NewChildSpan() *TraceInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	child := &TraceInfo{
		RequestID:    t.RequestID,
		TraceID:      t.TraceID,
		SpanID:       generateID(),
		ParentSpanID: t.SpanID,
		ServiceName:  t.ServiceName,
		Environment:  t.Environment,
		CustomFields: make(map[string]string),
	}

	// 复制自定义字段
	for k, v := range t.CustomFields {
		child.CustomFields[k] = v
	}

	return child
}

// ToFields 将跟踪信息转换为日志字段
func (t *TraceInfo) ToFields() []Field {
	t.mu.RLock()
	defer t.mu.RUnlock()

	fields := []Field{}

	if t.RequestID != "" {
		fields = append(fields, F(string(RequestIDKey), t.RequestID))
	}

	if t.TraceID != "" {
		fields = append(fields, F(string(TraceIDKey), t.TraceID))
	}

	if t.SpanID != "" {
		fields = append(fields, F(string(SpanIDKey), t.SpanID))
	}

	if t.ParentSpanID != "" {
		fields = append(fields, F(string(ParentSpanIDKey), t.ParentSpanID))
	}

	if t.ServiceName != "" {
		fields = append(fields, F(string(ServiceNameKey), t.ServiceName))
	}

	if t.Environment != "" {
		fields = append(fields, F(string(EnvironmentKey), t.Environment))
	}

	// 添加自定义字段
	for k, v := range t.CustomFields {
		fields = append(fields, F(k, v))
	}

	return fields
}

// String 返回跟踪信息的字符串表示
func (t *TraceInfo) String() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var sb strings.Builder

	if t.RequestID != "" {
		sb.WriteString(fmt.Sprintf("%s=%s ", RequestIDKey, t.RequestID))
	}

	if t.TraceID != "" {
		sb.WriteString(fmt.Sprintf("%s=%s ", TraceIDKey, t.TraceID))
	}

	if t.SpanID != "" {
		sb.WriteString(fmt.Sprintf("%s=%s ", SpanIDKey, t.SpanID))
	}

	if t.ParentSpanID != "" {
		sb.WriteString(fmt.Sprintf("%s=%s ", ParentSpanIDKey, t.ParentSpanID))
	}

	return strings.TrimSpace(sb.String())
}

// generateID 生成一个随机ID
func generateID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// WithTraceInfo 将跟踪信息添加到上下文
func WithTraceInfo(ctx context.Context, traceInfo *TraceInfo) context.Context {
	return context.WithValue(ctx, traceKey, traceInfo)
}

// FromContext 从上下文中获取跟踪信息
func TraceInfoFromContext(ctx context.Context) *TraceInfo {
	if ctx == nil {
		return NewTraceInfo()
	}

	if traceInfo, ok := ctx.Value(traceKey).(*TraceInfo); ok {
		return traceInfo
	}

	return NewTraceInfo()
}

// WithTraceContext 创建一个带有跟踪信息的上下文
func WithTraceContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	// 如果上下文中已经有跟踪信息，则不创建新的
	if _, ok := ctx.Value(traceKey).(*TraceInfo); ok {
		return ctx
	}

	return WithTraceInfo(ctx, NewTraceInfo())
}

// WithChildSpan 创建一个带有子跨度的上下文
func WithChildSpan(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	parentTrace := TraceInfoFromContext(ctx)
	childTrace := parentTrace.NewChildSpan()

	return WithTraceInfo(ctx, childTrace)
}

// LoggerWithTrace 返回一个带有跟踪信息的日志器
func LoggerWithTrace(ctx context.Context, logger Logger) Logger {
	traceInfo := TraceInfoFromContext(ctx)
	return logger.WithFields(traceInfo.ToFields()...)
}

// DebugWithTrace 使用跟踪信息记录调试日志
func DebugWithTrace(ctx context.Context, args ...interface{}) {
	LoggerWithTrace(ctx, global).Debug(args...)
}

// DebugfWithTrace 使用跟踪信息记录格式化调试日志
func DebugfWithTrace(ctx context.Context, format string, args ...interface{}) {
	LoggerWithTrace(ctx, global).Debugf(format, args...)
}

// InfoWithTrace 使用跟踪信息记录信息日志
func InfoWithTrace(ctx context.Context, args ...interface{}) {
	LoggerWithTrace(ctx, global).Info(args...)
}

// InfofWithTrace 使用跟踪信息记录格式化信息日志
func InfofWithTrace(ctx context.Context, format string, args ...interface{}) {
	LoggerWithTrace(ctx, global).Infof(format, args...)
}

// WarnWithTrace 使用跟踪信息记录警告日志
func WarnWithTrace(ctx context.Context, args ...interface{}) {
	LoggerWithTrace(ctx, global).Warn(args...)
}

// WarnfWithTrace 使用跟踪信息记录格式化警告日志
func WarnfWithTrace(ctx context.Context, format string, args ...interface{}) {
	LoggerWithTrace(ctx, global).Warnf(format, args...)
}

// ErrorWithTrace 使用跟踪信息记录错误日志
func ErrorWithTrace(ctx context.Context, args ...interface{}) {
	LoggerWithTrace(ctx, global).Error(args...)
}

// ErrorfWithTrace 使用跟踪信息记录格式化错误日志
func ErrorfWithTrace(ctx context.Context, format string, args ...interface{}) {
	LoggerWithTrace(ctx, global).Errorf(format, args...)
}

// FatalWithTrace 使用跟踪信息记录致命日志
func FatalWithTrace(ctx context.Context, args ...interface{}) {
	LoggerWithTrace(ctx, global).Fatal(args...)
}

// FatalfWithTrace 使用跟踪信息记录格式化致命日志
func FatalfWithTrace(ctx context.Context, format string, args ...interface{}) {
	LoggerWithTrace(ctx, global).Fatalf(format, args...)
}
