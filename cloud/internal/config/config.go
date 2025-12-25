package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	MQTT       MQTTConfig       `mapstructure:"mqtt"`
	EdgeMQTT   EdgeMQTTConfig   `mapstructure:"edge_mqtt"` // Edge端MQTT配置（订阅传感器数据）
	EdgeAPI    EdgeAPIConfig    `mapstructure:"edge_api"`  // 调用Edge HTTP API配置
	JWT        JWTConfig        `mapstructure:"jwt"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Business   BusinessConfig   `mapstructure:"business"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	CORS       CORSConfig       `mapstructure:"cors"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port           int    `mapstructure:"port"`
	Host           string `mapstructure:"host"`
	Mode           string `mapstructure:"mode"`
	ReadTimeout    string `mapstructure:"read_timeout"`
	WriteTimeout   string `mapstructure:"write_timeout"`
	MaxHeaderBytes int    `mapstructure:"max_header_bytes"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

// PostgresConfig PostgreSQL配置
type PostgresConfig struct {
	Host                  string `mapstructure:"host"`
	Port                  int    `mapstructure:"port"`
	User                  string `mapstructure:"user"`
	Password              string `mapstructure:"password"`
	DBName                string `mapstructure:"dbname"`
	SSLMode               string `mapstructure:"sslmode"`
	MaxConnections        int    `mapstructure:"max_connections"`
	MaxIdleConnections    int    `mapstructure:"max_idle_connections"`
	ConnectionMaxLifetime string `mapstructure:"connection_max_lifetime"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
	DialTimeout  string `mapstructure:"dial_timeout"`
	ReadTimeout  string `mapstructure:"read_timeout"`
	WriteTimeout string `mapstructure:"write_timeout"`
}

// MQTTConfig MQTT配置（Cloud端自己的MQTT broker，用于命令下发）
type MQTTConfig struct {
	Broker               string            `mapstructure:"broker"`
	Username             string            `mapstructure:"username"`
	Password             string            `mapstructure:"password"`
	ClientID             string            `mapstructure:"client_id"`
	QoS                  byte              `mapstructure:"qos"`
	CleanSession         bool              `mapstructure:"clean_session"`
	KeepAlive            int               `mapstructure:"keep_alive"`
	ReconnectDelay       string            `mapstructure:"reconnect_delay"`
	MaxReconnectInterval string            `mapstructure:"max_reconnect_interval"`
	TLS                  EdgeMQTTTLSConfig `mapstructure:"tls"` // TLS配置
}

// EdgeMQTTConfig Edge端MQTT配置（订阅Edge端传感器数据）
type EdgeMQTTConfig struct {
	Enabled              bool              `mapstructure:"enabled"`                // 是否启用Edge端MQTT订阅
	Broker               string            `mapstructure:"broker"`                 // Edge端MQTT broker地址
	Username             string            `mapstructure:"username"`               // 用户名
	Password             string            `mapstructure:"password"`               // 密码
	ClientID             string            `mapstructure:"client_id"`              // 客户端ID
	QoS                  byte              `mapstructure:"qos"`                    // QoS级别
	CleanSession         bool              `mapstructure:"clean_session"`          // 是否清理会话
	KeepAlive            int               `mapstructure:"keep_alive"`             // 保持连接时间（秒）
	ReconnectDelay       string            `mapstructure:"reconnect_delay"`        // 重连延迟
	MaxReconnectInterval string            `mapstructure:"max_reconnect_interval"` // 最大重连间隔
	TLS                  EdgeMQTTTLSConfig `mapstructure:"tls"`                    // TLS配置
}

// EdgeMQTTTLSConfig Edge端MQTT TLS配置
type EdgeMQTTTLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`              // 是否启用TLS
	CAFile             string `mapstructure:"ca_file"`              // CA证书路径
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"` // 是否跳过证书验证
}

