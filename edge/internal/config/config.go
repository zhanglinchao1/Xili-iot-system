/*
 * 配置管理模块
 * 负责加载和管理系统配置
 */
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 系统配置
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Auth          AuthConfig          `yaml:"auth"`
	Device        DeviceConfig        `yaml:"device"`
	Data          DataConfig          `yaml:"data"`
	Database      DatabaseConfig      `yaml:"database"`
	Cloud         CloudConfig         `yaml:"cloud"`
	Registration  RegistrationConfig  `yaml:"registration"`
	Alert         AlertConfig         `yaml:"alert"`
	License       LicenseConfig       `yaml:"license"`
	Log           LogConfig           `yaml:"log"`
	Monitoring    MonitoringConfig    `yaml:"monitoring"`
	MQTT          MQTTConfig          `yaml:"mqtt"`
	Vulnerability VulnerabilityConfig `yaml:"vulnerability"`
	ABAC          ABACConfig          `yaml:"abac"`
	Map           MapConfig           `yaml:"map"`
}

// ABACConfig ABAC设备权限管理配置
type ABACConfig struct {
	Enabled bool `yaml:"enabled"` // 是否启用设备ABAC权限控制
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // debug, release, test
}

// AuthConfig 认证配置
type AuthConfig struct {
	ChallengeTTL time.Duration `yaml:"challenge_ttl"`
	SessionTTL   time.Duration `yaml:"session_ttl"`
	MaxRetry     int           `yaml:"max_retry"`
	ZKP          ZKPConfig     `yaml:"zkp"`
}

// ZKPConfig 零知识证明配置
type ZKPConfig struct {
	CircuitPath      string `yaml:"circuit_path"`
	ProvingScheme    string `yaml:"proving_scheme"`
	VerifyingKeyPath string `yaml:"verifying_key_path"` // verifying key文件路径
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	OfflineTimeout    time.Duration `yaml:"offline_timeout"`
	MaxDevices        int           `yaml:"max_devices"`
	SupportedSensors  []string      `yaml:"supported_sensors"`
}

// DataConfig 数据配置
type DataConfig struct {
	CollectInterval time.Duration `yaml:"collect_interval"`
	SyncInterval    time.Duration `yaml:"sync_interval"`
	RetentionDays   int           `yaml:"retention_days"`
	BatchSize       int           `yaml:"batch_size"`
	BufferSize      int           `yaml:"buffer_size"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver             string `yaml:"driver"`
	Path               string `yaml:"path"`
	MaxConnections     int    `yaml:"max_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
}

// CloudConfig 云端配置
type CloudConfig struct {
	Enabled       bool          `yaml:"enabled"`
	Endpoint      string        `yaml:"endpoint"`
	APIKey        string        `yaml:"api_key"`
	AdminToken    string        `yaml:"admin_token"` // Cloud端管理员JWT token（用于预注册）
	CabinetID     string        `yaml:"cabinet_id"`  // 储能柜ID(同时作为Edge端标识)
	Timeout       time.Duration `yaml:"timeout"`
	RetryCount    int           `yaml:"retry_count"`
	RetryInterval time.Duration `yaml:"retry_interval"`
	// 储能柜详细信息（保存到配置文件，避免localStorage跨域问题）
	CabinetName   string   `yaml:"cabinet_name,omitempty"`   // 储能柜名称
	Location      string   `yaml:"location,omitempty"`       // 位置信息
	Latitude      *float64 `yaml:"latitude,omitempty"`       // 纬度
	Longitude     *float64 `yaml:"longitude,omitempty"`      // 经度
	CapacityKWh   *float64 `yaml:"capacity_kwh,omitempty"`   // 容量(kWh)
	DeviceModel   string   `yaml:"device_model,omitempty"`   // 设备型号
}

// RegistrationConfig 注册激活配置
type RegistrationConfig struct {
	Enabled    bool   `yaml:"enabled"`     // 是否启用自动激活
	Token      string `yaml:"token"`       // 预注册Token (24小时有效)
	MACAddress string `yaml:"mac_address"` // 绑定的MAC地址
}

// AlertConfig 告警配置
type AlertConfig struct {
	Enabled    bool            `yaml:"enabled"`
	Thresholds AlertThresholds `yaml:"thresholds"`
}

// LicenseConfig 许可证配置
type LicenseConfig struct {
	Enabled     bool          `yaml:"enabled"`      // 是否启用许可证验证
	Path        string        `yaml:"path"`         // 许可证文件路径
	PubKeyPath  string        `yaml:"pubkey_path"`  // 厂商公钥路径
	GracePeriod time.Duration `yaml:"grace_period"` // 过期宽限期（默认72小时）
}

