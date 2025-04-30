# New Milli 日志模块

New Milli 日志模块是一个灵活、可扩展的日志系统，支持多种输出格式和目标。

## 特性

- 多级别日志（DEBUG, INFO, WARN, ERROR, FATAL）
- 结构化日志和字段支持
- 彩色控制台输出
- JSON 格式输出
- 文件输出和日志轮转
- 上下文（Context）支持
- 可配置的时间格式和调用者信息
- 全局默认日志器和自定义日志器

## 快速开始

### 基本用法

```go
package main

import "new-milli/logger"

func main() {
    // 使用默认日志器
    logger.Info("这是一条信息日志")
    logger.Warn("这是一条警告日志")
    logger.Error("这是一条错误日志")
    
    // 格式化日志
    logger.Infof("用户 %s 登录成功", "张三")
    
    // 使用字段
    logger.WithFields(
        logger.F("user_id", 123),
        logger.F("request_id", "abc-123"),
    ).Info("用户登录成功")
}
```

### 日志级别

```go
// 默认级别是 INFO，DEBUG 日志不会显示
logger.Debug("这条日志不会显示")

// 修改日志级别
debugLogger := logger.WithLevel(logger.DebugLevel)
debugLogger.Debug("现在可以看到调试日志了")

// 全局修改日志级别
logger.SetGlobal(logger.WithLevel(logger.DebugLevel))
logger.Debug("现在全局都可以看到调试日志了")
```

### 输出到文件

```go
// 输出到文件
fileWriter := logger.NewFileWriter("logs/app.log")
defer fileWriter.Close()
fileLogger := logger.WithOutput(fileWriter)
fileLogger.Info("这条日志写入到文件")

// 使用轮转文件
rotatingWriter := logger.NewRotatingFileWriter("logs/rotating.log")
rotatingWriter.MaxSize = 100 * 1024 * 1024 // 100MB
rotatingWriter.MaxBackups = 10
rotatingWriter.MaxAge = 30 // 30天
rotatingLogger := logger.WithOutput(rotatingWriter)
rotatingLogger.Info("这条日志写入到轮转文件")
```

### JSON 格式

```go
// 使用 JSON 格式
jsonLogger := logger.NewJSONLogger(nil)
jsonLogger.Info("这是一条 JSON 格式的日志")

// 带有漂亮打印的 JSON 格式
prettyConfig := logger.DefaultJSONConfig()
prettyConfig.PrettyPrint = true
prettyLogger := logger.NewJSONLogger(prettyConfig)
prettyLogger.Info("这是一条漂亮打印的 JSON 日志")
```

### 上下文支持

```go
// 使用上下文
ctx := context.Background()
ctx = logger.WithContextFields(ctx, logger.F("trace_id", "xyz-789"))
logger.InfoContext(ctx, "这条日志带有上下文字段")

// 使用自定义上下文日志器
ctxLogger := logger.WithFields(logger.F("component", "auth"))
ctx = logger.WithLogger(ctx, ctxLogger)
logger.InfoContext(ctx, "这条日志使用自定义上下文日志器")

// 在函数间传递上下文
func ProcessRequest(ctx context.Context, req Request) {
    logger.InfoContext(ctx, "开始处理请求")
    // ...
}
```

### 自定义配置

```go
// 创建自定义配置的日志器
customLogger := logger.New(&logger.Config{
    Level:        logger.DebugLevel,
    Output:       os.Stdout,
    EnableCaller: true,
    EnableTime:   true,
    EnableColor:  true,
    TimeFormat:   time.RFC3339,
    CallerSkip:   2,
})
customLogger.Info("这是一条自定义配置的日志")
```

### 多个输出

```go
// 同时输出到控制台和文件
multiWriter := io.MultiWriter(os.Stdout, fileWriter)
multiLogger := logger.WithOutput(multiWriter)
multiLogger.Info("这条日志同时输出到控制台和文件")
```

## 日志级别

日志模块支持以下级别（从低到高）：

- `DEBUG`: 调试信息，用于开发和调试
- `INFO`: 一般信息，表示正常运行状态
- `WARN`: 警告信息，表示可能的问题
- `ERROR`: 错误信息，表示发生了错误但程序仍在运行
- `FATAL`: 致命错误，记录后程序会退出

## 字段

字段可以为日志添加结构化信息：

