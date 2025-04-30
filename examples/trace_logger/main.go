package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"new-milli/logger"
)

// 模拟服务名称
const (
	ServiceName = "user-service"
	Environment = "development"
)

func main() {
	// 创建一个自定义日志器，启用链路追踪
	log := logger.New(&logger.Config{
		Level:        logger.DebugLevel,
		Output:       os.Stdout,
		EnableCaller: true,
		EnableTime:   true,
		EnableColor:  true,
		EnableTrace:  true,
		ServiceName:  ServiceName,
		Environment:  Environment,
	})

	// 创建一个JSON格式的文件日志器
	fileWriter := logger.NewFileWriter("logs/trace.log")
	defer fileWriter.Close()
	
	jsonConfig := logger.DefaultJSONConfig()
	jsonConfig.Output = fileWriter
	jsonConfig.PrettyPrint = false
	jsonLogger := logger.NewJSONLogger(jsonConfig).
		WithServiceName(ServiceName).
		WithEnvironment(Environment)

	// 设置全局日志器
	logger.SetGlobal(log)

	// 模拟HTTP服务器
	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		// 为每个请求创建一个新的上下文，包含跟踪信息
		ctx := createRequestContext(r)
		
		// 使用上下文记录日志
		logger.InfoWithTrace(ctx, "收到用户请求")
		
		// 处理请求
		processRequest(ctx, w, r)
		
		// 响应请求
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
		
		// 记录请求完成
		logger.InfoWithTrace(ctx, "请求处理完成")
		
		// 同时记录到JSON文件
		jsonLogger.WithContext(ctx).Info("请求处理完成")
	})

	// 模拟一个请求
	simulateRequest()

	fmt.Println("\n--- 模拟微服务调用链 ---")
	// 模拟微服务调用链
	simulateMicroserviceChain()
}

// createRequestContext 为请求创建上下文，包含跟踪信息
func createRequestContext(r *http.Request) context.Context {
	ctx := context.Background()
	
	// 创建跟踪信息
	traceInfo := logger.NewTraceInfo().
		WithRequestID(generateRequestID()).
		WithServiceName(ServiceName).
		WithEnvironment(Environment)
	
	// 从请求头中获取跟踪ID（如果存在）
	if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
		traceInfo.WithTraceID(traceID)
	}
	
	// 从请求头中获取父跨度ID（如果存在）
	if parentSpanID := r.Header.Get("X-Parent-Span-ID"); parentSpanID != "" {
		traceInfo.WithParentSpanID(parentSpanID)
	}
	
	// 添加自定义字段
	traceInfo.WithCustomField("http_method", r.Method).
		WithCustomField("http_path", r.URL.Path).
		WithCustomField("user_agent", r.UserAgent())
	
	// 将跟踪信息添加到上下文
	return logger.WithTraceInfo(ctx, traceInfo)
}

// processRequest 处理请求
func processRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// 创建一个子跨度
	childCtx := logger.WithChildSpan(ctx)
	
	// 记录处理开始
	logger.DebugWithTrace(childCtx, "开始处理请求")
	
	// 模拟处理时间
	time.Sleep(100 * time.Millisecond)
	
	// 调用用户服务
	getUserInfo(childCtx, "123")
	
	// 记录处理结束
	logger.DebugWithTrace(childCtx, "请求处理完成")
}

// getUserInfo 获取用户信息
func getUserInfo(ctx context.Context, userID string) {
	// 创建一个子跨度
	childCtx := logger.WithChildSpan(ctx)
	
	// 记录处理开始
	logger.DebugWithTrace(childCtx, "获取用户信息", logger.F("user_id", userID))
	
	// 模拟处理时间
	time.Sleep(50 * time.Millisecond)
	
	// 模拟数据库查询
	queryDatabase(childCtx, "SELECT * FROM users WHERE id = ?", userID)
	
	// 记录处理结束
	logger.DebugWithTrace(childCtx, "用户信息获取完成")
}

// queryDatabase 模拟数据库查询
func queryDatabase(ctx context.Context, query string, args ...interface{}) {
	// 创建一个子跨度
	childCtx := logger.WithChildSpan(ctx)
	
	// 记录查询开始
	logger.DebugWithTrace(childCtx, "执行数据库查询", 
		logger.F("query", query),
		logger.F("args", fmt.Sprintf("%v", args)),
	)
	
	// 模拟查询时间
	time.Sleep(30 * time.Millisecond)
	
	// 记录查询结束
	logger.DebugWithTrace(childCtx, "数据库查询完成")
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// simulateRequest 模拟一个HTTP请求
func simulateRequest() {
	fmt.Println("--- 模拟HTTP请求 ---")
	
	// 创建一个模拟请求
	r, _ := http.NewRequest("GET", "/api/users", nil)
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	// 处理请求
	w := &mockResponseWriter{}
	http.DefaultServeMux.ServeHTTP(w, r)
}

// mockResponseWriter 模拟HTTP响应写入器
type mockResponseWriter struct {
	headers http.Header
	status  int
	body    []byte
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(http.Header)
	}
	return m.headers
}

