# Research: Cloud端储能柜集群管理系统

**Feature**: Cloud端储能柜集群管理系统  
**Date**: 2025-11-04  
**Phase**: 0 - Research & Technology Selection

## Technology Decisions

### Web Framework: Gin vs Fiber

**Decision**: Gin (github.com/gin-gonic/gin)

**Rationale**: 
- Gin是Go生态中最成熟和广泛使用的Web框架之一
- 性能优秀，中间件生态丰富
- 文档完善，社区支持好
- 与项目原则中的"轻量级、高性能"要求匹配
- 团队成员熟悉度更高

**Alternatives Considered**:
- Fiber: 性能略好，但相对较新，生态不如Gin成熟
- Echo: 功能丰富但相对较重

### Database: PostgreSQL + InfluxDB/TimescaleDB

**Decision**: PostgreSQL + TimescaleDB

**Rationale**:
- TimescaleDB是PostgreSQL的扩展，可以同时存储关系数据和时序数据
- 统一的数据库管理，降低运维复杂度
- 利用PostgreSQL的强大功能（事务、ACID、复杂查询）
- 与项目原则中的"时序数据库"要求匹配
- 社区成熟，文档完善

**Alternatives Considered**:
- PostgreSQL + InfluxDB: 需要维护两个数据库系统，运维复杂度更高
- 纯PostgreSQL: 对于大规模时序数据查询性能可能不足

### MQTT Client Library

**Decision**: eclipse/paho.mqtt.golang

**Rationale**:
- Eclipse Paho是MQTT的官方实现，标准兼容性好
- 功能完整，支持QoS、自动重连等特性
- 社区活跃，维护良好
- 与项目原则中的"MQTT实时指令"要求匹配

**Alternatives Considered**:
- paho.mqtt.golang的其他fork: 稳定性不如官方版本

### Authentication: JWT vs API Key

**Decision**: JWT Token (golang-jwt/jwt)

**Rationale**:
- JWT支持无状态认证，适合分布式系统
- 包含过期时间、权限信息等，更灵活
- 项目原则要求JWT Token或API Key，JWT更符合现代API设计
- golang-jwt/jwt是Go生态中最成熟的JWT库

**Alternatives Considered**:
- API Key: 简单但缺少过期机制，不适合长期运行的系统
- Session: 需要服务器端存储，增加复杂度

### Logging: Zap

**Decision**: uber-go/zap (结构化日志)

**Rationale**:
- 高性能结构化日志库
- 与项目原则中的"结构化日志"要求完全匹配
- 支持多种日志级别和输出格式
- 生产环境广泛使用

**Alternatives Considered**:
- logrus: 功能类似但性能略低
- 标准log包: 缺少结构化日志支持

### Configuration Management

**Decision**: 统一使用 config.yaml 配置文件（后端Viper + 前端动态加载）

**Rationale**:
- **统一配置源**: 后端和前端都使用config.yaml，便于统一管理和版本控制
- **动态配置**: 支持配置文件热重载，无需重启服务即可更新配置
- **系统参数和业务参数分离**: 系统参数（数据库、服务器等）和业务参数（健康评分权重、告警阈值等）都可通过config.yaml配置
- **多环境支持**: 通过不同配置文件（config.dev.yaml, config.prod.yaml）或环境变量覆盖支持多环境
- **前端配置**: Vue前端通过API获取配置或直接读取config.yaml（构建时），支持运行时动态配置
- **便于运维**: 所有配置集中在一个文件中，便于运维人员管理和修改
- **符合用户需求**: 满足"所有配置参数放在config.yaml中，实现动态配置系统和业务参数"的要求

**配置结构**:
```yaml
# config.yaml 统一配置结构
server:
  port: 8003
  host: 0.0.0.0
  
database:
  postgres:
    host: localhost
    port: 5432
    # ...
  redis:
    host: localhost
    port: 6379
    # ...
    
mqtt:
  broker: tcp://localhost:1883
  # ...
  
business:
  health_score:
    weights:
      online_rate: 0.4
      data_quality: 0.3
      alert_severity: 0.2
      sensor_normalcy: 0.1
  alerts:
    thresholds:
      offline_timeout: 300  # 秒
      # ...
  frontend:
    api_base_url: http://localhost:8003/api/v1
    polling_interval: 5000  # 毫秒
    # ...
```

