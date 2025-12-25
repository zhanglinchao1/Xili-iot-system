# Configuration Design: 统一配置管理系统

**Feature**: Cloud端储能柜集群管理系统  
**Date**: 2025-11-04  
**Phase**: Phase 1 - Design

## 概述

系统采用统一的 `config.yaml` 配置文件管理所有系统参数和业务参数，支持动态配置和热重载。

## 配置结构设计

### 配置文件位置

```
项目根目录/
├── config.yaml              # 默认配置文件
├── config.dev.yaml          # 开发环境配置（可选）
├── config.prod.yaml         # 生产环境配置（可选）
└── config.example.yaml      # 配置示例文件（不包含敏感信息）
```

### 配置文件结构

```yaml
# config.yaml 统一配置结构

# 服务器配置
server:
  port: 8003
  host: 0.0.0.0
  mode: release  # debug, release, test
  read_timeout: 30s
  write_timeout: 30s
  max_header_bytes: 1048576

# 数据库配置
database:
  postgres:
    host: localhost
    port: 5432
    user: cloud_user
    password: your_password
    dbname: cloud_system
    sslmode: disable
    max_connections: 100
    max_idle_connections: 10
    connection_max_lifetime: 3600s
  redis:
    host: localhost
    port: 6379
    password: ""
    db: 0
    pool_size: 10
    min_idle_conns: 5
    dial_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s

# MQTT配置
mqtt:
  broker: tcp://localhost:1883
  username: ""
  password: ""
  client_id: cloud-server
  qos: 1
  clean_session: true
  keep_alive: 60
  reconnect_delay: 5s
  max_reconnect_interval: 300s

# JWT认证配置
jwt:
  secret: your-secret-key-change-in-production
  expiry: 24h
  refresh_expiry: 168h  # 7天

# 日志配置
logging:
  level: info  # debug, info, warn, error
  format: json  # json, text
  output: stdout  # stdout, file, both
  file_path: logs/cloud-server.log
  max_size: 100  # MB
  max_backups: 5
  max_age: 30  # days
  compress: true

# 业务参数配置
business:
  # 健康评分算法权重
  health_score:
    weights:
      online_rate: 0.4      # 设备在线率权重
      data_quality: 0.3     # 数据质量权重
      alert_severity: 0.2   # 告警严重度权重
      sensor_normalcy: 0.1 # 传感器数值正常率权重
    update_interval: 5m    # 健康评分更新间隔
  
  # 告警配置
  alerts:
    offline_timeout: 300   # 设备离线超时时间（秒）
    severity_levels:
      info: 0
      warning: 1
      error: 2
      critical: 3
    retention_days: 90     # 告警保留天数
  
  # 数据同步配置
  sync:
    batch_size: 1000        # 每次同步最大数据量
    sync_interval: 5m       # Edge端同步间隔
    timeout: 30s            # 同步超时时间
  
  # 许可证配置
  license:
    cache_ttl: 3600s        # Redis缓存TTL（秒）
    validation_interval: 1h # Edge端验证间隔
    offline_grace_period: 24h # 离线宽限期
  
  # 命令下发配置
  command:
    timeout: 30s            # 命令超时时间
    retry_count: 3         # 重试次数
    retry_delay: 5s        # 重试延迟
  
  # 前端配置（通过API暴露给前端）
  frontend:
    api_base_url: http://localhost:8003/api/v1
    polling_interval: 5000  # 前端轮询间隔（毫秒）
    chart_refresh_interval: 30000  # 图表刷新间隔（毫秒）
    page_size: 20          # 默认分页大小
    max_page_size: 100     # 最大分页大小

# 监控配置
monitoring:
  enabled: true
  metrics_path: /metrics
  health_check_path: /health
  profiler_enabled: false
  profiler_path: /debug/pprof

# CORS配置
cors:
  enabled: true
  allow_origins:
    - http://localhost:8002
    - http://localhost:3000
  allow_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  allow_headers:
    - Content-Type
    - Authorization
  max_age: 86400  # 24小时
```

## 后端配置实现

### Go后端配置加载

**文件**: `internal/config/config.go`

```go
package config

import (
    "github.com/spf13/viper"
    "sync"
)

type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    MQTT      MQTTConfig
    JWT       JWTConfig
    Logging   LoggingConfig
    Business  BusinessConfig
    Monitoring MonitoringConfig
    CORS      CORSConfig
}

var (
    instance *Config
    once     sync.Once
    mu       sync.RWMutex
)

func Load(configPath string) (*Config, error) {
    viper.SetConfigFile(configPath)
    viper.SetConfigType("yaml")
    
    // 支持环境变量覆盖
    viper.AutomaticEnv()
    viper.SetEnvPrefix("CLOUD")
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }
    
    once.Do(func() {
        instance = &config
    })
    
    return instance, nil
}

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
            log.Error("Failed to reload config", "error", err)
            return
        }
        
        instance = &newConfig
        if callback != nil {
            callback(instance)
        }
    })
    return nil
}
```

### 配置热重载

支持配置文件变更时自动重新加载，无需重启服务：

