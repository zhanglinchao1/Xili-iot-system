package api

import (
	"cloud-system/internal/abac"
	"cloud-system/internal/api/handlers"
	"cloud-system/internal/api/middleware"
	"cloud-system/internal/config"
	"cloud-system/internal/mqtt"
	"cloud-system/internal/repository/postgres"
	"cloud-system/internal/repository/timescaledb"
	"cloud-system/internal/services"
	"cloud-system/internal/utils"
	"cloud-system/internal/websocket"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置API路由，返回传感器服务、流量服务和告警服务供MQTT订阅使用
func SetupRoutes(router *gin.Engine, cfg *config.Config, pgClient *postgres.Client, mqttClient *mqtt.Client, wsHub *websocket.Hub, edgeMQTTClient *mqtt.EdgeClient) (services.SensorService, *services.TrafficService, services.AlertService) {
	// 初始化Repository
	userRepo := postgres.NewUserRepo(pgClient.GetPool())
	cabinetRepo := postgres.NewCabinetRepo(pgClient.GetPool())
	sensorDeviceRepo := postgres.NewSensorDeviceRepo(pgClient.GetPool())
	sensorDataRepo := timescaledb.NewSensorDataRepo(pgClient.GetPool())
	commandRepo := postgres.NewCommandRepo(pgClient.GetPool())
	licenseRepo := postgres.NewLicenseRepo(pgClient.GetPool())
	alertRepo := postgres.NewAlertRepo(pgClient.GetPool())
	vulnRepo := postgres.NewVulnerabilityRepository(pgClient.GetPool(), utils.GetLogger())
	trafficRepo := postgres.NewTrafficRepository(pgClient.GetPool(), utils.GetLogger())
	policyRepo := postgres.NewPolicyRepo(pgClient.GetPool())

	// 初始化Service
	authService := services.NewAuthService(userRepo, cfg)
	userService := services.NewUserService(userRepo)
	sensorService := services.NewSensorService(sensorDataRepo, sensorDeviceRepo, cabinetRepo, alertRepo)
	trafficService := services.NewTrafficService()
	commandService := services.NewCommandService(commandRepo, cabinetRepo, mqttClient)
	// 许可证签名密钥路径，如果未配置使用默认值
	signingKeyPath := cfg.Business.License.SigningKeyPath
	if signingKeyPath == "" {
		signingKeyPath = "./configs/keys/license_signing_key.pem"
	}
	licenseService := services.NewLicenseService(licenseRepo, cabinetRepo, signingKeyPath)
	cabinetService := services.NewCabinetService(cabinetRepo, licenseService)
	alertService := services.NewAlertService(alertRepo, cabinetRepo, cfg.EdgeAPI)

	// 注入Edge MQTT客户端到alertService(用于下发告警解决命令)
	if edgeMQTTClient != nil {
		utils.Info("Attempting to set Edge MQTT client for alert service")
		if svc, ok := alertService.(interface{ SetEdgeMQTTClient(*mqtt.EdgeClient) }); ok {
			svc.SetEdgeMQTTClient(edgeMQTTClient)
		} else {
			utils.Warn("Failed to cast alertService to SetEdgeMQTTClient interface")
		}
	} else {
		utils.Warn("edgeMQTTClient is nil, skipping MQTT client injection")
	}

	vulnService := services.NewVulnerabilityService(vulnRepo, cabinetRepo, utils.GetLogger())
	mapService := services.NewMapService(cfg, utils.GetLogger())

	// 初始化Handler
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	cabinetHandler := handlers.NewCabinetHandler(cabinetService)
	sensorHandler := handlers.NewSensorHandler(sensorService)
	commandHandler := handlers.NewCommandHandler(commandService)
	licenseHandler := handlers.NewLicenseHandler(licenseService, commandService)
	alertHandler := handlers.NewAlertHandler(alertService)
	vulnHandler := handlers.NewVulnerabilityHandler(vulnService)
	trafficHandler := handlers.NewTrafficHandler(trafficService, cabinetService, trafficRepo)
	mapHandler := handlers.NewMapHandler(mapService)
	abacHandler := handlers.NewABACHandler(policyRepo)

	// 【ABAC策略分发】初始化PolicyPublisher
	if edgeMQTTClient != nil {
		policyPublisher := mqtt.NewPolicyPublisher(edgeMQTTClient.GetClient(), policyRepo)
		abacHandler.SetPolicyPublisher(policyPublisher)
		utils.Info("ABAC PolicyPublisher已初始化")
	}

	// 全局中间件
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.CORSMiddleware(cfg))

	// 健康检查端点（无需认证）
	router.GET("/health", HealthCheckHandler(cfg))

	// WebSocket端点（无需认证，用于实时数据推送）
	router.GET("/ws", func(c *gin.Context) {
		wsHub.HandleWebSocket(c.Writer, c.Request)
	})

	// 腾讯地图资源通用代理（放在v1组外，避免中间件影响）
	router.GET("/api/v1/tencent-map-proxy/*path", ProxyTencentMapResourceHandler(cfg))
	router.POST("/api/v1/tencent-map-proxy/*path", ProxyTencentMapResourceHandler(cfg))
	router.OPTIONS("/api/v1/tencent-map-proxy/*path", ProxyTencentMapResourceHandler(cfg))

	// API v1路由组
	v1 := router.Group("/api/v1")
	{
		// 公开端点（无需认证）
		v1.GET("/", APIInfoHandler())

		// 腾讯地图SDK代理（避免域名白名单问题）
		v1.GET("/tencent-map/gljs", ProxyTencentMapSDKHandler(cfg))

		// 认证端点
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)                                // 用户注册
			auth.POST("/login", authHandler.Login)                                      // 用户登录
			auth.GET("/me", middleware.AuthMiddleware(cfg), authHandler.GetCurrentUser) // 获取当前用户（需认证）
		}

		// 配置端点（前端获取配置）
		v1.GET("/config", GetConfigHandler(cfg))

		// Edge端同步端点组（使用可选的API Key认证 + ABAC访问控制）
		edgeSync := v1.Group("")
		edgeSync.Use(middleware.EdgeAPIKeyMiddleware(cabinetRepo))
		edgeSync.Use(abac.ABACMiddleware(policyRepo, cabinetRepo, vulnRepo))
		{
			// 许可证验证端点
			edgeSync.POST("/license/validate", licenseHandler.ValidateLicense)

			// 传感器数据同步端点
			edgeSync.POST("/cabinets/:cabinet_id/sync", sensorHandler.SyncSensorData)

			// 脆弱性评估同步端点
			edgeSync.POST("/cabinets/:cabinet_id/vulnerability/sync", vulnHandler.SyncAssessment)

			// 告警同步端点
			edgeSync.POST("/cabinets/:cabinet_id/alerts/sync", alertHandler.SyncAlerts)

			// 储能柜信息同步端点
			edgeSync.PUT("/cabinets/:cabinet_id/sync", cabinetHandler.SyncCabinetInfo)

			// 命令回执
			edgeSync.POST("/commands/:command_id/ack", commandHandler.AckCommand)
		}

		// Edge端激活端点（公开端点，使用注册Token认证）
		v1.POST("/cabinets/activate", cabinetHandler.ActivateCabinet)

		// Edge端直接注册端点（公开端点，无需认证，一步完成注册和激活）
		v1.POST("/cabinets/register", cabinetHandler.RegisterCabinet)

		// 需要JWT认证的端点（+ ABAC访问控制）
		authorized := v1.Group("")
		authorized.Use(middleware.AuthMiddleware(cfg))
		authorized.Use(abac.ABACMiddleware(policyRepo, nil, nil))
		{
			// 储能柜管理
			cabinets := authorized.Group("/cabinets")
			{
				cabinets.GET("", cabinetHandler.ListCabinets)                                  // 列表
				cabinets.GET("/locations", cabinetHandler.GetCabinetLocations)                 // 位置信息（地图展示）
				cabinets.GET("/statistics", cabinetHandler.GetCabinetStatistics)               // 统计信息
				cabinets.POST("", cabinetHandler.CreateCabinet)                                // 创建
				cabinets.POST("/pre-register", cabinetHandler.PreRegisterCabinet)              // 预注册
				cabinets.GET("/:cabinet_id", cabinetHandler.GetCabinet)                        // 详情
				cabinets.PUT("/:cabinet_id", cabinetHandler.UpdateCabinet)                     // 更新
				cabinets.DELETE("/:cabinet_id", cabinetHandler.DeleteCabinet)                  // 删除
				cabinets.GET("/:cabinet_id/activation-info", cabinetHandler.GetActivationInfo) // 获取激活信息
				cabinets.POST("/:cabinet_id/regenerate-token", cabinetHandler.RegenerateToken) // 重新生成Token

				// API Key管理
				cabinets.GET("/:cabinet_id/api-key", cabinetHandler.GetAPIKey)                    // 获取API Key信息
				cabinets.POST("/:cabinet_id/api-key/regenerate", cabinetHandler.RegenerateAPIKey) // 重新生成API Key
				cabinets.DELETE("/:cabinet_id/api-key", cabinetHandler.RevokeAPIKey)              // 撤销API Key

				// 储能柜传感器数据
				cabinets.GET("/:cabinet_id/devices", sensorHandler.ListCabinetDevices)
				cabinets.GET("/:cabinet_id/sensors/latest", sensorHandler.GetLatestSensorData)
				cabinets.GET("/:cabinet_id/alerts", alertHandler.GetCabinetAlerts)
				cabinets.GET("/:cabinet_id/health-score", alertHandler.GetHealthScore)

				// 储能柜脆弱性评估
				cabinets.GET("/:cabinet_id/vulnerability/latest", vulnHandler.GetLatestAssessment)
				cabinets.GET("/:cabinet_id/vulnerability/history", vulnHandler.GetHistory)
				cabinets.GET("/:cabinet_id/vulnerability/stats", vulnHandler.GetStats)

				// 储能柜命令下发
				cabinets.POST("/:cabinet_id/commands", commandHandler.SendCommand)
			}

			// 传感器设备管理
			devices := authorized.Group("/devices")
			{
				devices.GET("/:device_id", GetDeviceHandler())
			}

			// 传感器数据查询
			authorized.GET("/devices/data", sensorHandler.GetHistoricalData)

			// 许可证管理
			licenses := authorized.Group("/licenses")
			{
				licenses.GET("", licenseHandler.ListLicenses)
				licenses.POST("", licenseHandler.CreateLicense)
				licenses.GET("/:cabinet_id", licenseHandler.GetLicense)
				licenses.PUT("/:cabinet_id", licenseHandler.RenewLicense)
				licenses.POST("/:cabinet_id/revoke", licenseHandler.RevokeLicense)
				licenses.POST("/:cabinet_id/push", licenseHandler.PushLicense)
				licenses.POST("/sync", licenseHandler.SyncLicenses)
				licenses.DELETE("/:cabinet_id", licenseHandler.DeleteLicense)
			}

			// 命令管理
			commands := authorized.Group("/commands")
			{
				commands.GET("/:command_id", commandHandler.GetCommand)
				commands.GET("", commandHandler.ListCommands)
			}

			// 告警管理
			alerts := authorized.Group("/alerts")
			{
				alerts.GET("", alertHandler.ListAlerts)
				alerts.GET("/:alert_id", alertHandler.GetAlert)
				alerts.PUT("/:alert_id/resolve", alertHandler.ResolveAlert)
				alerts.POST("/batch-resolve", alertHandler.BatchResolveAlerts)
			}

			// 脆弱性评估管理
			vulnerability := authorized.Group("/vulnerability")
			{
				vulnerability.GET("/assessments", vulnHandler.ListAssessments)
				vulnerability.GET("/assessments/:id", vulnHandler.GetAssessmentDetail)
			}

			traffic := authorized.Group("/traffic")
			{
				traffic.GET("/summary", trafficHandler.GetSummary)
				traffic.GET("/cabinets", trafficHandler.ListCabinets)
				traffic.GET("/cabinets/:cabinet_id", trafficHandler.GetCabinetDetail)
			}

			// 用户管理
			users := authorized.Group("/users")
			{
				// 普通用户可访问的端点
				users.GET("/profile", userHandler.GetProfile)      // 获取个人信息
				users.PUT("/profile", userHandler.UpdateProfile)   // 更新个人信息
				users.PUT("/password", userHandler.UpdatePassword) // 修改密码

				// 管理员专用端点
				adminUsers := users.Group("")
				adminUsers.Use(middleware.AdminMiddleware())
				{
					adminUsers.GET("", userHandler.ListUsers)                                 // 获取用户列表
					adminUsers.POST("", userHandler.CreateUser)                               // 创建用户
					adminUsers.GET("/:user_id", userHandler.GetUser)                          // 获取用户详情
					adminUsers.PUT("/:user_id", userHandler.UpdateUser)                       // 更新用户
					adminUsers.PUT("/:user_id/reset-password", userHandler.ResetUserPassword) // 重置用户密码
					adminUsers.DELETE("/:user_id", userHandler.DeleteUser)                    // 删除用户
				}
			}

			// 审计日志
			authorized.GET("/audit-logs", ListAuditLogsHandler())

			// ABAC策略管理（仅管理员）
			abac := authorized.Group("/abac")
			abac.Use(middleware.AdminMiddleware())
			{
				abac.GET("/policies", abacHandler.ListPolicies)                                        // 列出所有策略
				abac.POST("/policies", abacHandler.CreatePolicy)                                       // 创建策略
				abac.GET("/policies/:id", abacHandler.GetPolicy)                                       // 获取策略详情
				abac.PUT("/policies/:id", abacHandler.UpdatePolicy)                                    // 更新策略
				abac.DELETE("/policies/:id", abacHandler.DeletePolicy)                                 // 删除策略
				abac.POST("/policies/:id/toggle", abacHandler.TogglePolicy)                            // 切换策略启用状态
				abac.POST("/policies/:id/distribute", abacHandler.DistributePolicy)                    // 分发策略到储能柜
				abac.GET("/policies/:id/distribution-status", abacHandler.GetPolicyDistributionStatus) // 获取策略分发状态
				abac.POST("/policies/:id/broadcast", abacHandler.BroadcastPolicy)                      // 广播策略到所有储能柜
				abac.POST("/cabinets/:cabinet_id/policies/sync", abacHandler.FullSyncPolicies)         // 全量同步策略
				abac.POST("/cabinets/:cabinet_id/logs/sync", abacHandler.SyncDeviceAccessLogs)         // 接收设备访问日志
				abac.GET("/access-logs", abacHandler.ListAccessLogs)                                   // 访问日志
				abac.GET("/access-stats", abacHandler.GetAccessStats)                                  // 访问统计
				abac.GET("/distribution-logs", abacHandler.ListDistributionLogs)                       // 策略分发历史
				abac.POST("/evaluate", abacHandler.EvaluatePolicy)                                     // 测试策略评估
			}

		}
	}

	// 地图服务（公开接口，供Edge端使用）
	mapGroup := v1.Group("/map")
	{
		mapGroup.POST("/geocode", mapHandler.Geocode)                  // 地理编码
		mapGroup.POST("/search", mapHandler.SearchPlace)               // 地点搜索 (POST)
		mapGroup.GET("/search", mapHandler.SearchLocationGET)          // 地点搜索 (GET)
		mapGroup.GET("/geocode/reverse", mapHandler.ReverseGeocodeGET) // 逆地理编码 (GET)
	}

	// 监控端点
	if cfg.Monitoring.Enabled {
		router.GET(cfg.Monitoring.MetricsPath, MetricsHandler())
	}

	// 返回服务供MQTT订阅使用
	return sensorService, trafficService, alertService
}

