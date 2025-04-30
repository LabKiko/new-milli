# New Milli 配置系统

New Milli 配置系统是一个灵活、可扩展的配置管理解决方案，支持多种配置源和格式。

## 特性

- 支持多种配置源：文件、环境变量、内存等
- 支持多种配置格式：YAML、JSON、TOML
- 支持配置热更新
- 支持配置层级覆盖
- 类型安全的配置访问
- 全局配置管理器

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "log"
    
    "new-milli/config"
)

func main() {
    // 创建文件配置源
    fileSource := config.NewFileSource("config.yaml")
    
    // 创建配置
    cfg := config.NewConfig(fileSource)
    
    // 加载配置
    if err := cfg.Load(); err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // 获取配置值
    appName, err := cfg.GetString("app.name")
    if err != nil {
        appName = "default-app"
    }
    
    fmt.Printf("App name: %s\n", appName)
}
```

### 使用多个配置源

```go
// 创建文件配置源
fileSource := config.NewFileSource("config.yaml")

// 创建环境变量配置源（使用 APP_ 前缀）
envSource := config.NewEnvSource("APP_")

// 创建内存配置源
memorySource := config.NewMemorySource(map[string]interface{}{
    "debug": true,
})

// 创建复合配置源（优先级：内存 > 环境变量 > 文件）
compositeSource := config.NewCompositeSource(fileSource, envSource, memorySource)

// 创建配置
cfg := config.NewConfig(compositeSource)
```

### 监听配置变化

```go
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
```

### 使用全局配置管理器

```go
// 获取全局配置管理器
manager := config.Global()

// 注册配置
manager.Register("app", cfg)

// 获取配置
appConfig := manager.Get("app")

// 加载所有配置
if err := manager.LoadAll(); err != nil {
    log.Fatalf("Failed to load all configurations: %v", err)
}

// 关闭所有配置
if err := manager.CloseAll(); err != nil {
    log.Fatalf("Failed to close all configurations: %v", err)
}
```

## 配置源

### 文件配置源

支持从YAML、JSON、TOML文件读取配置。

```go
// 创建YAML文件配置源
yamlSource := config.NewFileSource("config.yaml")

// 创建JSON文件配置源
jsonSource := config.NewFileSource("config.json", config.WithFormat("json"))

// 创建TOML文件配置源
tomlSource := config.NewFileSource("config.toml", config.WithFormat("toml"))

// 设置文件监视间隔
source := config.NewFileSource("config.yaml", config.WithWatchInterval(10 * time.Second))
```

### 环境变量配置源

支持从环境变量读取配置。

```go
// 创建环境变量配置源（使用 APP_ 前缀）
envSource := config.NewEnvSource("APP_")

// 不使用前缀
envSource := config.NewEnvSource("")
```

环境变量会自动转换为配置键：
- 环境变量名会转换为小写
- 下划线会转换为点
- 前缀会被移除

例如：
- `APP_SERVER_HTTP_PORT=8080` 会转换为 `server.http.port=8080`

### 内存配置源

支持在内存中存储配置。

```go
// 创建内存配置源
memorySource := config.NewMemorySource(map[string]interface{}{
    "debug": true,
    "server": map[string]interface{}{
        "port": 8080,
    },
})

// 动态更新配置
memorySource.Set("debug", false)
memorySource.Delete("server.port")
```

## 配置格式

配置系统支持以下格式：

### YAML

```yaml
app:
  name: "my-app"
  version: "1.0.0"

server:
  http:
    port: 8080
    timeout: "5s"
```

### JSON

```json
{
  "app": {
    "name": "my-app",
    "version": "1.0.0"
  },
  "server": {
    "http": {
      "port": 8080,
      "timeout": "5s"
    }
  }
}
```

### TOML

```toml
[app]
name = "my-app"
version = "1.0.0"

[server.http]
port = 8080
timeout = "5s"
```

## 配置访问

配置系统提供了类型安全的配置访问方法：

```go
// 获取字符串
name, err := cfg.GetString("app.name")

// 获取整数
port, err := cfg.GetInt("server.http.port")

// 获取布尔值
debug, err := cfg.GetBool("debug")

// 获取浮点数
ratio, err := cfg.GetFloat("ratio")

// 获取字符串映射
headers, err := cfg.GetStringMap("headers")

// 获取字符串切片
tags, err := cfg.GetStringSlice("tags")

// 获取字符串到字符串的映射
env, err := cfg.GetStringMapString("env")

// 检查键是否存在
if cfg.Has("app.name") {
    // ...
}
```
