package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud-system/internal/api"
	"cloud-system/internal/config"
	"cloud-system/internal/mqtt"
	"cloud-system/internal/repository/postgres"
	"cloud-system/internal/repository/redis"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/internal/websocket"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	// 配置文件路径可以通过以下方式指定：
	// 1. 环境变量：export CLOUD_CONFIG_PATH=/path/to/config.yaml
	// 2. 默认路径：自动查找 Cloud/config.yaml
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志系统
	if err := utils.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer utils.Sync()

	utils.Info("Starting Cloud System Server...")

	// 初始化PostgreSQL
	pgClient, err := postgres.NewClient(cfg)
	if err != nil {
		utils.Fatal("Failed to initialize PostgreSQL", zap.Error(err))
	}
	defer pgClient.Close()

	// 运行数据库迁移
	ctx := context.Background()
	if err := pgClient.RunMigrations(ctx, "migrations"); err != nil {
		utils.Warn("Database migration completed with warnings", zap.Error(err))
	}

	// 初始化Redis（可选）
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		utils.Warn("Redis connection failed (optional, continuing without cache)", zap.Error(err))
		redisClient = nil // 继续运行，但缓存功能不可用
	} else {
		defer redisClient.Close()
		utils.Info("Redis connection established")
	}

	// 初始化MQTT（Cloud端自己的MQTT broker）
	mqttClient, err := mqtt.NewClient(cfg)
	if err != nil {
		utils.Fatal("Failed to initialize MQTT", zap.Error(err))
	}
	defer mqttClient.Close()

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建Gin路由器
	router := gin.New()

	// 初始化WebSocket Hub
	wsHub := websocket.NewHub(utils.GetLogger())
	go wsHub.Run()
	utils.Info("WebSocket Hub started")

	// 初始化Edge端MQTT客户端（如果启用），用于下发命令和订阅数据
	var edgeMQTTClient *mqtt.EdgeClient
	if cfg.EdgeMQTT.Enabled {
		// 创建Edge端MQTT客户端
		var err error
		edgeMQTTClient, err = mqtt.NewEdgeClient(cfg)
		if err != nil {
			utils.Warn("Failed to initialize Edge MQTT client, sensor data subscription disabled",
				zap.Error(err),
			)
		} else {
			defer edgeMQTTClient.Close()
			utils.Info("Edge MQTT client initialized successfully")
		}
	} else {
		utils.Info("Edge MQTT subscription disabled")
	}

	// 设置路由（这里会创建传感器服务等，传入edgeMQTTClient用于告警解决命令下发）
	sensorService, trafficService, alertService := api.SetupRoutes(router, cfg, pgClient, mqttClient, wsHub, edgeMQTTClient)

	// 如果Edge MQTT已启用且客户端初始化成功，创建MQTT订阅服务
	if cfg.EdgeMQTT.Enabled && edgeMQTTClient != nil {
		// 初始化Repository（用于MQTT订阅服务）
		sensorDeviceRepo := postgres.NewSensorDeviceRepo(pgClient.GetPool())

		// 创建MQTT订阅服务（传入WebSocket Hub）
		edgeMQTTSubscriber := services.NewMQTTSubscriberService(
			edgeMQTTClient.GetClient(),
			sensorService,
			sensorDeviceRepo,
			trafficService,
			cfg,
			wsHub,
		)

		// 设置告警服务（用于处理MQTT实时告警）
		edgeMQTTSubscriber.SetAlertService(alertService)

		// 【修复】设置ABAC日志处理器（用于处理策略分发ACK）
		policyRepo := postgres.NewPolicyRepo(pgClient.GetPool())
		abacLogHandler := services.NewABACLogHandler(policyRepo)
		edgeMQTTSubscriber.SetABACLogHandler(abacLogHandler)
		utils.Info("ABAC log handler configured for policy ACK processing")

		// 启动订阅
		if err := edgeMQTTSubscriber.Start(); err != nil {
			utils.Warn("Failed to start Edge MQTT subscriber",
				zap.Error(err),
			)
		} else {
			utils.Info("Edge MQTT subscriber started successfully")
		}

		// 停止MQTT订阅服务
		defer func() {
			if err := edgeMQTTSubscriber.Stop(); err != nil {
				utils.Warn("Failed to stop Edge MQTT subscriber", zap.Error(err))
			}
		}()
	}

	// 启动配置热重载监听
	go config.WatchConfig(func(newCfg *config.Config) {
		utils.Info("Configuration reloaded, updating logger level...")
		utils.InitLogger(newCfg.Logging.Level, newCfg.Logging.Format, newCfg.Logging.Output)
	})

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// 在goroutine中启动服务器
	go func() {
		utils.Info(fmt.Sprintf("Server starting on %s", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	utils.Info("Shutting down server...")

	// 优雅关闭，等待5秒
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		utils.Fatal("Server forced to shutdown", zap.Error(err))
	}

	utils.Info("Server exited")
}
