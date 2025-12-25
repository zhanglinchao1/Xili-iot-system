/*
 * Edge系统主程序入口
 * 储能柜边缘认证网关主程序
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edge/storage-cabinet/api"
	"github.com/edge/storage-cabinet/internal/abac"
	"github.com/edge/storage-cabinet/internal/activation"
	"github.com/edge/storage-cabinet/internal/api/handlers"
	"github.com/edge/storage-cabinet/internal/auth"
	"github.com/edge/storage-cabinet/internal/cloud"
	"github.com/edge/storage-cabinet/internal/collector"
	"github.com/edge/storage-cabinet/internal/config"
	"github.com/edge/storage-cabinet/internal/device"
	"github.com/edge/storage-cabinet/internal/license"
	"github.com/edge/storage-cabinet/internal/mqtt"
	"github.com/edge/storage-cabinet/internal/storage"
	"github.com/edge/storage-cabinet/internal/sync"
	"github.com/edge/storage-cabinet/internal/vulnerability"
	"github.com/edge/storage-cabinet/internal/zkp"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// 解析命令行参数
	var (
		configFile  = flag.String("config", "./configs/config.yaml", "配置文件路径")
		showVersion = flag.Bool("version", false, "显示版本信息")
		migrate     = flag.Bool("migrate", false, "执行数据库迁移")
	)
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Printf("Edge System v%s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		os.Exit(0)
	}

	// 初始化日志
	logger := initLogger()
	defer logger.Sync()

	logger.Info("启动Edge系统",
		zap.String("version", version),
		zap.String("config", *configFile),
	)

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.Fatal("加载配置失败", zap.Error(err))
	}

	// 初始化数据库
	db, err := storage.NewSQLiteDB(cfg.Database, logger)
	if err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer db.Close()

	// 执行数据库迁移
	if *migrate {
		if err := db.Migrate(); err != nil {
			logger.Fatal("数据库迁移失败", zap.Error(err))
		}
		logger.Info("数据库迁移完成")
		return
	}

	// 初始化ZKP验证器（使用预生成的 verifying key）
	zkpVerifier := zkp.NewVerifier(logger)
	vkPath := cfg.Auth.ZKP.VerifyingKeyPath
	if vkPath == "" {
		vkPath = "./auth_verifying.key" // 默认路径
	}
	if err := zkpVerifier.InitializeWithKeyPath(vkPath); err != nil {
		logger.Fatal("初始化ZKP验证器失败", zap.Error(err))
	}

	// 【SPA单包授权】初始化许可证服务
	var licenseService *license.Service
	if cfg.License.Enabled {
		licenseService, err = license.NewService(
			cfg.License.Enabled,
			cfg.License.Path,
			cfg.License.PubKeyPath,
			cfg.License.GracePeriod,
			logger,
		)
		if err != nil {
			logger.Fatal("初始化许可证服务失败", zap.Error(err))
		}
		logger.Info("许可证服务已启用",
			zap.Int("max_devices", licenseService.GetMaxDevices()))
	} else {
		// 开发模式：创建一个禁用状态的许可证服务实例，避免nil pointer
		licenseService, _ = license.NewService(
			false, // enabled = false
			"",    // 不需要许可证文件路径
			"",    // 不需要公钥路径
			0,     // 不需要宽限期
			logger,
		)
		logger.Info("许可证验证已禁用（开发模式）")
	}

	// 【自动激活】尝试自动激活储能柜
	activationService := activation.NewService(cfg, db, logger)
	if err := activationService.TryAutoActivate(context.Background()); err != nil {
		logger.Warn("自动激活失败（非致命错误）", zap.Error(err))
		// 不中断启动流程，允许系统继续运行
	} else {
		logger.Info("储能柜激活成功，API凭证已保存到数据库")
		// 注意:不再需要重新加载配置文件,因为凭证保存在数据库中
	}

	// 初始化服务（传入许可证服务）
	authService := auth.NewService(cfg.Auth, db, zkpVerifier, licenseService, logger)
	deviceManager := device.NewManager(cfg.Device, db, licenseService, logger, cfg.Cloud.CabinetID)
	dataCollector := collector.NewService(cfg.Data, cfg.Alert, db, deviceManager, logger)
	// 使用配置文件中的sync_interval，避免过于频繁的同步
	// 传递db以便从数据库读取API凭证
	cloudSync := sync.NewCloudSync(cfg.Cloud, db, db, logger, cfg.Data.SyncInterval)

	// 注入CloudSync到数据采集服务(用于即时告警上报)
	dataCollector.SetCloudSync(cloudSync)

	// 初始化脆弱性评估服务
	vulnConfig := convertVulnerabilityConfig(cfg)
	vulnService := vulnerability.NewService(vulnConfig, db.GetDB(), deviceManager, licenseService, logger)

	// 注入CloudSync到脆弱性评估服务(用于同步评估结果到Cloud端)
	vulnService.SetCloudSync(cloudSync)

	// 【ABAC设备权限管理】初始化
	abacRepo, err := abac.NewSQLiteRepository(db.GetDB())
	if err != nil {
		logger.Warn("初始化ABAC存储失败", zap.Error(err))
	} else {
		logger.Info("ABAC设备权限管理已初始化")
	}
	var abacMQTTHandler *abac.MQTTHandler
	if abacRepo != nil && cfg.Cloud.CabinetID != "" {
		abacMQTTHandler = abac.NewMQTTHandler(abacRepo, cfg.Cloud.CabinetID)
	}

	// 启动后台服务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动设备管理器
	if err := deviceManager.Start(ctx); err != nil {
		logger.Fatal("启动设备管理器失败", zap.Error(err))
	}

	// 启动数据采集服务
	if err := dataCollector.Start(ctx); err != nil {
		logger.Fatal("启动数据采集服务失败", zap.Error(err))
	}

	// 启动脆弱性评估服务
	if err := vulnService.Start(ctx); err != nil {
		logger.Fatal("启动脆弱性评估服务失败", zap.Error(err))
	}

	// 启动 MQTT 订阅器（新增）
	var mqttSubscriber *mqtt.Subscriber
	var trafficPublisher *mqtt.TrafficPublisher
	if cfg.MQTT.Enabled {
		var commandClient *cloud.CommandClient
		if cfg.Cloud.Enabled {
			commandClient = cloud.NewCommandClient(cfg.Cloud, logger)
		}

		mqttSubscriber = mqtt.NewSubscriber(
			cfg.MQTT,
			dataCollector,
			deviceManager,
			licenseService,
			commandClient,
			logger,
			cfg.Cloud.CabinetID,
		)

		// 注入ABAC策略处理器
		if abacMQTTHandler != nil {
			mqttSubscriber.SetABACHandler(abacMQTTHandler)
		}

		if err := mqttSubscriber.Start(ctx); err != nil {
			logger.Fatal("启动 MQTT 订阅器失败", zap.Error(err))
		}

		// 将MQTT统计数据注入到脆弱性服务
		vulnService.SetMQTTStats(mqttSubscriber.GetStats())

		// 启动流量发布器
		trafficPublisher = mqtt.NewTrafficPublisher(cfg.MQTT, logger)
		if err := trafficPublisher.Start(); err != nil {
			logger.Warn("启动流量发布器失败", zap.Error(err))
		} else {
			vulnService.SetTrafficPublisher(trafficPublisher)
		}

		logger.Info("MQTT 订阅器已启动",
			zap.String("broker", cfg.MQTT.BrokerAddress),
			zap.String("client_id", cfg.MQTT.ClientID))

		// 【告警实时推送】创建告警MQTT发布器并注入到数据采集服务
		if cfg.Cloud.Enabled && cfg.Cloud.CabinetID != "" {
			alertPublisher := sync.NewAlertPublisher(
				mqttSubscriber.GetMQTTClient(),
				cfg.Cloud.CabinetID,
				logger,
			)
			dataCollector.SetAlertPublisher(alertPublisher)
			logger.Info("告警MQTT发布器已注入到数据采集服务")
		}

		// 【ABAC功能】配置MQTT发布函数和启动日志同步
		if abacRepo != nil && cfg.Cloud.Enabled && cfg.Cloud.CabinetID != "" {
			// 获取MQTT发布函数
			publishFunc := func(topic string, payload []byte) error {
				token := mqttSubscriber.GetMQTTClient().Publish(topic, 1, false, payload)
				token.Wait()
				return token.Error()
			}

			// 设置ABAC MQTT Handler的发布函数(用于发送ACK)
			if abacMQTTHandler != nil {
				abacMQTTHandler.SetPublishFunc(publishFunc)
			}

			// 启动ABAC日志同步服务
			logSyncService := abac.NewLogSyncService(abacRepo, cfg.Cloud.CabinetID, publishFunc)
			go logSyncService.StartPeriodicSync(ctx, 5*time.Minute)
			logger.Info("ABAC日志同步服务已启动", zap.Duration("interval", 5*time.Minute))
		}
	}

	// 启动云端同步服务
	if cfg.Cloud.Enabled {
		if err := cloudSync.Start(ctx); err != nil {
			logger.Fatal("启动云端同步服务失败", zap.Error(err))
		}

		// 启动Token续签服务（传入数据库实例以支持从数据库读取endpoint）
		tokenRefreshService := cloud.NewTokenRefreshService(cfg, db, logger)
		if err := tokenRefreshService.Start(); err != nil {
			logger.Warn("启动Token续签服务失败", zap.Error(err))
		} else {
			defer tokenRefreshService.Stop()
			logger.Info("Token续签服务已启动")
		}
	}

	// 初始化HTTP服务器
	router := setupRouter(cfg, authService, deviceManager, dataCollector, db, mqttSubscriber, licenseService, vulnService, abacRepo, cloudSync, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动HTTP服务器
	go func() {
		logger.Info("HTTP服务器启动",
			zap.String("address", srv.Addr),
			zap.String("mode", cfg.Server.Mode),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("收到停止信号，正在关闭服务...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 停止HTTP服务器
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP服务器关闭失败", zap.Error(err))
	}

	// 停止 MQTT 订阅器
	if mqttSubscriber != nil {
		mqttSubscriber.Stop()
		logger.Info("MQTT 订阅器已停止")
	}
	if trafficPublisher != nil {
		trafficPublisher.Stop()
	}

	// 停止脆弱性评估服务
	if err := vulnService.Stop(); err != nil {
		logger.Error("停止脆弱性评估服务失败", zap.Error(err))
	} else {
		logger.Info("脆弱性评估服务已停止")
	}

	// 停止后台服务
	cancel()

	// 等待所有服务停止
	time.Sleep(2 * time.Second)

	logger.Info("Edge系统已停止")
}

// initLogger 初始化日志系统
func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 设置日志级别
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	// 输出到控制台和文件
	config.OutputPaths = []string{"stdout", "./logs/edge.log"}
	config.ErrorOutputPaths = []string{"stderr", "./logs/edge_error.log"}

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}

	return logger
}

// setupRouter 设置路由
func setupRouter(
	cfg *config.Config,
	authService *auth.Service,
	deviceManager *device.Manager,
	dataCollector *collector.Service,
	db *storage.SQLiteDB,
	mqttSubscriber *mqtt.Subscriber,
	licenseService *license.Service,
	vulnService *vulnerability.Service,
	abacRepo abac.Repository,
	cloudSync api.CloudSyncInterface,
	logger *zap.Logger,
) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	router := gin.New()

	// 中间件
	router.Use(gin.Recovery())
	router.Use(api.ErrorHandlerMiddleware(logger))
	router.Use(api.LoggerMiddleware(logger))
	router.Use(api.CORSMiddleware())
	router.Use(api.RequestIDMiddleware())
	router.Use(api.RateLimitMiddleware(500, time.Minute)) // 每分钟最多500次请求

	// 健康检查
	router.GET("/health", api.HealthCheck)
	router.GET("/ready", api.ReadyCheck)

	// WebSocket端点（用于实时数据推送）
	if mqttSubscriber != nil {
		wsHub := mqttSubscriber.GetWebSocketHub()
		if wsHub != nil {
			router.GET("/ws", func(c *gin.Context) {
				wsHub.HandleWebSocket(c.Writer, c.Request)
			})
			logger.Info("WebSocket端点已注册", zap.String("path", "/ws"))
		} else {
			logger.Warn("WebSocket Hub未初始化")
		}
	} else {
		logger.Warn("MQTT订阅器未初始化，WebSocket端点不可用")
	}

	// 静态文件服务
	router.Static("/static", "./web")
	router.StaticFile("/", "./web/index.html")

	// API路由
	v1 := router.Group("/api/v1")
	{
		// 认证相关
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/challenge", api.GetChallenge(authService))
			authGroup.POST("/verify", api.VerifyProof(authService))
			authGroup.POST("/refresh", api.RefreshSession(authService))
		}

		// 设备管理（无需认证，用于Web管理界面）
		deviceGroup := v1.Group("/devices")
		{
			deviceGroup.GET("", api.ListDevices(deviceManager))
			deviceGroup.GET("/statistics", api.GetDeviceStatistics(deviceManager))
			deviceGroup.GET("/:id", api.GetDevice(deviceManager))
			deviceGroup.GET("/:id/latest-data", api.GetDeviceLatestData(dataCollector))
			deviceGroup.POST("", api.RegisterDevice(deviceManager))
			deviceGroup.PUT("/:id", api.UpdateDevice(deviceManager))
			deviceGroup.DELETE("/:id", api.UnregisterDevice(deviceManager))
			deviceGroup.POST("/:id/heartbeat", api.DeviceHeartbeat(deviceManager))
		}

		// 储能柜管理（无需认证，用于云端同步）
		cabinetGroup := v1.Group("/cabinets")
		{
			cabinetGroup.GET("", api.GetCabinetList(deviceManager))
			cabinetGroup.GET("/:cabinet_id/devices", api.GetDevicesByCabinet(deviceManager))
			// 保存储能柜信息（前端调用，后端自动同步到Cloud）
			cabinetGroup.PUT("/info", api.SaveCabinetInfo(cfg, cloudSync))
		}

		// 数据采集
		dataGroup := v1.Group("/data")
		{
			// 数据上传需要认证，可选启用设备ABAC权限控制
			if abacRepo != nil && cfg.ABAC.Enabled {
				// 启用ABAC: 设备认证中间件 + ABAC权限评估中间件 + ZKP认证
				deviceAuthMiddleware := abac.NewDeviceAuthMiddleware()
				deviceABACMiddleware := abac.NewDeviceABACMiddleware(abacRepo)
				dataGroup.POST("/collect",
					deviceAuthMiddleware.Handle(),
					deviceABACMiddleware.Handle(),
					api.AuthMiddleware(authService),
					api.CollectData(dataCollector))
				logger.Info("数据采集端点已启用ABAC设备权限控制")
			} else {
				// 仅ZKP认证
				dataGroup.POST("/collect", api.AuthMiddleware(authService), api.CollectData(dataCollector))
			}

			// 查询和统计无需认证（Web管理界面使用）
			dataGroup.GET("/query", api.QueryData(dataCollector))
			dataGroup.GET("/statistics", api.GetStatistics(dataCollector))
		}

		// 告警（无需认证，用于Web管理界面）
		alertGroup := v1.Group("/alerts")
		{
			alertGroup.GET("", api.ListAlerts(dataCollector))
			alertGroup.POST("", api.CreateAlert(dataCollector))
			alertGroup.PUT("/:id/resolve", api.ResolveAlert(dataCollector))
			alertGroup.GET("/config", api.GetAlertConfig(cfg))
		}

		// 日志查询（无需认证，用于Web管理界面）
		logGroup := v1.Group("/logs")
		{
			logGroup.GET("/alerts", api.GetAlertLogs(db))
			logGroup.GET("/auth", api.GetAuthLogs(db))
			logGroup.DELETE("/alerts/batch", api.BatchDeleteAlertLogs(db))
			logGroup.DELETE("/auth/batch", api.BatchDeleteAuthLogs(db))
			logGroup.DELETE("/auth/clear", api.ClearAllAuthLogs(db))
		}

		// 许可证信息（无需认证，用于Web管理界面）
		v1.GET("/license/info", api.GetLicenseInfo(licenseService))

		// 系统配置和信息（无需认证，用于Web管理界面）
		systemGroup := v1.Group("/system")
		{
			systemGroup.GET("/mac", api.GetSystemMAC())
			systemGroup.GET("/ip", api.GetSystemIP())
		}

		// 配置信息（无需认证，用于Web管理界面）
		// API Key从数据库读取和保存，其他配置从配置文件读取
		v1.GET("/config", api.GetConfig(cfg, db))
		v1.PUT("/config", api.UpdateConfig(cfg, db))
		v1.GET("/config/test-cloud", api.TestCloudConnection(cfg, db)) // 测试Cloud连接（代理请求，避免CORS）
		v1.POST("/cloud/register", api.RegisterToCloud(cfg, db))   // 注册到Cloud端（代理请求，避免CORS）
		v1.PUT("/config/credentials", api.UpdateCloudCredentials(cfg, db))
		v1.PUT("/config/cabinet-id", api.UpdateConfigCabinetID(cfg))

		// 脆弱性评估（无需认证，用于Web管理界面）
		vulnerabilityGroup := v1.Group("/vulnerability")
		{
			vulnerabilityGroup.GET("/current", api.GetCurrentVulnerability(vulnService))
			vulnerabilityGroup.GET("/history", api.GetVulnerabilityHistory(vulnService))
			vulnerabilityGroup.GET("/metrics", api.GetTransmissionMetrics(vulnService))
			vulnerabilityGroup.POST("/trigger", api.TriggerAssessment(vulnService))
			vulnerabilityGroup.POST("/dismiss", api.DismissVulnerability(vulnService))
		}

		// 地图服务（无需认证，用于Web管理界面）
		mapGroup := v1.Group("/map")
		{
			mapGroup.POST("/search", handlers.SearchPlace(cfg, logger))
		}

		// ABAC策略查询（只读API，无需认证，用于Web管理界面）
		if abacRepo != nil {
			abacHandler := handlers.NewABACHandler(abacRepo, logger)
			abacGroup := v1.Group("/abac")
			{
				abacGroup.GET("/policies", abacHandler.ListPolicies)         // 查询策略列表
				abacGroup.GET("/policies/:id", abacHandler.GetPolicy)        // 查询策略详情
				abacGroup.GET("/policies/stats", abacHandler.GetPolicyStats) // 策略统计
			}
		}
	}

	return router
}

// convertVulnerabilityConfig 转换配置格式
func convertVulnerabilityConfig(allCfg *config.Config) vulnerability.VulnerabilityConfig {
	cfg := allCfg.Vulnerability

	vulnCfg := vulnerability.VulnerabilityConfig{
		Enabled:              cfg.Enabled,
		AssessmentInterval:   cfg.AssessmentInterval,
		ScoreChangeThreshold: cfg.ScoreChangeThreshold,
		HistoryRetention:     cfg.HistoryRetention,
	}

	// 权重配置
	vulnCfg.Weights.Communication = cfg.Weights.Communication
	vulnCfg.Weights.ConfigSecurity = cfg.Weights.ConfigSecurity
	vulnCfg.Weights.DataAnomaly = cfg.Weights.DataAnomaly

	// 通信评分配置
	vulnCfg.Communication.LatencyThresholdMs = cfg.Communication.LatencyThresholdMs
	vulnCfg.Communication.PacketLossThreshold = cfg.Communication.PacketLossThreshold
	vulnCfg.Communication.ReconnectThresholdPerHour = cfg.Communication.ReconnectThresholdPerHour

	// 数据异常评分配置
	vulnCfg.DataAnomaly.MissingRateThreshold = cfg.DataAnomaly.MissingRateThreshold
	vulnCfg.DataAnomaly.AbnormalValueThreshold = cfg.DataAnomaly.AbnormalValueThreshold
	vulnCfg.DataAnomaly.AlertFrequencyThreshold = cfg.DataAnomaly.AlertFrequencyThreshold

	// 配置检查参数
	vulnCfg.MQTTBroker = allCfg.MQTT.BrokerAddress
	vulnCfg.ServerHost = allCfg.Server.Host
	vulnCfg.LogLevel = allCfg.Log.Level

	return vulnCfg
}
