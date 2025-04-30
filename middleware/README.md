# New Milli 中间件系统

New Milli 中间件系统是一个灵活、可扩展的中间件集成解决方案，支持多种常用中间件。

## 支持的中间件

- **Recovery**: 从 panic 中恢复，防止服务崩溃
- **Logging**: 请求日志记录
- **Tracing**: 分布式链路追踪
- **Rate Limiting**: 限流，防止服务过载
- **Circuit Breaker**: 熔断，提高系统容错性
- **Metrics**: 监控指标，用于系统监控和告警

## 快速开始

### 基本用法

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/cloudwego/hertz/pkg/app"
    "new-milli"
    "new-milli/middleware/logging"
    "new-milli/middleware/recovery"
    "new-milli/transport"
    "new-milli/transport/http"
)

func main() {
    // 创建 HTTP 服务器
    httpServer := http.NewServer(
        transport.Address(":8000"),
        transport.Middleware(
            recovery.Server(),
            logging.Server(),
        ),
    )
    
    // 获取 Hertz 服务器实例
    hertzServer := httpServer.GetHertzServer()
    
    // 注册路由
    hertzServer.GET("/", func(ctx context.Context, c *app.RequestContext) {
        c.String(200, "Hello, World!")
    })
    
    // 创建应用
    app, err := newMilli.New(
        newMilli.Name("example"),
        newMilli.Version("v1.0.0"),
        newMilli.Server(httpServer),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 运行应用
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

## 中间件链

中间件按照添加的顺序执行，可以使用 `middleware.Chain` 函数组合多个中间件：

```go
// 创建中间件链
chain := middleware.Chain(
    recovery.Server(),
    tracing.Server(),
    logging.Server(),
)

// 使用中间件链
httpServer := http.NewServer(
    transport.Middleware(chain),
)
```

## 中间件详解

### Recovery 中间件

Recovery 中间件用于从 panic 中恢复，防止服务崩溃。

```go
// 使用默认配置
recovery.Server()

// 自定义配置
recovery.Server(
    recovery.WithStackSize(8 * 1024), // 设置堆栈大小
    recovery.WithDisableStackAll(false), // 是否禁用堆栈跟踪
    recovery.WithDisablePrintStack(false), // 是否禁用打印堆栈
    recovery.WithRecoveryHandler(func(ctx context.Context, err interface{}) error {
        // 自定义恢复处理函数
        return fmt.Errorf("panic: %v", err)
    }),
)
```

### Logging 中间件

Logging 中间件用于记录请求日志。

```go
// 使用默认配置
logging.Server()

// 自定义配置
logging.Server(
    logging.WithLevel(klog.LevelInfo), // 设置日志级别
    logging.WithSlowThreshold(time.Millisecond * 500), // 设置慢请求阈值
)
```

### Tracing 中间件

Tracing 中间件用于分布式链路追踪。

```go
// 使用默认配置
tracing.Server()

// 自定义配置
tracing.Server(
    tracing.WithTracerProvider(provider), // 设置 TracerProvider
    tracing.WithPropagators(propagators), // 设置 TextMapPropagator
)
```

### Rate Limiting 中间件

Rate Limiting 中间件用于限流，防止服务过载。

```go
// 使用默认配置
ratelimit.Server()

// 自定义配置
ratelimit.Server(
    ratelimit.WithRate(100), // 设置每秒填充速率
    ratelimit.WithCapacity(100), // 设置桶容量
    ratelimit.WithWaitIfFull(false), // 是否等待令牌可用
)

// 创建自定义限流器
limiter := ratelimit.NewLimiter(100, 100)

// 检查是否允许请求
if ratelimit.Allow(limiter, 1) {
    // 处理请求
}
```

### Circuit Breaker 中间件

Circuit Breaker 中间件用于熔断，提高系统容错性。

```go
// 使用默认配置
circuitbreaker.Server()

// 自定义配置
circuitbreaker.Server(
    circuitbreaker.WithMaxRequests(100), // 设置半开状态下允许的最大请求数
    circuitbreaker.WithInterval(time.Minute), // 设置关闭状态的周期
    circuitbreaker.WithTimeout(time.Minute), // 设置开启状态的超时时间
    circuitbreaker.WithReadyToTrip(func(counts gobreaker.Counts) bool {
        // 自定义熔断条件
        failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
        return counts.Requests >= 10 && failureRatio >= 0.5
    }),
    circuitbreaker.WithOnStateChange(func(name string, from gobreaker.State, to gobreaker.State) {
        // 状态变化回调
        log.Printf("Circuit breaker %s changed from %s to %s", name, from, to)
    }),
    circuitbreaker.WithFallbackHandler(func(ctx context.Context, req interface{}) (interface{}, error) {
        // 熔断时的降级处理
        return nil, errors.New("service unavailable")
    }),
)

// 创建自定义熔断器
cb := circuitbreaker.NewCircuitBreaker("my-service",
    circuitbreaker.WithTimeout(time.Second * 10),
)
```

### Metrics 中间件

Metrics 中间件用于收集监控指标，用于系统监控和告警。

```go
// 使用默认配置
metrics.Server()

// 自定义配置
metrics.Server(
    metrics.WithNamespace("my_service"), // 设置命名空间
    metrics.WithSubsystem("http"), // 设置子系统
    metrics.WithBuckets([]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}), // 设置直方图桶
    metrics.WithConstLabels(prometheus.Labels{"env": "prod"}), // 设置常量标签
    metrics.WithLabelNames("method", "path", "status"), // 设置标签名称
)

// 注册 Prometheus 指标处理程序
hertzServer.GET("/metrics", metrics.Handler())

// 创建自定义指标
counter := metrics.NewCounter("my_counter", "My counter",
    metrics.WithNamespace("my_service"),
    metrics.WithLabelNames("method", "path"),
)

// 增加计数器
counter.WithLabelValues("GET", "/api").Inc()

// 创建直方图
histogram := metrics.NewHistogram("my_histogram", "My histogram",
    metrics.WithNamespace("my_service"),
    metrics.WithBuckets([]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}),
)

// 观察值
histogram.WithLabelValues().Observe(0.1)
```

## 客户端中间件

所有中间件都支持客户端版本，用法与服务器端类似：

```go
// 创建 HTTP 客户端
httpClient := http.NewClient(
    transport.Middleware(
        recovery.Client(),
        tracing.Client(),
        metrics.Client(),
        ratelimit.Client(),
        circuitbreaker.Client(),
        logging.Client(),
    ),
)
```

## 自定义中间件

可以轻松创建自定义中间件：

```go
// 创建自定义中间件
func MyMiddleware() middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
            // 前置处理
            log.Println("Before request")
            
            // 调用下一个处理程序
            reply, err = handler(ctx, req)
            
            // 后置处理
            log.Println("After request")
            
            return reply, err
        }
    }
}
```