// Handler占位符（待实现）
func HealthCheckHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "cloud-system",
		})
	}
}

func APIInfoHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Cloud端储能柜集群管理系统 API",
			"version": "1.0.0",
		})
	}
}

// GetConfigHandler 返回前端需要的配置信息
func GetConfigHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"mqtt": gin.H{
				"broker":  cfg.EdgeMQTT.Broker,
				"enabled": cfg.EdgeMQTT.Enabled,
			},
			"websocket": gin.H{
				"enabled": true,
				"url":     "/ws",
			},
			"frontend": gin.H{
				"api_base_url":           cfg.Business.Frontend.APIBaseURL,
				"polling_interval":       cfg.Business.Frontend.PollingInterval,
				"chart_refresh_interval": cfg.Business.Frontend.ChartRefreshInterval,
				"page_size":              cfg.Business.Frontend.PageSize,
				"max_page_size":          cfg.Business.Frontend.MaxPageSize,
			},
			"map": gin.H{
				"enabled":                cfg.Business.Map.Enabled,
				"provider":               cfg.Business.Map.Provider,
				"tencent_map_key":        cfg.Business.Map.TencentMapKey,
				"tencent_webservice_key": cfg.Business.Map.TencentWebServiceKey,
				"default_center": gin.H{
					"latitude":  cfg.Business.Map.DefaultCenter.Latitude,
					"longitude": cfg.Business.Map.DefaultCenter.Longitude,
				},
				"default_zoom": cfg.Business.Map.DefaultZoom,
			},
		})
	}
}

func ValidateLicenseHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func SyncDataHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func ListCabinetsHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func CreateCabinetHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func GetCabinetHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func UpdateCabinetHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func DeleteCabinetHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func GetLatestSensorDataHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func GetCabinetAlertsHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func GetDeviceHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func GetDeviceDataHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func ListLicensesHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func CreateLicenseHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func GetLicenseHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func RenewLicenseHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func RevokeLicenseHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func SendCommandHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func GetCommandStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func ListCommandsHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func ListAlertsHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func ResolveAlertHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func ListAuditLogsHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) { c.JSON(501, gin.H{"message": "Not implemented"}) }
}

// ProxyTencentMapSDKHandler 代理腾讯地图SDK请求（避免域名白名单问题）
// 并自动替换SDK内的腾讯域名为本地Nginx代理路径
func ProxyTencentMapSDKHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 构建腾讯地图SDK URL
		tencentMapURL := "https://map.qq.com/api/gljs"

		// 获取查询参数（v和key）
		v := c.Query("v")
		key := c.Query("key")

		// 如果没有提供key，使用配置中的key
		if key == "" {
			key = cfg.Business.Map.TencentMapKey
		}

		// 构建完整URL
		fullURL := tencentMapURL + "?v=" + v + "&key=" + key

		// 创建HTTP客户端
		client := &http.Client{}
		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			c.JSON(500, gin.H{"error": "创建请求失败", "message": err.Error()})
			return
		}

		// 设置请求头
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Accept", "application/javascript,*/*")

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(502, gin.H{"error": "请求腾讯地图SDK失败", "message": err.Error()})
			return
		}
		defer resp.Body.Close()

		// 读取响应体内容
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(500, gin.H{"error": "读取响应失败", "message": err.Error()})
			return
		}

		// 将SDK内容转为字符串，准备替换域名
		sdkContent := string(bodyBytes)

		// 替换SDK内硬编码的腾讯域名为本地Nginx代理路径
		// 这样SDK内部发起的请求会经过Nginx代理，避免CORS错误
		sdkContent = strings.ReplaceAll(sdkContent, "https://rt0.map.gtimg.com", "/tencent-map-tiles/rt0")
		sdkContent = strings.ReplaceAll(sdkContent, "https://rt1.map.gtimg.com", "/tencent-map-tiles/rt1")
		sdkContent = strings.ReplaceAll(sdkContent, "https://rt2.map.gtimg.com", "/tencent-map-tiles/rt2")
		sdkContent = strings.ReplaceAll(sdkContent, "https://rt3.map.gtimg.com", "/tencent-map-tiles/rt3")
		sdkContent = strings.ReplaceAll(sdkContent, "https://apikey.map.qq.com", "/tencent-map-apikey")
		sdkContent = strings.ReplaceAll(sdkContent, "https://vectorsdk.map.qq.com", "/tencent-map-vectorsdk")
		sdkContent = strings.ReplaceAll(sdkContent, "https://confinfo.map.qq.com", "/tencent-map-confinfo")
		sdkContent = strings.ReplaceAll(sdkContent, "https://overseactrl.map.qq.com", "/tencent-map-overseactrl")
		sdkContent = strings.ReplaceAll(sdkContent, "https://mapapi.qq.com", "/tencent-map-mapapi")
		sdkContent = strings.ReplaceAll(sdkContent, "https://pr.map.qq.com", "/tencent-map-pr")
		sdkContent = strings.ReplaceAll(sdkContent, "https://mapstyle.qpic.cn", "/tencent-map-mapstyle")

		// 设置响应头
		c.Writer.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// 返回修改后的SDK内容
		c.String(200, sdkContent)
	}
}

// ProxyTencentMapResourceHandler 代理腾讯地图资源请求（通用代理，支持所有腾讯地图域名）
func ProxyTencentMapResourceHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取路径参数（已编码的完整URL）
		encodedPath := c.Param("path")
		if encodedPath == "" {
			c.JSON(400, gin.H{"error": "路径不能为空"})
			return
		}

		// 移除开头的斜杠
		if encodedPath[0] == '/' {
			encodedPath = encodedPath[1:]
		}

		// 解码URL
		decodedPath, err := url.QueryUnescape(encodedPath)
		if err != nil {
			decodedPath = encodedPath // 如果解码失败，使用原始路径
		}

		// 解析完整URL
		fullURL, err := url.Parse(decodedPath)
		if err != nil {
			c.JSON(400, gin.H{"error": "无效的URL", "message": err.Error()})
			return
		}

		// 如果URL不完整，尝试构建完整URL
		if fullURL.Scheme == "" {
			// 根据host判断目标域名
			host := fullURL.Host
			if host == "" {
				// 尝试从路径中提取host
				if strings.Contains(decodedPath, "apikey.map.qq.com") {
					host = "apikey.map.qq.com"
				} else if strings.Contains(decodedPath, "vectorsdk.map.qq.com") {
					host = "vectorsdk.map.qq.com"
				} else if strings.Contains(decodedPath, "rt0.map.gtimg.com") {
					host = "rt0.map.gtimg.com"
				} else if strings.Contains(decodedPath, "rt1.map.gtimg.com") {
					host = "rt1.map.gtimg.com"
				} else if strings.Contains(decodedPath, "rt2.map.gtimg.com") {
					host = "rt2.map.gtimg.com"
				} else if strings.Contains(decodedPath, "pr.map.qq.com") {
					host = "pr.map.qq.com"
				} else if strings.Contains(decodedPath, "confinfo.map.qq.com") {
					host = "confinfo.map.qq.com"
				} else if strings.Contains(decodedPath, "overseactrl.map.qq.com") {
					host = "overseactrl.map.qq.com"
				} else if strings.Contains(decodedPath, "mapapi.qq.com") {
					host = "mapapi.qq.com"
				} else {
					host = "map.qq.com"
				}
			}
			fullURL.Scheme = "https"
			fullURL.Host = host
		}

		// 构建完整URL字符串
		targetURL := fullURL.String()

		// 创建HTTP客户端
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		// 创建请求
		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.JSON(500, gin.H{"error": "创建请求失败", "message": err.Error()})
			return
		}

		// 复制请求头（排除Host和Connection）
		for key, values := range c.Request.Header {
			if key != "Host" && key != "Connection" {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
		}

		// 设置User-Agent
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Referer", c.Request.Referer())

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(502, gin.H{"error": "请求失败", "message": err.Error(), "url": fullURL})
			return
		}
		defer resp.Body.Close()

		// 复制响应头
		for key, values := range resp.Header {
			// 排除一些不需要的响应头
			if key != "Connection" && key != "Transfer-Encoding" {
				for _, value := range values {
					c.Writer.Header().Add(key, value)
				}
			}
		}

		// 设置CORS头
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, HEAD")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// 处理OPTIONS请求
		if c.Request.Method == "OPTIONS" {
			c.Writer.WriteHeader(200)
			return
		}

		// 设置状态码
		c.Writer.WriteHeader(resp.StatusCode)

		// 复制响应体
		io.Copy(c.Writer, resp.Body)
	}
}