// AlertThresholds 告警阈值
type AlertThresholds struct {
	CO2Max          float64 `yaml:"co2_max"`
	COMax           float64 `yaml:"co_max"`
	SmokeMax        float64 `yaml:"smoke_max"`
	LiquidLevelMin  float64 `yaml:"liquid_level_min"`
	LiquidLevelMax  float64 `yaml:"liquid_level_max"`
	ConductivityMin float64 `yaml:"conductivity_min"`
	ConductivityMax float64 `yaml:"conductivity_max"`
	TemperatureMin  float64 `yaml:"temperature_min"`
	TemperatureMax  float64 `yaml:"temperature_max"`
	FlowMin         float64 `yaml:"flow_min"`
	FlowMax         float64 `yaml:"flow_max"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`  // debug, info, warn, error
	Output     string `yaml:"output"` // console, file, both
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"` // MB
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"` // days
	Compress   bool   `yaml:"compress"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricsEnabled      bool          `yaml:"metrics_enabled"`
	MetricsPort         int           `yaml:"metrics_port"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Enabled              bool          `yaml:"enabled"`
	BrokerAddress        string        `yaml:"broker_address"`
	ClientID             string        `yaml:"client_id"`
	Username             string        `yaml:"username"`
	Password             string        `yaml:"password"`
	QoS                  byte          `yaml:"qos"`
	KeepAlive            int           `yaml:"keep_alive"`
	CleanSession         bool          `yaml:"clean_session"`
	ReconnectInterval    time.Duration `yaml:"reconnect_interval"`
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts"`
	// TLS配置
	TLS TLSConfig `yaml:"tls"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`              // 是否启用TLS
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"` // 是否跳过证书验证（仅用于测试）
	CAFile             string `yaml:"ca_file"`              // CA证书文件路径
	CertFile           string `yaml:"cert_file"`            // 客户端证书文件路径
	KeyFile            string `yaml:"key_file"`             // 客户端私钥文件路径
}

// Load 加载配置文件
func Load(configFile string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 处理环境变量
	config.processEnvVars()

	// 验证配置
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// processEnvVars 处理环境变量
func (c *Config) processEnvVars() {
	// 替换API Key
	if c.Cloud.APIKey == "${CLOUD_API_KEY}" {
		if apiKey := os.Getenv("CLOUD_API_KEY"); apiKey != "" {
			c.Cloud.APIKey = apiKey
		}
	}

	// 替换数据库路径
	if dbPath := os.Getenv("EDGE_DB_PATH"); dbPath != "" {
		c.Database.Path = dbPath
	}

	// 替换云端端点
	if endpoint := os.Getenv("CLOUD_ENDPOINT"); endpoint != "" {
		c.Cloud.Endpoint = endpoint
	}

	// 替换储能柜ID
	if cabinetID := os.Getenv("CABINET_ID"); cabinetID != "" {
		c.Cloud.CabinetID = cabinetID
	}

	// 替换MQTT broker地址
	if mqttBroker := os.Getenv("MQTT_BROKER_ADDRESS"); mqttBroker != "" {
		c.MQTT.BrokerAddress = mqttBroker
	}
}

// validate 验证配置
func (c *Config) validate() error {
	// 验证服务器配置
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", c.Server.Port)
	}

	// 验证模式
	validModes := map[string]bool{
		"debug":   true,
		"release": true,
		"test":    true,
	}
	if !validModes[c.Server.Mode] {
		return fmt.Errorf("无效的运行模式: %s", c.Server.Mode)
	}

	// 验证认证配置
	if c.Auth.ChallengeTTL <= 0 {
		return fmt.Errorf("挑战有效期必须大于0")
	}
	if c.Auth.SessionTTL <= 0 {
		return fmt.Errorf("会话有效期必须大于0")
	}

	// 验证设备配置
	if c.Device.MaxDevices <= 0 {
		return fmt.Errorf("最大设备数必须大于0")
	}
	if c.Device.HeartbeatInterval <= 0 {
		return fmt.Errorf("心跳间隔必须大于0")
	}

	// 验证数据配置
	if c.Data.BatchSize <= 0 {
		return fmt.Errorf("批量大小必须大于0")
	}
	if c.Data.BufferSize <= 0 {
		return fmt.Errorf("缓冲区大小必须大于0")
	}

	// 验证数据库配置
	if c.Database.Driver != "sqlite3" && c.Database.Driver != "mysql" && c.Database.Driver != "postgres" {
		return fmt.Errorf("不支持的数据库驱动: %s", c.Database.Driver)
	}

	// 验证云端配置
	if c.Cloud.Enabled {
		if c.Cloud.Endpoint == "" {
			return fmt.Errorf("云端端点不能为空")
		}
		// API Key现在是可选的：
		// - 使用直接注册方式时，注册成功后才会获得api_key
		// - 使用预注册方式时，激活后才会获得api_key
		// 因此不再强制要求启动时必须配置api_key
		if c.Cloud.CabinetID == "" {
			return fmt.Errorf("储能柜ID不能为空(cloud.cabinet_id)")
		}
	}

	// 验证告警阈值
	if c.Alert.Enabled {
		if err := c.validateAlertThresholds(); err != nil {
			return err
		}
	}

	return nil
}

