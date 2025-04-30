package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"new-milli"
	"new-milli/config"
	"new-milli/middleware/logging"
	"new-milli/middleware/recovery"
	"new-milli/middleware/tracing"
	"new-milli/transport"
	"new-milli/transport/http"
)

func main() {
	// 创建配置管理器
	manager := config.Global()

	// 创建文件配置源
	fileSource := config.NewFileSource("examples/config/config.yaml")
	
	// 创建环境变量配置源（使用 APP_ 前缀）
	envSource := config.NewEnvSource("APP_")
	
	// 创建复合配置源（先读取文件，再读取环境变量，环境变量会覆盖文件中的配置）
	compositeSource := config.NewCompositeSource(fileSource, envSource)
	
	// 创建配置
	cfg := config.NewConfig(compositeSource)
	
	// 注册配置
	manager.Register("app", cfg)
	
	// 加载配置
	if err := cfg.Load(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// 监听配置变化
	watchCh, err := cfg.Watch()
	if err != nil {
		log.Fatalf("Failed to watch configuration: %v", err)
	}
	
	// 在后台处理配置变化
	go func() {
		for range watchCh {
			log.Println("Configuration changed, reloading...")
			if err := cfg.Load(); err != nil {
				log.Printf("Failed to reload configuration: %v", err)
			}
		}
	}()
	
	// 获取配置值
	appName, err := cfg.GetString("app.name")
	if err != nil {
		appName = "default-app"
	}
	
	appVersion, err := cfg.GetString("app.version")
	if err != nil {
		appVersion = "1.0.0"
	}
	
	httpAddress, err := cfg.GetString("server.http.address")
	if err != nil {
		httpAddress = ":8000"
	}
	
	httpTimeoutStr, err := cfg.GetString("server.http.timeout")
	if err != nil {
		httpTimeoutStr = "5s"
	}
	
	httpTimeout, err := time.ParseDuration(httpTimeoutStr)
	if err != nil {
		httpTimeout = 5 * time.Second
	}
	
	// 创建 HTTP 服务器
	httpServer := http.NewServer(
		transport.Address(httpAddress),
		transport.Timeout(httpTimeout),
		transport.Middleware(
			recovery.Server(),
			tracing.Server(),
			logging.Server(),
		),
	)
	
	// 获取 Hertz 服务器实例
	hertzServer := httpServer.GetHertzServer()
	
	// 注册路由
	hertzServer.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "Welcome to %s %s!", appName, appVersion)
	})
	
	hertzServer.GET("/config", func(ctx context.Context, c *app.RequestContext) {
		key := c.Query("key")
		if key == "" {
			c.String(400, "Missing key parameter")
			return
		}
		
		value, err := cfg.Get(key)
		if err != nil {
			c.String(404, "Key not found: %s", key)
			return
		}
		
		c.JSON(200, map[string]interface{}{
			"key":   key,
			"value": value,
		})
	})
	
	// 创建应用
	app, err := newMilli.New(
		newMilli.Name(appName),
		newMilli.Version(appVersion),
		newMilli.Server(httpServer),
		newMilli.BeforeStart(func(ctx context.Context) error {
			fmt.Printf("Starting %s %s...\n", appName, appVersion)
			return nil
		}),
		newMilli.AfterStart(func(ctx context.Context) error {
			fmt.Printf("%s %s started successfully!\n", appName, appVersion)
			return nil
		}),
		newMilli.BeforeStop(func(ctx context.Context) error {
			fmt.Printf("Stopping %s %s...\n", appName, appVersion)
			return nil
		}),
		newMilli.AfterStop(func(ctx context.Context) error {
			fmt.Printf("%s %s stopped successfully!\n", appName, appVersion)
			
			// 关闭配置
			if err := manager.CloseAll(); err != nil {
				return err
			}
			
			return nil
		}),
	)
	
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}
	
	// 运行应用
	if err := app.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
		os.Exit(1)
	}
}
