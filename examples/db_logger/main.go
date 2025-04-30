package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"new-milli/connector/mysql"
	"new-milli/connector/postgres"
	"new-milli/logger"
)

// User 是一个示例模型
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:255;not null"`
	Email     string `gorm:"size:255;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	// 创建一个自定义日志器
	customLogger := logger.New(&logger.Config{
		Level:        logger.DebugLevel, // 设置为调试级别，可以看到所有SQL查询
		Output:       os.Stdout,
		EnableCaller: true,
		EnableTime:   true,
		EnableColor:  true,
	})

	// 创建一个JSON格式的文件日志器
	fileWriter := logger.NewFileWriter("logs/db.log")
	defer fileWriter.Close()

	jsonConfig := logger.DefaultJSONConfig()
	jsonConfig.Output = fileWriter
	jsonConfig.PrettyPrint = false
	fileLogger := logger.NewJSONLogger(jsonConfig)

	// 创建一个多输出日志器
	multiLogger := logger.New(&logger.Config{
		Level:        logger.DebugLevel,
		Output:       os.Stdout, // 控制台输出
		EnableCaller: true,
		EnableTime:   true,
		EnableColor:  true,
	}).WithFields(
		logger.F("component", "database"),
		logger.F("app_version", "1.0.0"),
	)

	// 连接MySQL
	fmt.Println("=== 连接MySQL ===")
	mysqlConn := mysql.New(
		mysql.WithAddress("localhost:3306"),
		mysql.WithUsername("root"),
		mysql.WithPassword("password"),
		mysql.WithDatabase("test"),
		mysql.WithLogger(multiLogger.WithFields(logger.F("db", "mysql"))),
		mysql.WithSlowThreshold(time.Millisecond*100), // 将慢查询阈值设置为100毫秒
	)

	// 连接到数据库
	ctx := context.Background()
	if err := mysqlConn.Connect(ctx); err != nil {
		customLogger.Fatalf("无法连接到MySQL: %v", err)
	}
	defer mysqlConn.Disconnect(ctx)

	// 获取GORM数据库实例
	mysqlDB := mysqlConn.(*mysql.Connector).DB()

	// 自动迁移
	if err := mysqlDB.AutoMigrate(&User{}); err != nil {
		customLogger.Errorf("MySQL自动迁移失败: %v", err)
	} else {
		customLogger.Info("MySQL自动迁移成功")
	}

	// 创建用户
	mysqlUser := User{
		Name:  "张三",
		Email: "zhangsan@example.com",
	}
	if err := mysqlDB.Create(&mysqlUser).Error; err != nil {
		customLogger.Errorf("MySQL创建用户失败: %v", err)
	} else {
		customLogger.Infof("MySQL创建用户成功: ID=%d", mysqlUser.ID)
	}

	// 查询用户
	var users []User
	if err := mysqlDB.Find(&users).Error; err != nil {
		customLogger.Errorf("MySQL查询用户失败: %v", err)
	} else {
		customLogger.Infof("MySQL查询到%d个用户", len(users))
		for _, user := range users {
			customLogger.Debugf("用户: ID=%d, 名称=%s, 邮箱=%s", user.ID, user.Name, user.Email)
		}
	}

	// 执行一个慢查询
	customLogger.Info("执行一个慢查询...")
	if err := mysqlDB.Exec("SELECT SLEEP(0.2)").Error; err != nil {
		customLogger.Errorf("MySQL慢查询失败: %v", err)
	}

	// 连接PostgreSQL
	fmt.Println("\n=== 连接PostgreSQL ===")
	pgConn := postgres.New(
		postgres.WithAddress("localhost:5432"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		postgres.WithDatabase("test"),
		postgres.WithLogger(fileLogger.WithFields(logger.F("db", "postgres"))), // 使用JSON文件日志器
		postgres.WithSlowThreshold(time.Millisecond*50),                        // 将慢查询阈值设置为50毫秒
	)

	// 连接到数据库
	if err := pgConn.Connect(ctx); err != nil {
		customLogger.Fatalf("无法连接到PostgreSQL: %v", err)
	}
	defer pgConn.Disconnect(ctx)

	// 获取GORM数据库实例
	pgDB := pgConn.(*postgres.Connector).DB()

	// 自动迁移
	if err := pgDB.AutoMigrate(&User{}); err != nil {
		customLogger.Errorf("PostgreSQL自动迁移失败: %v", err)
	} else {
		customLogger.Info("PostgreSQL自动迁移成功")
	}

	// 创建用户
	pgUser := User{
		Name:  "李四",
		Email: "lisi@example.com",
	}
	if err := pgDB.Create(&pgUser).Error; err != nil {
		customLogger.Errorf("PostgreSQL创建用户失败: %v", err)
	} else {
		customLogger.Infof("PostgreSQL创建用户成功: ID=%d", pgUser.ID)
	}

	// 查询用户
	var pgUsers []User
	if err := pgDB.Find(&pgUsers).Error; err != nil {
		customLogger.Errorf("PostgreSQL查询用户失败: %v", err)
	} else {
		customLogger.Infof("PostgreSQL查询到%d个用户", len(pgUsers))
		for _, user := range pgUsers {
			customLogger.Debugf("用户: ID=%d, 名称=%s, 邮箱=%s", user.ID, user.Name, user.Email)
		}
	}

	// 执行一个慢查询
	customLogger.Info("执行一个慢查询...")
	if err := pgDB.Exec("SELECT pg_sleep(0.1)").Error; err != nil {
		customLogger.Errorf("PostgreSQL慢查询失败: %v", err)
	}

	// 使用上下文传递日志器
	ctxWithLogger := logger.WithLogger(ctx, customLogger.WithFields(logger.F("request_id", "123456")))

	// 在其他函数中使用上下文中的日志器
	processUser(ctxWithLogger, mysqlDB)
}

// processUser 处理用户数据
func processUser(ctx context.Context, db interface{}) {
	// 从上下文获取日志器
	log := logger.FromContext(ctx)

	log.Info("开始处理用户数据")

	// 简化处理，不实际操作数据库
	log.Info("模拟数据库操作")
	log.WithFields(logger.F("db_type", "mysql")).Info("用户数据处理完成")
}