// validateAlertThresholds 验证告警阈值
func (c *Config) validateAlertThresholds() error {
	t := c.Alert.Thresholds

	// CO2阈值验证
	if t.CO2Max <= 0 || t.CO2Max > 10000 {
		return fmt.Errorf("CO2阈值无效: %f", t.CO2Max)
	}

	// CO阈值验证
	if t.COMax <= 0 || t.COMax > 1000 {
		return fmt.Errorf("CO阈值无效: %f", t.COMax)
	}

	// 烟雾阈值验证
	if t.SmokeMax <= 0 || t.SmokeMax > 5000 {
		return fmt.Errorf("烟雾阈值无效: %f", t.SmokeMax)
	}

	// 液位阈值验证 (传感器刻度: 0-16cm = 0-1600mm)
	if t.LiquidLevelMin < 0 || t.LiquidLevelMax > 2000 {
		return fmt.Errorf("液位阈值无效: min=%f, max=%f (范围: 0-2000mm)", t.LiquidLevelMin, t.LiquidLevelMax)
	}
	if t.LiquidLevelMin >= t.LiquidLevelMax {
		return fmt.Errorf("液位最小值必须小于最大值")
	}

	// 电导率阈值验证
	if t.ConductivityMin < 0 || t.ConductivityMax > 20 {
		return fmt.Errorf("电导率阈值无效: min=%f, max=%f", t.ConductivityMin, t.ConductivityMax)
	}
	if t.ConductivityMin >= t.ConductivityMax {
		return fmt.Errorf("电导率最小值必须小于最大值")
	}

	// 温度阈值验证
	if t.TemperatureMin < -40 || t.TemperatureMax > 125 {
		return fmt.Errorf("温度阈值无效: min=%f, max=%f", t.TemperatureMin, t.TemperatureMax)
	}
	if t.TemperatureMin >= t.TemperatureMax {
		return fmt.Errorf("温度最小值必须小于最大值")
	}

	// 流速阈值验证
	if t.FlowMin < 0 || t.FlowMax > 200 {
		return fmt.Errorf("流速阈值无效: min=%f, max=%f", t.FlowMin, t.FlowMax)
	}
	if t.FlowMin >= t.FlowMax {
		return fmt.Errorf("流速最小值必须小于最大值")
	}

	return nil
}

// GetSensorThreshold 获取传感器阈值
func (c *Config) GetSensorThreshold(sensorType string) (min, max float64, enabled bool) {
	if !c.Alert.Enabled {
		return 0, 0, false
	}

	t := c.Alert.Thresholds
	switch sensorType {
	case "co2":
		return 0, t.CO2Max, true
	case "co":
		return 0, t.COMax, true
	case "smoke":
		return 0, t.SmokeMax, true
	case "liquid_level":
		return t.LiquidLevelMin, t.LiquidLevelMax, true
	case "conductivity":
		return t.ConductivityMin, t.ConductivityMax, true
	case "temperature":
		return t.TemperatureMin, t.TemperatureMax, true
	case "flow":
		return t.FlowMin, t.FlowMax, true
	default:
		return 0, 0, false
	}
}

// IsSensorSupported 检查传感器是否支持
func (c *Config) IsSensorSupported(sensorType string) bool {
	for _, sensor := range c.Device.SupportedSensors {
		if sensor == sensorType {
			return true
		}
	}
	return false
}

// GetCloudConfig 获取云端配置（用于API）
func (c *Config) GetCloudConfig() (enabled bool, endpoint, apiKey, adminToken, cabinetID string) {
	return c.Cloud.Enabled, c.Cloud.Endpoint, c.Cloud.APIKey, c.Cloud.AdminToken, c.Cloud.CabinetID
}

// GetCabinetInfo 获取储能柜详细信息（用于API）
func (c *Config) GetCabinetInfo() (name, location string, latitude, longitude, capacityKWh *float64, deviceModel string) {
	return c.Cloud.CabinetName, c.Cloud.Location, c.Cloud.Latitude, c.Cloud.Longitude, c.Cloud.CapacityKWh, c.Cloud.DeviceModel
}