```go
logger.WithFields(
    logger.F("user_id", 123),
    logger.F("request_id", "abc-123"),
    logger.F("ip", "192.168.1.1"),
).Info("用户登录成功")
```

## 文件输出

### 简单文件输出

```go
fileWriter := logger.NewFileWriter("logs/app.log")
defer fileWriter.Close()
fileLogger := logger.WithOutput(fileWriter)
```

`FileWriter` 支持以下选项：

- `Path`: 日志文件路径
- `MaxSize`: 日志文件最大大小（字节）
- `MaxBackups`: 保留的旧日志文件数量
- `BufferSize`: 缓冲区大小（字节）
- `FlushInterval`: 刷新缓冲区的间隔

### 轮转文件输出

```go
rotatingWriter := logger.NewRotatingFileWriter("logs/rotating.log")
rotatingWriter.MaxSize = 100 * 1024 * 1024 // 100MB
rotatingWriter.MaxBackups = 10
rotatingWriter.MaxAge = 30 // 30天
rotatingWriter.LocalTime = true
rotatingWriter.Compress = true
```

`RotatingFileWriter` 支持以下选项：

- `Path`: 日志文件路径
- `MaxSize`: 日志文件最大大小（字节）
- `MaxBackups`: 保留的旧日志文件数量
- `MaxAge`: 旧日志文件的最大保留天数
- `LocalTime`: 是否使用本地时间
- `Compress`: 是否压缩旧日志文件

## JSON 格式

```go
jsonConfig := logger.DefaultJSONConfig()
jsonConfig.PrettyPrint = false
jsonConfig.TimeKey = "timestamp"
jsonConfig.LevelKey = "severity"
jsonConfig.MessageKey = "msg"
jsonConfig.CallerKey = "source"
jsonLogger := logger.NewJSONLogger(jsonConfig)
```

`JSONConfig` 支持以下选项：

- `Level`: 日志级别
- `Output`: 日志输出
- `Fields`: 默认字段
- `EnableCaller`: 是否启用调用者信息
- `EnableTime`: 是否启用时间信息
- `TimeFormat`: 时间格式
- `CallerSkip`: 调用者跳过级别
- `TimeKey`: 时间字段的键
- `LevelKey`: 级别字段的键
- `MessageKey`: 消息字段的键
- `CallerKey`: 调用者字段的键
- `StacktraceKey`: 堆栈跟踪字段的键
- `PrettyPrint`: 是否启用漂亮打印

## 上下文支持

```go
// 添加字段到上下文
ctx = logger.WithContextFields(ctx, logger.F("trace_id", "xyz-789"))

// 从上下文获取日志器
log := logger.FromContext(ctx)
log.Info("这条日志使用上下文中的日志器")

// 使用上下文日志函数
logger.InfoContext(ctx, "这条日志使用上下文")
logger.ErrorContext(ctx, "发生错误")
```

## 自定义日志器

```go
// 创建自定义日志器
customLogger := logger.New(&logger.Config{
    Level:        logger.DebugLevel,
    Output:       os.Stdout,
    EnableCaller: true,
    EnableTime:   true,
    EnableColor:  true,
    TimeFormat:   time.RFC3339,
    CallerSkip:   2,
})

// 设置为全局默认日志器
logger.SetGlobal(customLogger)
```

## 最佳实践

1. **使用适当的日志级别**：
   - `DEBUG`: 详细的调试信息，仅在开发环境使用
   - `INFO`: 一般信息，表示正常运行状态
   - `WARN`: 警告信息，表示可能的问题
   - `ERROR`: 错误信息，表示发生了错误但程序仍在运行
   - `FATAL`: 致命错误，记录后程序会退出

2. **使用结构化日志**：
   - 使用字段而不是格式化字符串
   - 保持字段名称一致
   - 使用有意义的字段名称

3. **使用上下文传递信息**：
   - 在请求处理的开始创建带有请求 ID 的上下文
   - 在整个请求处理过程中传递上下文
   - 使用 `InfoContext`、`ErrorContext` 等函数记录日志

4. **在生产环境中使用 JSON 格式**：
   - JSON 格式更容易被日志收集和分析工具处理
   - 在开发环境中可以使用彩色控制台输出

5. **配置适当的文件轮转**：
   - 设置合理的文件大小限制
   - 设置合理的备份文件数量
   - 设置合理的备份文件保留时间