```go
// 在main.go中启动配置监听
config.WatchConfig(func(cfg *config.Config) {
    log.Info("Configuration reloaded")
    // 可以在这里更新需要重新初始化的组件
    // 例如：更新日志级别、数据库连接池等
})
```

## 前端配置实现

### Vue前端配置加载

**方案1: API端点获取配置（推荐）**

**后端API**: `GET /api/v1/config`

```go
// internal/api/handlers/config.go
func GetConfig(c *gin.Context) {
    cfg := config.Get()
    
    // 只返回前端需要的配置
    frontendConfig := map[string]interface{}{
        "apiBaseUrl": cfg.Business.Frontend.APIBaseURL,
        "pollingInterval": cfg.Business.Frontend.PollingInterval,
        "chartRefreshInterval": cfg.Business.Frontend.ChartRefreshInterval,
        "pageSize": cfg.Business.Frontend.PageSize,
        "maxPageSize": cfg.Business.Frontend.MaxPageSize,
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": frontendConfig,
    })
}
```

**前端配置Store**: `frontend/src/store/config.ts`

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from '@/utils/request'

interface FrontendConfig {
  apiBaseUrl: string
  pollingInterval: number
  chartRefreshInterval: number
  pageSize: number
  maxPageSize: number
}

export const useConfigStore = defineStore('config', () => {
  const config = ref<FrontendConfig | null>(null)
  const loading = ref(false)

  const loadConfig = async () => {
    loading.value = true
    try {
      const response = await axios.get('/config')
      config.value = response.data.data
    } catch (error) {
      console.error('Failed to load config:', error)
    } finally {
      loading.value = false
    }
  }

  return {
    config,
    loading,
    loadConfig,
  }
})
```

**方案2: 构建时注入配置（备选）**

在构建时读取config.yaml并注入到前端代码中：

```typescript
// vite.config.ts
import { readFileSync } from 'fs'
import { parse } from 'yaml'

export default defineConfig({
  define: {
    __APP_CONFIG__: JSON.stringify(
      parse(readFileSync('./config.yaml', 'utf-8')).business.frontend
    ),
  },
})
```

## 环境变量覆盖

支持通过环境变量覆盖配置文件中的值：

```bash
# 环境变量命名规则: CLOUD_<配置路径>，使用下划线分隔
export CLOUD_SERVER_PORT=8080
export CLOUD_DATABASE_POSTGRES_HOST=postgres.example.com
export CLOUD_BUSINESS_HEALTH_SCORE_WEIGHTS_ONLINE_RATE=0.5
```

## 配置验证

启动时验证配置文件的完整性和正确性：

```go
func (c *Config) Validate() error {
    if c.Server.Port <= 0 || c.Server.Port > 65535 {
        return fmt.Errorf("invalid server port: %d", c.Server.Port)
    }
    
    if c.Business.HealthScore.Weights.OnlineRate < 0 || 
       c.Business.HealthScore.Weights.OnlineRate > 1 {
        return fmt.Errorf("invalid health score weight: online_rate must be between 0 and 1")
    }
    
    // 验证所有权重之和为1
    totalWeight := c.Business.HealthScore.Weights.OnlineRate +
                   c.Business.HealthScore.Weights.DataQuality +
                   c.Business.HealthScore.Weights.AlertSeverity +
                   c.Business.HealthScore.Weights.SensorNormalcy
    
    if math.Abs(totalWeight - 1.0) > 0.001 {
        return fmt.Errorf("health score weights must sum to 1.0, got: %f", totalWeight)
    }
    
    return nil
}
```

## 配置管理最佳实践

1. **敏感信息处理**: 
   - 敏感信息（密码、密钥）不应直接写在config.yaml中
   - 使用环境变量或密钥管理服务（如Vault）存储敏感信息
   - config.example.yaml中提供占位符，不包含真实敏感信息

2. **版本控制**:
   - config.yaml应该纳入版本控制
   - config.example.yaml作为模板，包含所有配置项和说明
   - 敏感配置文件（config.prod.yaml）不应纳入版本控制

3. **多环境支持**:
   - 通过环境变量 `CLOUD_ENV` 指定环境（dev, test, prod）
   - 或通过命令行参数指定配置文件路径

4. **动态配置范围**:
   - 系统参数（端口、数据库连接）可以热重载，但需要谨慎处理
   - 业务参数（健康评分权重、告警阈值）支持热重载
   - 数据库连接等关键配置变更后需要重新连接

5. **配置文档**:
   - 每个配置项都应该有清晰的注释说明
   - 维护配置项变更日志

## 任务列表

以下任务需要添加到tasks.md中：

- [ ] T018-CONFIG [P] 创建config.yaml配置文件结构和示例文件
- [ ] T018-CONFIG2 [P] 实现Viper配置加载和解析（支持环境变量覆盖）
- [ ] T018-CONFIG3 [P] 实现配置热重载机制（WatchConfig）
- [ ] T018-CONFIG4 [P] 实现配置验证逻辑
- [ ] T018-CONFIG5 [P] 实现GET /api/v1/config API端点
- [ ] T018-CONFIG6 [P] 创建前端配置Store（Pinia）
- [ ] T018-CONFIG7 [P] 实现前端配置加载和缓存机制