func (m *mockResponseWriter) Write(body []byte) (int, error) {
	m.body = body
	if m.status == 0 {
		m.status = http.StatusOK
	}
	return len(body), nil
}

func (m *mockResponseWriter) WriteHeader(status int) {
	m.status = status
}

// simulateMicroserviceChain 模拟微服务调用链
func simulateMicroserviceChain() {
	// 创建根上下文
	rootCtx := context.Background()
	
	// 创建跟踪信息
	traceInfo := logger.NewTraceInfo().
		WithRequestID("req-12345").
		WithServiceName("api-gateway").
		WithEnvironment(Environment)
	
	// 将跟踪信息添加到上下文
	ctx := logger.WithTraceInfo(rootCtx, traceInfo)
	
	// API网关接收请求
	logger.InfoWithTrace(ctx, "API网关接收请求", logger.F("path", "/api/orders"))
	
	// 调用订单服务
	callOrderService(ctx)
	
	// API网关返回响应
	logger.InfoWithTrace(ctx, "API网关返回响应")
}

// callOrderService 调用订单服务
func callOrderService(ctx context.Context) {
	// 创建订单服务的上下文
	orderCtx := logger.WithChildSpan(ctx)
	
	// 更新服务名称
	traceInfo := logger.TraceInfoFromContext(orderCtx).WithServiceName("order-service")
	orderCtx = logger.WithTraceInfo(orderCtx, traceInfo)
	
	// 订单服务处理请求
	logger.InfoWithTrace(orderCtx, "订单服务接收请求")
	
	// 查询订单
	logger.DebugWithTrace(orderCtx, "查询订单信息", logger.F("order_id", "ORD-67890"))
	
	// 调用用户服务
	callUserService(orderCtx)
	
	// 调用库存服务
	callInventoryService(orderCtx)
	
	// 订单服务返回响应
	logger.InfoWithTrace(orderCtx, "订单服务返回响应")
}

// callUserService 调用用户服务
func callUserService(ctx context.Context) {
	// 创建用户服务的上下文
	userCtx := logger.WithChildSpan(ctx)
	
	// 更新服务名称
	traceInfo := logger.TraceInfoFromContext(userCtx).WithServiceName("user-service")
	userCtx = logger.WithTraceInfo(userCtx, traceInfo)
	
	// 用户服务处理请求
	logger.InfoWithTrace(userCtx, "用户服务接收请求")
	
	// 查询用户
	logger.DebugWithTrace(userCtx, "查询用户信息", logger.F("user_id", "USR-12345"))
	
	// 用户服务返回响应
	logger.InfoWithTrace(userCtx, "用户服务返回响应")
}

// callInventoryService 调用库存服务
func callInventoryService(ctx context.Context) {
	// 创建库存服务的上下文
	invCtx := logger.WithChildSpan(ctx)
	
	// 更新服务名称
	traceInfo := logger.TraceInfoFromContext(invCtx).WithServiceName("inventory-service")
	invCtx = logger.WithTraceInfo(invCtx, traceInfo)
	
	// 库存服务处理请求
	logger.InfoWithTrace(invCtx, "库存服务接收请求")
	
	// 查询库存
	logger.DebugWithTrace(invCtx, "查询库存信息", logger.F("product_id", "PRD-54321"))
	
	// 调用仓库服务
	callWarehouseService(invCtx)
	
	// 库存服务返回响应
	logger.InfoWithTrace(invCtx, "库存服务返回响应")
}

// callWarehouseService 调用仓库服务
func callWarehouseService(ctx context.Context) {
	// 创建仓库服务的上下文
	whCtx := logger.WithChildSpan(ctx)
	
	// 更新服务名称
	traceInfo := logger.TraceInfoFromContext(whCtx).WithServiceName("warehouse-service")
	whCtx = logger.WithTraceInfo(whCtx, traceInfo)
	
	// 仓库服务处理请求
	logger.InfoWithTrace(whCtx, "仓库服务接收请求")
	
	// 查询仓库
	logger.DebugWithTrace(whCtx, "查询仓库信息", logger.F("warehouse_id", "WH-001"))
	
	// 仓库服务返回响应
	logger.InfoWithTrace(whCtx, "仓库服务返回响应")
}