**实现方式**:
- **后端**: 使用Viper读取config.yaml，支持环境变量覆盖，支持配置热重载（WatchConfig）
- **前端**: 通过API端点（/api/v1/config）获取配置，或在构建时注入config.yaml内容，支持运行时更新

**Alternatives Considered**:
- Viper + 环境变量: 配置分散，不利于统一管理
- 纯环境变量: 不适合复杂的业务参数配置
- 数据库配置: 增加数据库依赖，启动慢
- 前端.env文件: 构建时固定，不支持运行时动态配置

### Frontend Framework: Vue.js 3 + Element Plus

**Decision**: Vue.js 3 + TypeScript + Element Plus + ECharts

**Rationale**:
- Vue.js 3学习曲线平缓，开发效率高，适合快速迭代
- Element Plus提供完整的后台管理组件，减少重复开发
- ECharts集成简单，适合IoT监控大屏和数据可视化场景
- TypeScript提供类型安全，提高代码质量和可维护性
- Pinia状态管理轻量级，适合中小型应用
- Vite构建工具性能优秀，开发体验好
- 社区活跃，文档完善，符合"完整实现基本功能，后续可迭代升级"的要求
- 适合中后台管理系统和实时监控场景

**技术栈组成**:
- 前端框架: Vue.js 3 + TypeScript
- UI组件库: Element Plus
- 图表库: ECharts (实时监控大屏、数据可视化)
- HTTP客户端: Axios
- 状态管理: Pinia
- 路由: Vue Router
- 构建工具: Vite

**Alternatives Considered**:
- React + Ant Design: 性能优秀但学习曲线较陡，适合大型团队
- 原生HTML/CSS/JS: 开发效率低，不适合复杂交互和数据可视化需求
- Go模板渲染: 快速但扩展性差，不适合后续迭代升级

## Architecture Patterns

### Layered Architecture

**Decision**: 采用三层架构（API层、Service层、Repository层）

**Rationale**:
- 职责清晰，便于测试和维护
- 符合Go项目最佳实践
- Service层包含业务逻辑，便于单元测试
- Repository层抽象数据访问，便于切换数据源

### Error Handling Pattern

**Decision**: 自定义错误类型 + 统一错误响应格式

**Rationale**:
- 符合Go error处理最佳实践
- 统一错误响应格式符合项目原则要求
- 便于错误追踪和日志记录
- 客户端可以统一处理错误

### Health Score Algorithm

**Decision**: 加权平均算法（在线率40%，数据质量30%，告警20%，传感器正常率10%）

**Rationale**:
- 算法简单、可解释，符合项目原则要求
- 权重可配置，后续可调整
- 计算效率高，适合实时计算
- 符合IoT监控系统的常见实践

## Integration Patterns

### Cloud-Edge Communication

**Decision**: HTTP批量同步 + MQTT实时指令

**Rationale**:
- HTTP适合批量数据传输（每5分钟一次）
- MQTT适合实时指令下发（秒级延迟）
- 符合项目原则中的"批量同步和实时指令"要求
- 混合架构平衡了性能和可靠性

### Data Synchronization Pattern

**Decision**: Edge端主动推送 + Cloud端接收确认

**Rationale**:
- Edge端控制同步时机，降低Cloud端压力
- 支持断点续传（通过sync标记）
- 符合项目原则中的"数据驱动"要求
- 容错性好，网络中断后可重试

### License Validation Pattern

**Decision**: Edge端定期验证 + Redis缓存

**Rationale**:
- 缓存减少数据库压力，符合性能要求
- 定期验证确保许可证状态及时更新
- 支持离线模式（Edge端可缓存24小时）
- 符合项目原则中的"缓存策略"要求

