package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"
	"new-milli/connector/mysql"
	"new-milli/connector/postgres"
	"new-milli/logger"
)

// User 是一个示例模型
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:255;not null"`
	Email     string    `gorm:"size:255;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Order 是一个示例模型
type Order struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	Amount    float64   `gorm:"not null"`
	Status    string    `gorm:"size:50;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 模拟服务名称
const (
	ServiceName = "order-service"
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
	fileWriter := logger.NewFileWriter("logs/db_trace.log")
	defer fileWriter.Close()
	
	jsonConfig := logger.DefaultJSONConfig()
	jsonConfig.Output = fileWriter
	jsonConfig.PrettyPrint = false
	jsonLogger := logger.NewJSONLogger(jsonConfig).
		WithServiceName(ServiceName).
		WithEnvironment(Environment)

	// 设置全局日志器
	logger.SetGlobal(log)

	// 创建一个带有跟踪信息的上下文
	ctx := context.Background()
	traceInfo := logger.NewTraceInfo().
		WithRequestID("req-12345").
		WithTraceID("trace-67890").
		WithSpanID("span-abcdef").
		WithServiceName(ServiceName).
		WithEnvironment(Environment).
		WithCustomField("user_id", "user-123")
	
	ctx = logger.WithTraceInfo(ctx, traceInfo)

	// 连接MySQL
	fmt.Println("=== 连接MySQL ===")
	mysqlConn := mysql.New(
		mysql.WithAddress("localhost:3306"),
		mysql.WithUsername("root"),
		mysql.WithPassword("password"),
		mysql.WithDatabase("test"),
		mysql.WithLogger(log.WithFields(logger.F("db", "mysql"))),
		mysql.WithSlowThreshold(time.Millisecond*100), // 将慢查询阈值设置为100毫秒
	)

	// 连接到数据库
	if err := mysqlConn.Connect(ctx); err != nil {
		logger.ErrorWithTrace(ctx, "无法连接到MySQL", logger.F("error", err))
		return
	}
	defer mysqlConn.Disconnect(ctx)

	// 获取GORM数据库实例
	mysqlDB := mysqlConn.(*mysql.Connector).DB()

	// 模拟处理订单
	processOrder(ctx, mysqlDB, 123, 99.99)

	// 连接PostgreSQL
	fmt.Println("\n=== 连接PostgreSQL ===")
	pgConn := postgres.New(
		postgres.WithAddress("localhost:5432"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		postgres.WithDatabase("test"),
		postgres.WithLogger(jsonLogger.WithFields(logger.F("db", "postgres"))), // 使用JSON文件日志器
		postgres.WithSlowThreshold(time.Millisecond*50), // 将慢查询阈值设置为50毫秒
	)

	// 连接到数据库
	if err := pgConn.Connect(ctx); err != nil {
		logger.ErrorWithTrace(ctx, "无法连接到PostgreSQL", logger.F("error", err))
		return
	}
	defer pgConn.Disconnect(ctx)

	// 获取GORM数据库实例
	pgDB := pgConn.(*postgres.Connector).DB()

	// 模拟处理用户
	processUser(ctx, pgDB, "张三", "zhangsan@example.com")
}

// processOrder 处理订单
func processOrder(ctx context.Context, db *gorm.DB, userID uint, amount float64) {
	// 创建一个子跨度
	childCtx := logger.WithChildSpan(ctx)
	
	// 记录处理开始
	logger.InfoWithTrace(childCtx, "开始处理订单", 
		logger.F("user_id", userID),
		logger.F("amount", amount),
	)
	
	// 模拟自动迁移
	logger.DebugWithTrace(childCtx, "执行订单表自动迁移")
	if err := db.WithContext(childCtx).AutoMigrate(&Order{}); err != nil {
		logger.ErrorWithTrace(childCtx, "订单表自动迁移失败", logger.F("error", err))
		return
	}
	
	// 模拟创建订单
	order := Order{
		UserID: userID,
		Amount: amount,
		Status: "pending",
	}
	
	logger.DebugWithTrace(childCtx, "创建订单记录", 
		logger.F("user_id", order.UserID),
		logger.F("amount", order.Amount),
	)
	
	if err := db.WithContext(childCtx).Create(&order).Error; err != nil {
		logger.ErrorWithTrace(childCtx, "创建订单失败", logger.F("error", err))
		return
	}
	
	logger.InfoWithTrace(childCtx, "订单创建成功", logger.F("order_id", order.ID))
	
	// 模拟查询订单
	var orders []Order
	logger.DebugWithTrace(childCtx, "查询用户订单", logger.F("user_id", userID))
	
	if err := db.WithContext(childCtx).Where("user_id = ?", userID).Find(&orders).Error; err != nil {
		logger.ErrorWithTrace(childCtx, "查询订单失败", logger.F("error", err))
		return
	}
	
	logger.InfoWithTrace(childCtx, "查询订单成功", 
		logger.F("user_id", userID),
		logger.F("order_count", len(orders)),
	)
	
	// 模拟更新订单
	logger.DebugWithTrace(childCtx, "更新订单状态", 
		logger.F("order_id", order.ID),
		logger.F("status", "completed"),
	)
	
	if err := db.WithContext(childCtx).Model(&order).Update("status", "completed").Error; err != nil {
		logger.ErrorWithTrace(childCtx, "更新订单失败", logger.F("error", err))
		return
	}
	
	logger.InfoWithTrace(childCtx, "订单处理完成", logger.F("order_id", order.ID))
}

// processUser 处理用户
func processUser(ctx context.Context, db *gorm.DB, name, email string) {
	// 创建一个子跨度
	childCtx := logger.WithChildSpan(ctx)
	
	// 记录处理开始
	logger.InfoWithTrace(childCtx, "开始处理用户", 
		logger.F("name", name),
		logger.F("email", email),
	)
	
	// 模拟自动迁移
	logger.DebugWithTrace(childCtx, "执行用户表自动迁移")
	if err := db.WithContext(childCtx).AutoMigrate(&User{}); err != nil {
		logger.ErrorWithTrace(childCtx, "用户表自动迁移失败", logger.F("error", err))
		return
	}
	
	// 模拟创建用户
	user := User{
		Name:  name,
		Email: email,
	}
	
	logger.DebugWithTrace(childCtx, "创建用户记录", 
		logger.F("name", user.Name),
		logger.F("email", user.Email),
	)
	
	if err := db.WithContext(childCtx).Create(&user).Error; err != nil {
		logger.ErrorWithTrace(childCtx, "创建用户失败", logger.F("error", err))
		return
	}
	
	logger.InfoWithTrace(childCtx, "用户创建成功", logger.F("user_id", user.ID))
	
	// 模拟查询用户
	var users []User
	logger.DebugWithTrace(childCtx, "查询用户", logger.F("email", email))
	
	if err := db.WithContext(childCtx).Where("email = ?", email).Find(&users).Error; err != nil {
		logger.ErrorWithTrace(childCtx, "查询用户失败", logger.F("error", err))
		return
	}
	
	logger.InfoWithTrace(childCtx, "查询用户成功", 
		logger.F("email", email),
		logger.F("user_count", len(users)),
	)
	
	// 模拟更新用户
	logger.DebugWithTrace(childCtx, "更新用户名称", 
		logger.F("user_id", user.ID),
		logger.F("name", name+" (已更新)"),
	)
	
	if err := db.WithContext(childCtx).Model(&user).Update("name", name+" (已更新)").Error; err != nil {
		logger.ErrorWithTrace(childCtx, "更新用户失败", logger.F("error", err))
		return
	}
	
	logger.InfoWithTrace(childCtx, "用户处理完成", logger.F("user_id", user.ID))
}