// EdgeAPIConfig Edge HTTP API配置（Cloud调用Edge进行控制）
type EdgeAPIConfig struct {
	BaseURL string `mapstructure:"base_url"` // 默认Edge API地址（含/api/v1）
	Scheme  string `mapstructure:"scheme"`   // 当没有BaseURL时，根据IP拼接的协议，默认http
	Port    int    `mapstructure:"port"`     // 当没有BaseURL时，根据IP拼接的端口，默认8001
	Timeout string `mapstructure:"timeout"`  // 调用Edge API的超时时间
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	Expiry        string `mapstructure:"expiry"`
	RefreshExpiry string `mapstructure:"refresh_expiry"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// BusinessConfig 业务参数配置
type BusinessConfig struct {
	HealthScore HealthScoreConfig `mapstructure:"health_score"`
	Alerts      AlertsConfig      `mapstructure:"alerts"`
	Sync        SyncConfig        `mapstructure:"sync"`
	License     LicenseConfig     `mapstructure:"license"`
	Command     CommandConfig     `mapstructure:"command"`
	Frontend    FrontendConfig    `mapstructure:"frontend"`
	Map         MapConfig         `mapstructure:"map"`
}

// HealthScoreConfig 健康评分配置
type HealthScoreConfig struct {
	Weights        HealthScoreWeights `mapstructure:"weights"`
	UpdateInterval string             `mapstructure:"update_interval"`
}

// HealthScoreWeights 健康评分权重
type HealthScoreWeights struct {
	OnlineRate     float64 `mapstructure:"online_rate"`
	DataQuality    float64 `mapstructure:"data_quality"`
	AlertSeverity  float64 `mapstructure:"alert_severity"`
	SensorNormalcy float64 `mapstructure:"sensor_normalcy"`
}

// AlertsConfig 告警配置
type AlertsConfig struct {
	OfflineTimeout int            `mapstructure:"offline_timeout"`
	SeverityLevels map[string]int `mapstructure:"severity_levels"`
	RetentionDays  int            `mapstructure:"retention_days"`
}

// SyncConfig 数据同步配置
type SyncConfig struct {
	BatchSize    int    `mapstructure:"batch_size"`
	SyncInterval string `mapstructure:"sync_interval"`
	Timeout      string `mapstructure:"timeout"`
}

// LicenseConfig 许可证配置
type LicenseConfig struct {
	CacheTTL           string `mapstructure:"cache_ttl"`
	ValidationInterval string `mapstructure:"validation_interval"`
	OfflineGracePeriod string `mapstructure:"offline_grace_period"`
	SigningKeyPath     string `mapstructure:"signing_key_path"`
}

// CommandConfig 命令下发配置
type CommandConfig struct {
	Timeout    string `mapstructure:"timeout"`
	RetryCount int    `mapstructure:"retry_count"`
	RetryDelay string `mapstructure:"retry_delay"`
}

// FrontendConfig 前端配置
type FrontendConfig struct {
	APIBaseURL           string `mapstructure:"api_base_url"`
	PollingInterval      int    `mapstructure:"polling_interval"`
	ChartRefreshInterval int    `mapstructure:"chart_refresh_interval"`
	PageSize             int    `mapstructure:"page_size"`
	MaxPageSize          int    `mapstructure:"max_page_size"`
}

// MapConfig 地图配置
type MapConfig struct {
	Enabled              bool            `mapstructure:"enabled"` // 是否启用地图功能
	Provider             string          `mapstructure:"provider"`
	TencentMapKey        string          `mapstructure:"tencent_map_key"`
	TencentWebServiceKey string          `mapstructure:"tencent_webservice_key"`
	DefaultCenter        MapCenterConfig `mapstructure:"default_center"`
	DefaultZoom          int             `mapstructure:"default_zoom"`
}

// MapCenterConfig 地图中心点配置
type MapCenterConfig struct {
	Latitude  float64 `mapstructure:"latitude"`
	Longitude float64 `mapstructure:"longitude"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	MetricsPath     string `mapstructure:"metrics_path"`
	HealthCheckPath string `mapstructure:"health_check_path"`
	ProfilerEnabled bool   `mapstructure:"profiler_enabled"`
	ProfilerPath    string `mapstructure:"profiler_path"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
	AllowHeaders []string `mapstructure:"allow_headers"`
	MaxAge       int      `mapstructure:"max_age"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
)