// UpdateCloudConfig 更新Cloud配置并保存到文件
func (c *Config) UpdateCloudConfig(enabled bool, endpoint string) error {
	c.Cloud.Enabled = enabled
	if endpoint != "" {
		c.Cloud.Endpoint = endpoint
	}

	// 保存到配置文件
	return c.SaveToFile("./configs/config.yaml")
}

// UpdateCloudCredentials 更新Cloud API凭证并保存到文件
// UpdateCloudCredentials 更新Cloud凭证
// Deprecated: API凭证已迁移到数据库存储(cloud_credentials表),不再使用配置文件
// 该方法保留仅用于向后兼容,实际不执行任何操作
func (c *Config) UpdateCloudCredentials(apiKey, apiSecret string) error {
	// 保留空实现,向后兼容
	// 实际凭证存储在数据库中,由storage.SaveCloudCredentials()完成
	return nil
}

// UpdateCabinetID 更新储能柜ID并保存到文件
func (c *Config) UpdateCabinetID(cabinetID string) error {
	if cabinetID == "" {
		return fmt.Errorf("储能柜ID不能为空")
	}
	c.Cloud.CabinetID = cabinetID

	// 保存到配置文件
	return c.SaveToFile("./configs/config.yaml")
}

// UpdateCabinetInfo 更新储能柜详细信息并保存到文件
func (c *Config) UpdateCabinetInfo(cabinetID, name, location string, latitude, longitude, capacityKWh *float64, deviceModel string) error {
	if cabinetID == "" {
		return fmt.Errorf("储能柜ID不能为空")
	}
	c.Cloud.CabinetID = cabinetID
	c.Cloud.CabinetName = name
	c.Cloud.Location = location
	c.Cloud.Latitude = latitude
	c.Cloud.Longitude = longitude
	c.Cloud.CapacityKWh = capacityKWh
	c.Cloud.DeviceModel = deviceModel

	// 保存到配置文件
	return c.SaveToFile("./configs/config.yaml")
}

// SaveToFile 保存配置到文件
// Deprecated: Docker环境中配置文件为只读挂载,不建议运行时修改
// API凭证已迁移到数据库存储,其他配置建议通过环境变量或重新部署修改
func (c *Config) SaveToFile(filename string) error {
	// 在Docker环境中,配置文件通常以只读模式挂载(:ro)
	// 写入操作会失败并返回"read-only file system"错误
	// 保留该方法用于非Docker环境的向后兼容
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		// 忽略只读文件系统错误,因为API凭证已在数据库中
		if strings.Contains(err.Error(), "read-only file system") {
			return nil // 静默失败,不影响程序运行
		}
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// VulnerabilityConfig 脆弱性评估配置
type VulnerabilityConfig struct {
	Enabled              bool                             `yaml:"enabled"`
	AssessmentInterval   time.Duration                    `yaml:"assessment_interval"`
	ScoreChangeThreshold float64                          `yaml:"score_change_threshold"`
	HistoryRetention     time.Duration                    `yaml:"history_retention"`
	Weights              VulnerabilityWeights             `yaml:"weights"`
	Communication        VulnerabilityCommunicationConfig `yaml:"communication"`
	DataAnomaly          VulnerabilityDataAnomalyConfig   `yaml:"data_anomaly"`
}

// VulnerabilityWeights 评分权重配置
type VulnerabilityWeights struct {
	Communication  float64 `yaml:"communication"`
	ConfigSecurity float64 `yaml:"config_security"`
	DataAnomaly    float64 `yaml:"data_anomaly"`
}

// VulnerabilityCommunicationConfig 通信评分配置
type VulnerabilityCommunicationConfig struct {
	LatencyThresholdMs        float64 `yaml:"latency_threshold_ms"`
	PacketLossThreshold       float64 `yaml:"packet_loss_threshold"`
	ReconnectThresholdPerHour int     `yaml:"reconnect_threshold_per_hour"`
}

// VulnerabilityDataAnomalyConfig 数据异常评分配置
type VulnerabilityDataAnomalyConfig struct {
	MissingRateThreshold    float64 `yaml:"missing_rate_threshold"`
	AbnormalValueThreshold  float64 `yaml:"abnormal_value_threshold"`
	AlertFrequencyThreshold int     `yaml:"alert_frequency_threshold"`
}

// MapConfig 地图配置
type MapConfig struct {
	TencentMapKey string `yaml:"tencent_map_key"` // 腾讯地图WebService API密钥
	Enabled       bool   `yaml:"enabled"`         // 是否启用地图功能
}