## Performance Considerations

### Database Indexing Strategy

**Decision**: 
- PostgreSQL: 在cabinet_id, device_id, timestamp上建立索引
- TimescaleDB: 使用Hypertable自动分区（按时间）

**Rationale**:
- 索引优化查询性能，符合性能目标
- Hypertable自动分区提高时序数据查询效率
- 符合项目原则中的"数据库优化"要求

### Caching Strategy

**Decision**: 
- Redis缓存许可证信息（TTL: 1小时）
- Redis缓存会话数据（TTL: 24小时）

**Rationale**:
- 缓存热点数据，减少数据库压力
- 合理的TTL平衡了数据新鲜度和缓存命中率
- 符合项目原则中的"缓存策略"要求

### Async Processing

**Decision**: 使用goroutine处理数据同步和告警生成

**Rationale**:
- Go的goroutine轻量级，适合并发处理
- 异步处理避免阻塞主流程
- 符合项目原则中的"异步处理"要求

## Security Considerations

### API Authentication

**Decision**: JWT Token in Authorization header

**Rationale**:
- 标准HTTP认证方式
- 支持Token过期和刷新
- 符合项目原则中的"API安全"要求

### Data Encryption

**Decision**: 
- TLS加密所有网络通信（HTTPS、MQTT over TLS）
- 敏感数据（MAC地址、许可证）在数据库中加密存储

**Rationale**:
- TLS保护传输安全
- 数据库加密保护存储安全
- 符合项目原则中的"数据安全"要求

### Rate Limiting

**Decision**: MVP阶段不实现，后续迭代添加

**Rationale**:
- 符合澄清中的决定
- 通过认证和监控防止滥用
- 后续可根据实际需求添加

## Testing Strategy

### Unit Testing

**Decision**: Go标准testing包 + testify

**Rationale**:
- 标准库满足基本需求
- testify提供丰富的断言，提高测试可读性
- 符合项目原则中的"测试驱动开发"要求

### Integration Testing

**Decision**: 使用testcontainers或mock依赖

**Rationale**:
- testcontainers可以启动真实的数据库容器
- mock适合快速测试
- 符合项目原则中的"集成测试"要求

### Contract Testing

**Decision**: 使用OpenAPI规范定义API合约

**Rationale**:
- OpenAPI是标准API规范格式
- 可以生成客户端和服务端代码
- 符合项目原则中的"合约测试"要求

## Deployment Considerations

### Containerization

**Decision**: Docker + Docker Compose（开发环境）

**Rationale**:
- 容器化便于部署和扩展
- Docker Compose适合本地开发环境
- 符合项目原则中的"容器化部署"要求

### Unified Configuration Management

**Decision**: 统一使用 config.yaml 配置文件，支持动态配置系统和业务参数

**Rationale**:
- **统一配置**: 后端和前端都使用config.yaml，配置集中管理
- **动态配置**: 支持配置文件热重载和运行时配置更新
- **业务参数**: 健康评分权重、告警阈值等业务参数可通过配置文件动态调整
- **多环境**: 通过不同配置文件或环境变量覆盖支持多环境部署
- **前端配置**: Vue前端通过API获取配置，支持运行时动态更新
- **便于运维**: 所有配置集中在一个文件中，便于管理和修改

## Frontend-Backend Integration

### API Communication Pattern

**Decision**: RESTful API + Axios + JWT Token

**Rationale**:
- RESTful API符合后端设计，与OpenAPI规范一致
- Axios支持请求拦截器，便于统一处理认证和错误
- JWT Token在HTTP Header中传递，与后端认证机制一致
- 统一的JSON响应格式便于前端错误处理

### Real-time Data Update Pattern

**Decision**: WebSocket或轮询（后续迭代）

**Rationale**:
- MVP阶段使用轮询方式获取最新数据
- 后续迭代可升级为WebSocket实现实时推送
- 符合"完整实现基本功能，后续可迭代升级"的原则

## Outstanding Questions

无剩余问题，所有技术决策已明确。