// Load 加载配置文件
// 支持多种配置文件路径查找方式：
// 1. 环境变量 CLOUD_CONFIG_PATH
// 2. 命令行参数指定的路径
// 3. 默认搜索路径（当前目录、项目根目录）
func Load(configPath string) (*Config, error) {
	// 优先使用环境变量指定的配置文件
	if envConfigPath := os.Getenv("CLOUD_CONFIG_PATH"); envConfigPath != "" {
		configPath = envConfigPath
	}

	// 如果没有指定配置文件，尝试默认路径
	if configPath == "" || configPath == "config.yaml" {
		configPath = findConfigFile()
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 支持环境变量覆盖
	viper.AutomaticEnv()
	viper.SetEnvPrefix("CLOUD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	once.Do(func() {
		instance = &config
	})

	return instance, nil
}

// findConfigFile 查找配置文件，按以下顺序搜索：
// 1. 当前工作目录
// 2. 可执行文件所在目录
// 3. 项目根目录（相对于可执行文件向上查找）
func findConfigFile() string {
	configFileName := "config.yaml"

	// 1. 检查当前工作目录
	if _, err := os.Stat(configFileName); err == nil {
		return configFileName
	}

	// 2. 检查可执行文件所在目录
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		configPath := filepath.Join(exeDir, configFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		// 3. 尝试向上查找项目根目录（假设可执行文件在 cmd/cloud-server 或 bin 目录下）
		// 向上查找最多3层
		for i := 0; i < 3; i++ {
			exeDir = filepath.Dir(exeDir)
			configPath := filepath.Join(exeDir, configFileName)
			if _, err := os.Stat(configPath); err == nil {
				return configPath
			}
		}
	}

	// 如果都找不到，返回默认路径（会在后续读取时报错）
	return configFileName
}

// Get 获取全局配置实例
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return instance
}

// WatchConfig 监听配置文件变化并热重载
func WatchConfig(callback func(*Config)) error {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		mu.Lock()
		defer mu.Unlock()

		var newConfig Config
		if err := viper.Unmarshal(&newConfig); err != nil {
			fmt.Printf("Failed to reload config: %v\n", err)
			return
		}

		// 验证新配置
		if err := newConfig.Validate(); err != nil {
			fmt.Printf("New config validation failed: %v\n", err)
			return
		}

		instance = &newConfig
		fmt.Println("Configuration reloaded successfully")

		if callback != nil {
			callback(instance)
		}
	})
	return nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证服务器端口
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// 验证健康评分权重
	weights := c.Business.HealthScore.Weights
	totalWeight := weights.OnlineRate + weights.DataQuality +
		weights.AlertSeverity + weights.SensorNormalcy

	if totalWeight < 0.99 || totalWeight > 1.01 {
		return fmt.Errorf("health score weights must sum to 1.0, got: %f", totalWeight)
	}

	// 验证权重范围
	if weights.OnlineRate < 0 || weights.OnlineRate > 1 ||
		weights.DataQuality < 0 || weights.DataQuality > 1 ||
		weights.AlertSeverity < 0 || weights.AlertSeverity > 1 ||
		weights.SensorNormalcy < 0 || weights.SensorNormalcy > 1 {
		return fmt.Errorf("health score weights must be between 0 and 1")
	}

	// 验证JWT密钥
	if c.JWT.Secret == "" || c.JWT.Secret == "your-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret must be set and changed from default")
	}

	if c.EdgeAPI.Port == 0 {
		c.EdgeAPI.Port = 8001
	}
	if c.EdgeAPI.Scheme == "" {
		c.EdgeAPI.Scheme = "http"
	}
	if c.EdgeAPI.Timeout == "" {
		c.EdgeAPI.Timeout = "5s"
	}
	if c.EdgeAPI.BaseURL == "" {
		c.EdgeAPI.BaseURL = fmt.Sprintf("%s://localhost:%d/api/v1", c.EdgeAPI.Scheme, c.EdgeAPI.Port)
	}

	return nil
}

// GetPostgresConnectionString 获取PostgreSQL连接字符串
func (c *Config) GetPostgresConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Postgres.Host,
		c.Database.Postgres.Port,
		c.Database.Postgres.User,
		c.Database.Postgres.Password,
		c.Database.Postgres.DBName,
		c.Database.Postgres.SSLMode,
	)
}

// GetRedisAddr 获取Redis地址
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Database.Redis.Host, c.Database.Redis.Port)
}

// ParseDuration 解析duration字符串
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
