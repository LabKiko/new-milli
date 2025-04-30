package main

import (
	"context"
	"io"
	"os"
	"time"

	"new-milli/logger"
)

func main() {
	// 使用默认日志器
	logger.Info("这是一条信息日志")
	logger.Warn("这是一条警告日志")
	logger.Error("这是一条错误日志")
	logger.Debug("这是一条调试日志") // 默认不会显示，因为默认级别是 INFO

	// 使用字段
	logger.WithFields(
		logger.F("user_id", 123),
		logger.F("request_id", "abc-123"),
	).Info("用户登录成功")

	// 修改日志级别
	debugLogger := logger.WithLevel(logger.DebugLevel)
	debugLogger.Debug("现在可以看到调试日志了")

	// 禁用颜色
	noColorLogger := logger.WithColor(false)
	noColorLogger.Info("这条日志没有颜色")

	// 禁用时间和调用者信息
	simpleLogger := logger.WithTime(false).WithCaller(false)
	simpleLogger.Info("这是一条简单的日志")

	// 输出到文件
	fileWriter := logger.NewFileWriter("logs/app.log")
	defer fileWriter.Close()
	fileLogger := logger.WithOutput(fileWriter)
	fileLogger.Info("这条日志写入到文件")

	// 使用 JSON 格式
	jsonLogger := logger.NewJSONLogger(nil)
	jsonLogger.Info("这是一条 JSON 格式的日志")

	// 带有漂亮打印的 JSON 格式
	prettyConfig := logger.DefaultJSONConfig()
	prettyConfig.PrettyPrint = true
	prettyLogger := logger.NewJSONLogger(prettyConfig)
	prettyLogger.WithFields(
		logger.F("user_id", 456),
		logger.F("request_id", "def-456"),
	).Info("这是一条漂亮打印的 JSON 日志")

	// 使用上下文
	ctx := context.Background()
	ctx = logger.WithContextFields(ctx, logger.F("trace_id", "xyz-789"))
	logger.InfoContext(ctx, "这条日志带有上下文字段")

	// 使用自定义上下文日志器
	ctxLogger := logger.WithFields(logger.F("component", "auth"))
	ctx = logger.WithLogger(ctx, ctxLogger)
	logger.InfoContext(ctx, "这条日志使用自定义上下文日志器")

	// 使用轮转文件
	rotatingWriter := logger.NewRotatingFileWriter("logs/rotating.log")
	rotatingWriter.MaxSize = 1024 // 1KB，方便演示
	rotatingWriter.MaxBackups = 3
	rotatingWriter.MaxAge = 1
	rotatingLogger := logger.WithOutput(rotatingWriter)

	// 写入足够的日志以触发轮转
	for i := 0; i < 100; i++ {
		rotatingLogger.Infof("这是第 %d 条日志，用于测试文件轮转", i)
	}
	rotatingWriter.Close()

	// 多个输出
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)
	multiLogger := logger.WithOutput(multiWriter)
	multiLogger.Info("这条日志同时输出到控制台和文件")

	// 使用不同的时间格式
	customTimeLogger := logger.New(&logger.Config{
		TimeFormat: time.RFC822,
		Output:     os.Stdout,
	})
	customTimeLogger.Info("这条日志使用自定义时间格式")

	// 使用不同的调用者跳过级别
	callerLogger := logger.New(&logger.Config{
		CallerSkip: 3,
		Output:     os.Stdout,
	})
	logWithCustomCaller(callerLogger)
}

func logWithCustomCaller(log logger.Logger) {
	log.Info("这条日志显示不同的调用者信息")
}
