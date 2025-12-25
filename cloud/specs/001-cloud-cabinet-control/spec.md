# Feature Specification: Cloud端储能柜集群管理系统

**Feature Branch**: `001-cloud-cabinet-control`  
**Created**: 2025-11-04  
**Status**: Draft  
**Input**: User description: "构建cloud端系统，控制多个储能柜（即储能柜edge系统）。 @Cloud"

## Clarifications

### Session 2025-11-04

- Q: MVP功能范围 - 应该实现哪些功能？ → A: 实现所有功能模块的主要功能，包括储能柜管理、数据接收、指令下发、许可证管理、告警接收和基础健康评分。每个功能模块完整实现核心功能，避免过度复杂，后续可迭代升级。
- Q: API错误响应格式 - 应该采用什么格式？ → A: 统一JSON格式，包含错误码、消息和可选的详细信息，使用标准HTTP状态码。
- Q: API速率限制策略 - 应该采用什么策略？ → A: MVP阶段不实现速率限制，后续迭代添加。系统依赖认证和基础监控来防止滥用。
- Q: 健康评分算法的基础权重 - 应该如何分配？ → A: 设备在线率40%，数据质量30%，告警严重度20%，传感器数值正常率10%。MVP阶段完整实现此算法，后续可迭代优化。**传感器数值正常率**定义为:传感器读数在告警阈值正常范围内(未触发告警)的比例。
- Q: API成功响应格式 - 应该采用什么格式？ → A: 统一JSON格式：`{success: true, data: {...}, message: "可选消息"}`，与错误响应格式保持一致。
- Q: 认证机制 - 应该使用什么认证方式？ → A: 使用**JWT (JSON Web Token)** 进行API认证。所有API请求需在HTTP Header中携带`Authorization: Bearer <token>`，token有效期24小时，支持刷新机制。
- Q: 脆弱性评价模块的范围 - 应该实现什么级别的脆弱性检测？ → A: MVP阶段实现基础脆弱性评价:基于规则的安全评估(许可证状态、数据异常、通信异常),预留AI模型接口
- Q: 流量检测功能的范围 - Cloud端应该如何实现流量检测？ → A: Cloud端接收并展示Edge端上报的流量统计数据(连接数、流量量、协议分布、异常流量),提供可视化和告警
- Q: 监控大屏的核心展示指标 - 监控大屏应该展示哪些内容？ → A: 综合展示:储能柜概览(总数/在线/离线)、健康评分分布、实时告警列表、关键传感器趋势图、地理分布地图
- Q: 脆弱性评价的评估维度 - 应该从哪些维度评估安全性？ → A: 多维度评估:许可证合规性(30%)、数据异常检测(25%)、通信异常(25%)、配置安全性(20%),综合评分0-100
- Q: 流量检测的异常判定阈值 - 什么样的流量模式会被判定为异常？ → A: 动态阈值:基于历史7天数据计算基线,偏离基线2倍标准差触发告警;同时设置固定上限作为兜底,管理员可配置

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 注册和监控储能柜 (Priority: P1)

作为运维管理员，我需要能够注册新的储能柜并实时监控其运行状态，以便及时了解所有储能柜的健康状况和位置信息。

**Why this priority**: 这是系统的基础功能，没有储能柜管理就无法进行后续的监控和控制操作。这是MVP的核心价值。

**Independent Test**: 可以独立测试通过创建储能柜、查看储能柜列表、查看单个储能柜详情，验证系统能够正确管理储能柜资产信息。

**Acceptance Scenarios**:

1. **Given** 系统已启动，**When** 管理员创建新的储能柜（提供cabinet_id、位置、容量、MAC地址等信息），**Then** 系统成功创建储能柜并返回储能柜信息
2. **Given** 系统中已有多个储能柜，**When** 管理员查询储能柜列表（可筛选状态、位置），**Then** 系统返回符合条件的储能柜列表，包含每个储能柜的基本信息和健康状态
3. **Given** 储能柜已注册，**When** 管理员查询特定储能柜的详情，**Then** 系统返回储能柜的完整信息，包括设备数量、在线状态、健康评分、最后同步时间
4. **Given** 储能柜已注册并连接到Edge端，**When** Edge端定期同步传感器数据，**Then** 系统接收并存储数据，更新储能柜的状态和健康评分

---

### User Story 2 - 接收和展示传感器数据 (Priority: P1)

作为运维管理员，我需要能够查看储能柜的实时和历史传感器数据，以便了解设备运行情况和趋势。

**Why this priority**: 传感器数据是监控储能柜健康状态的核心数据，管理员需要这些数据来做出运维决策。

**Independent Test**: 可以独立测试通过Edge端发送传感器数据同步请求，然后查询最新数据和历史数据，验证系统能够正确接收、存储和查询传感器数据。

**Acceptance Scenarios**:

1. **Given** Edge端已连接并采集传感器数据，**When** Edge端每5分钟同步一批传感器数据（最多1000条），**Then** 系统接收数据并返回同步成功确认，包括同步的数据条数
2. **Given** 系统已存储传感器数据，**When** 管理员查询储能柜的最新传感器数据，**Then** 系统返回7种传感器的最新读数、单位、状态和质量指标
3. **Given** 系统已存储传感器数据，**When** 管理员查询特定传感器在指定时间范围内的历史数据，**Then** 系统返回时间序列数据，包含平均值、最小值、最大值和统计信息

---

### User Story 3 - 下发指令控制储能柜 (Priority: P1)

作为运维管理员，我需要能够向储能柜Edge端下发配置和许可证更新指令，以便远程管理和控制储能柜配置。

**Why this priority**: 远程控制能力是云边协同的核心价值，允许管理员无需现场操作即可更新配置和管理许可证。

**Independent Test**: 可以独立测试通过向特定储能柜发送配置更新指令（如更新告警阈值），然后验证Edge端接收并执行指令，返回执行结果。

**Acceptance Scenarios**:

1. **Given** 储能柜Edge端已连接，**When** 管理员发送配置更新指令（如更新告警阈值），**Then** 系统通过实时消息通道发送指令，Edge端接收并执行，返回执行结果
2. **Given** 储能柜许可证需要更新，**When** 管理员推送新的许可证信息，**Then** 系统发送许可证更新指令，Edge端更新许可证并缓存
3. **Given** 需要吊销储能柜许可证，**When** 管理员发送许可证吊销指令，**Then** 系统发送吊销指令，Edge端立即停止需要许可证的功能并返回确认

---

### User Story 4 - 管理许可证和访问控制 (Priority: P1)

作为管理员，我需要能够管理储能柜的许可证，包括签发、续期、吊销和权限配置，以便控制储能柜的功能访问权限。

**Why this priority**: 许可证管理是安全和业务控制的核心机制，确保只有授权的储能柜能够使用系统功能。

**Independent Test**: 可以独立测试通过创建许可证、验证许可证有效性、更新许可证权限，验证系统能够正确管理许可证生命周期。

**Acceptance Scenarios**:

1. **Given** 新储能柜已注册，**When** 管理员为储能柜创建许可证（绑定cabinet_id和MAC地址），**Then** 系统创建许可证并返回许可证信息，包括有效期、权限列表
2. **Given** Edge端需要验证许可证，**When** Edge端提交验证请求（cabinet_id和MAC地址），**Then** 系统验证许可证有效性并返回许可证信息，包括权限列表和缓存建议
3. **Given** 许可证即将过期，**When** 管理员延长许可证有效期，**Then** 系统更新许可证过期时间，Edge端下次验证时获得新的有效期
4. **Given** 需要吊销许可证，**When** 管理员吊销许可证（记录原因和操作员），**Then** 系统标记许可证为已吊销，Edge端下次验证时收到吊销通知

---

### User Story 5 - 监控告警和健康评估 (Priority: P1)

作为运维管理员，我需要能够查看储能柜的告警信息和健康评分，以便及时发现和处理异常情况。

**Why this priority**: 告警和健康评估是监控系统的重要组成部分，MVP阶段完整实现主要功能（接收告警、查询告警列表、健康评分计算），后续可迭代优化。

**Independent Test**: 可以独立测试通过Edge端发送告警信息，然后查询告警列表和储能柜健康评分，验证系统能够正确聚合和展示告警数据。

**Acceptance Scenarios**:

1. **Given** Edge端检测到传感器异常，**When** Edge端上报告警信息（告警类型、严重程度、数值），**Then** 系统接收告警并存储，更新储能柜健康评分
2. **Given** 系统中存在多个告警，**When** 管理员查询告警列表（可筛选严重程度、状态），**Then** 系统返回符合条件的告警列表，包含告警详情和关联的储能柜信息
3. **Given** 储能柜已运行一段时间，**When** 管理员查询储能柜健康评分，**Then** 系统返回基于设备在线率、数据质量、告警情况的综合评分（0-100分）

---

### User Story 6 - 监控大屏综合展示 (Priority: P1)

作为运维管理员,我需要能够在监控大屏上一览全局,快速了解所有储能柜的整体运行状态、健康评分分布、实时告警和关键趋势,以便进行全局监控和快速决策。

**Why this priority**: 监控大屏是系统的入口和核心可视化界面,提供全局视角,帮助管理员快速识别问题和做出响应,是运维监控的关键工具。

**Independent Test**: 可以独立测试通过访问监控大屏页面,验证显示储能柜概览统计、健康评分分布图表、实时告警列表、传感器趋势图和地理分布地图,所有数据实时更新。

**Acceptance Scenarios**:

1. **Given** 系统中有多个储能柜运行,**When** 管理员打开监控大屏,**Then** 系统显示储能柜总数、在线数量、离线数量、故障数量等概览统计
2. **Given** 监控大屏已打开,**When** 管理员查看健康评分分布,**Then** 系统显示健康评分的分布图表(优秀/良好/一般/差),并显示各区间的储能柜数量
3. **Given** 系统存在未解决的告警,**When** 管理员查看实时告警列表,**Then** 系统按严重程度排序显示最新的告警信息(最多显示20条),包括告警时间、储能柜ID、告警类型
4. **Given** 系统已采集传感器数据,**When** 管理员查看关键传感器趋势,**Then** 系统显示近24小时的温度、CO2等关键传感器的趋势图表
5. **Given** 储能柜配置了地理位置信息,**When** 管理员查看地理分布地图,**Then** 系统在地图上标注各储能柜的位置,并用颜色区分在线/离线/故障状态

---

### User Story 7 - 脆弱性评价与安全监控 (Priority: P1)

作为安全管理员,我需要能够查看储能柜的脆弱性评分和安全风险详情,以便识别安全隐患并采取防护措施,确保系统安全运行。

**Why this priority**: 脆弱性评价是安全管理的核心功能,帮助管理员主动识别和防范安全风险,保障储能柜系统的安全性和合规性。

**Independent Test**: 可以独立测试通过查看脆弱性评价页面,验证系统显示每个储能柜的安全评分、各维度得分(许可证合规性、数据异常、通信异常、配置安全性)和安全建议。

**Acceptance Scenarios**:

1. **Given** 系统已收集储能柜的运行数据,**When** 管理员查询储能柜的脆弱性评分,**Then** 系统返回综合安全评分(0-100分)和各维度评分:许可证合规性(30%)、数据异常检测(25%)、通信异常(25%)、配置安全性(20%)
2. **Given** 储能柜的许可证已过期,**When** 系统计算脆弱性评分,**Then** 许可证合规性维度得分降低,触发安全告警,提示"许可证已过期"
3. **Given** 储能柜存在异常数据模式(如传感器质量持续<50),**When** 系统评估数据异常维度,**Then** 数据异常检测维度得分降低,标记为"数据异常风险"
4. **Given** 储能柜通信频繁中断(离线次数>5次/天),**When** 系统评估通信异常维度,**Then** 通信异常维度得分降低,提示"通信不稳定"
5. **Given** 储能柜的配置存在安全隐患(如使用默认密钥、未加密传输),**When** 系统评估配置安全性,**Then** 配置安全性维度得分降低,提供安全加固建议

---

### User Story 8 - 流量检测与异常分析 (Priority: P1)

作为网络安全管理员,我需要能够监控储能柜Edge端的网络流量统计和异常检测,以便识别异常流量模式、潜在的网络攻击或设备故障,确保网络安全。

**Why this priority**: 流量检测是网络安全监控的重要手段,帮助管理员及时发现异常流量、DDoS攻击、数据泄露等安全事件,保障系统网络安全。

**Independent Test**: 可以独立测试通过Edge端上报流量统计数据,然后查询流量检测页面,验证系统显示流量趋势、协议分布、异常告警,并能基于历史基线检测流量异常。

**Acceptance Scenarios**:

1. **Given** Edge端已上报流量统计数据(连接数、流量量、协议分布),**When** 管理员查询储能柜的流量统计,**Then** 系统显示近24小时的连接数趋势、流量量趋势(入站/出站)、协议类型分布(MQTT/HTTP/其他)
2. **Given** 系统已积累7天的历史流量数据,**When** 系统计算流量基线,**Then** 系统自动计算各储能柜的流量基线(平均值和标准差),用于异常检测
3. **Given** 储能柜的当前流量偏离基线超过2倍标准差,**When** 系统检测流量异常,**Then** 系统触发WARNING级别告警,记录"流量异常:当前连接数XXX,基线YYY,偏离ZZZ%"
4. **Given** 储能柜的流量超过固定上限(如连接数>1000或流量>100MB/min),**When** 系统检测到超限,**Then** 系统触发CRITICAL级别告警,提示"流量超限,可能遭受攻击"
5. **Given** 管理员需要调整流量异常阈值,**When** 管理员配置自定义阈值(基线偏离倍数、固定上限),**Then** 系统保存配置并应用于后续的异常检测
6. **Given** 储能柜出现非预期协议类型(如SMB、Telnet),**When** 系统检测协议分布异常,**Then** 系统触发告警,标记"异常协议检测:发现非预期协议XXX"

---

### Edge Cases

**边缘案例处理策略**:

1. **Edge端离线检测** (FR-016):
   - **场景**: Edge端超过5分钟未发送数据同步请求
   - **处理**: 系统自动将储能柜状态标记为`offline`，触发INFO级别告警，健康评分下降
   - **容错**: 使用7分钟检测窗口(5分钟同步间隔 + 2分钟网络延迟容错)，避免正常延迟误触发

2. **大批量数据同步** (FR-004):
   - **场景**: 数据同步请求包含接近1000条数据上限
   - **处理**: 使用异步goroutine批量插入TimescaleDB，返回快速确认响应(< 2秒)。如超过1000条，返回400错误拒绝请求
   - **监控**: 记录批量大小到metrics，超过800条时记录WARN日志

3. **许可证验证不匹配** (FR-010):
   - **场景**: Edge端发送的cabinet_id与MAC地址绑定不一致
   - **处理**: 拒绝验证请求，返回401错误，记录安全告警(auth_failed)到audit_logs，通知管理员
   - **缓存**: 验证失败的cabinet_id在Redis中缓存5分钟黑名单，避免重复验证攻击

4. **指令发送超时** (FR-008, FR-009):
   - **场景**: 指令发送后Edge端在30秒内无响应
   - **处理**: 标记指令状态为`timeout`，记录ERROR日志。如指令允许重试(retry=true)，3分钟后自动重试1次，最多重试3次
   - **通知**: 超时后触发告警通知管理员

5. **高并发数据同步负载** (SC-002):
   - **场景**: 多个储能柜(如1000个)同时发送数据同步请求
   - **处理**: 使用Go的goroutine pool(worker pool模式)限制并发数据库写入为200个并发，超出部分排队。连接池配置100个PostgreSQL连接
   - **降级**: 如队列积压超过5000个请求，返回503错误提示稍后重试

6. **许可证过期但继续使用** (FR-010):
   - **场景**: 许可证已过期但Edge端仍在宽限期内(grace_period=72小时)
   - **处理**: 验证请求返回警告但允许通过，响应中包含`warning: "License expired, grace period remaining: 48h"`。超过宽限期后拒绝验证并返回403错误
   - **通知**: 过期前7天、3天、1天发送邮件提醒管理员续期

7. **传感器数据质量异常** (FR-005):
   - **场景**: 传感器数据quality指标 < 50
   - **处理**: 仍然存储数据但标记为低质量，在健康评分计算中降低该传感器数据权重(权重 × quality/100)。quality < 30时触发WARNING级别告警
   - **展示**: 前端显示数据时标注低质量标识，建议管理员检查传感器

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow administrators to register new energy storage cabinets with unique cabinet_id, location, capacity, MAC address, customer information, and installation date
- **FR-002**: System MUST allow administrators to query energy storage cabinet list with filtering by status, location, customer, and pagination support
- **FR-003**: System MUST allow administrators to query detailed information for a specific energy storage cabinet, including device count, online status, health score, and last synchronization time
- **FR-004**: System MUST receive and store sensor data synchronization requests from Edge systems every 5 minutes, accepting up to 1000 sensor data points per request. Each data point MUST include complete timestamp, device_id, sensor_type, value, unit, and quality indicator (0-100). Data is stored in TimescaleDB hypertable for efficient time-series querying
- **FR-006**: System MUST allow administrators to query the latest sensor readings for all 7 sensor types for a specific energy storage cabinet
- **FR-007**: System MUST allow administrators to query historical sensor data for a specific device within a time range, with aggregation options (hourly, daily, or raw data)
- **FR-008**: System MUST send configuration update commands to Edge systems via real-time messaging channel
- **FR-009**: System MUST send license update commands to Edge systems, including license creation, renewal, and revocation
- **FR-010**: System MUST provide license validation API for Edge systems to verify license validity. Validation checks: (1) cabinet_id and MAC address binding match, (2) license not expired (considering grace period), (3) license not revoked. Response includes license status, permissions, and cache TTL. Uses Redis cache (1-hour TTL) for performance
- **FR-011**: System MUST provide license management API for administrators to create, update, renew, and revoke licenses. Operations include: (1) Create license with cabinet_id/MAC binding, expiration date, and permissions, (2) Renew license by extending expiration date, (3) Revoke license with reason and operator tracking. All operations trigger audit logging
- **FR-012**: System MUST maintain license revocation list and include it in license validation responses
- **FR-013**: System MUST receive and store alert information from Edge systems, including alert type, severity, value, threshold, and timestamp
- **FR-014**: System MUST calculate health score for energy storage cabinets using weighted algorithm: device online rate (40%), data quality (30%), alert severity (20%), sensor value normalcy (10%). MVP implementation fully implements this algorithm; weights can be adjusted in future iterations.
- **FR-015**: System MUST allow administrators to query alert list with filtering by severity, status (resolved/unresolved), cabinet, and time range
- **FR-016**: System MUST mark energy storage cabinets as offline when no data synchronization received for more than 7 minutes (5-minute sync interval + 2-minute tolerance for network delays). Offline detection runs every 1 minute, triggers INFO-level alert, and updates cabinet health score
- **FR-017**: System MUST authenticate all API requests using JWT (JSON Web Token) authentication. Clients send token in HTTP Authorization header (`Bearer <token>`). Token validity: 24 hours with refresh mechanism. Edge system API endpoints (sync, license validation) use API keys instead of JWT
- **FR-021**: System MUST return unified JSON error response format with appropriate HTTP status codes. **Complete format example**:
```json
{
  "error": {
    "code": "CABINET_NOT_FOUND",
    "message": "储能柜不存在",
    "details": {
      "cabinet_id": "CAB-001",
      "request_id": "req-123456"
    }
  }
}
```
- **FR-022**: System MUST return unified JSON success response format: `{success: true, data: {...}, message: "optional success message"}` for all successful API responses, maintaining consistency with error response format
- **FR-018**: System MUST log all critical operations including license issuance/revocation, configuration changes, and permission modifications
- **FR-019**: System MUST maintain audit logs with operator information, operation time, operation type, and operation result
- **FR-020**: System MUST support graceful degradation when Edge systems are temporarily unreachable, allowing Edge systems to continue operating independently for at least 24 hours
- **FR-023**: System MUST calculate vulnerability score for energy storage cabinets using multi-dimensional weighted algorithm: license compliance (30%), data anomaly detection (25%), communication anomaly (25%), configuration security (20%). Score range: 0-100, where higher score indicates better security posture
- **FR-024**: System MUST evaluate license compliance dimension by checking: (1) license validity (not expired considering grace period), (2) license not revoked, (3) cabinet_id and MAC address binding match. Non-compliance reduces dimension score and triggers security alerts
- **FR-025**: System MUST evaluate data anomaly dimension by analyzing: (1) sensor data quality trends (sustained quality < 50), (2) missing data patterns (data gaps > 15 minutes), (3) out-of-range sensor values frequency. Anomalies reduce dimension score
- **FR-026**: System MUST evaluate communication anomaly dimension by tracking: (1) offline occurrences (>5 times/day), (2) connection stability (disconnection duration), (3) sync delay patterns. Communication issues reduce dimension score
- **FR-027**: System MUST evaluate configuration security dimension by checking: (1) use of default credentials, (2) unencrypted data transmission, (3) outdated firmware versions, (4) insecure protocol usage. Security gaps reduce dimension score and provide hardening recommendations
- **FR-028**: System MUST receive and store network traffic statistics from Edge systems, including: connection count, traffic volume (inbound/outbound), protocol distribution (MQTT/HTTP/other), timestamp. Data stored in TimescaleDB for time-series analysis
- **FR-029**: System MUST calculate traffic baseline for each cabinet using 7-day historical data, computing mean and standard deviation for connection count and traffic volume. Baseline updated daily
- **FR-030**: System MUST detect traffic anomalies using dynamic threshold: trigger WARNING alert when current traffic deviates from baseline by >2 standard deviations. Alert message includes current value, baseline, and deviation percentage
- **FR-031**: System MUST detect traffic over-limit using fixed threshold: trigger CRITICAL alert when connection count >1000 or traffic volume >100MB/min, indicating potential attack
- **FR-032**: System MUST allow administrators to configure custom traffic thresholds, including baseline deviation multiplier and fixed upper limits. Configuration applies to subsequent anomaly detection
- **FR-033**: System MUST detect unexpected protocol types in traffic distribution. Trigger alert when non-standard protocols (e.g., SMB, Telnet, FTP) are detected, marking as "Unexpected protocol: XXX"
- **FR-034**: System MUST provide monitoring dashboard API for administrators to query: (1) cabinet overview statistics (total/online/offline/fault counts), (2) health score distribution (excellent/good/fair/poor ranges), (3) recent alerts (latest 20), (4) key sensor trends (24-hour data), (5) geographic distribution (if location configured)

**Note**: API rate limiting is deferred to future iterations. MVP relies on authentication and basic monitoring to prevent abuse.

### Key Entities *(include if feature involves data)*

- **Energy Storage Cabinet**: Represents a physical energy storage cabinet managed by the system. Key attributes include unique cabinet_id (also used as Edge system identifier), location, type, capacity, MAC address, status (online/offline/maintenance/fault), customer information, installation date, device count, online device count, health score, and last synchronization timestamp

- **Sensor Device**: Represents a sensor device connected to an energy storage cabinet. Key attributes include device_id, cabinet_id, sensor_type (one of 7 types: co2, co, smoke, liquid_level, conductivity, temperature, flow), status (online/offline/disabled/fault), model, manufacturer, firmware version, last seen timestamp, and last synced timestamp

- **Sensor Data**: Represents a single sensor reading. Key attributes include device_id, sensor_type, value, unit, timestamp, quality indicator (0-100), and sync status

- **Alert**: Represents an alert generated by Edge system or Cloud system. Key attributes include alert_id, cabinet_id, device_id, alert_type (12 types), severity (info/warning/error/critical), message, value, threshold, timestamp, resolved status, and resolved timestamp

- **License**: Represents a license for an energy storage cabinet. Key attributes include license_id, cabinet_id, MAC address binding, max_devices limit, expiration timestamp, grace period, status (active/suspended/expired/revoked), permissions list, and customer information

- **Command**: Represents a command sent from Cloud to Edge system. Key attributes include command_id, command_type, cabinet_id, timestamp, parameters, timeout, retry flag, and execution result

- **Vulnerability Assessment**: Represents a vulnerability/security assessment for an energy storage cabinet. Key attributes include assessment_id, cabinet_id, overall_score (0-100), license_compliance_score, data_anomaly_score, communication_anomaly_score, configuration_security_score, assessment_timestamp, risk_level (low/medium/high/critical), and security recommendations list

- **Traffic Statistics**: Represents network traffic statistics for an energy storage cabinet. Key attributes include stat_id, cabinet_id, connection_count, inbound_traffic_bytes, outbound_traffic_bytes, protocol_distribution (JSON map of protocol:count), timestamp, and anomaly_flag

- **Traffic Baseline**: Represents the calculated traffic baseline for an energy storage cabinet. Key attributes include baseline_id, cabinet_id, connection_count_mean, connection_count_stddev, traffic_volume_mean, traffic_volume_stddev, calculation_period_start, calculation_period_end, and last_updated_timestamp

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Administrators can register a new energy storage cabinet and view its information within 3 seconds of registration completion
- **SC-002**: System can handle data synchronization from 1000 energy storage cabinets concurrently connected via MQTT persistent connections, supporting 2000 QPS for HTTP API requests, without performance degradation (response time remains < 2 seconds for 95% of requests). Connection type: MQTT for real-time commands + HTTP for RESTful API
- **SC-003**: System successfully receives and processes 95% of sensor data synchronization requests from Edge systems within 2 seconds
- **SC-004**: Administrators can query and view the latest sensor readings for any energy storage cabinet within 1 second
- **SC-005**: System successfully delivers configuration and license commands to Edge systems within 5 seconds, with 90% acknowledgment rate
- **SC-006**: License validation requests complete within 200 milliseconds for 95% of requests (using caching when appropriate)
- **SC-007**: Administrators can query alert list for a specific energy storage cabinet and view all unresolved alerts within 1 second
- **SC-008**: System calculates and updates health scores for all energy storage cabinets within 30 seconds of receiving new sensor data
- **SC-009**: System maintains 99.9% uptime (annual downtime less than 8.76 hours), ensuring continuous monitoring capability
- **SC-010**: System correctly identifies and marks offline energy storage cabinets (no data received for more than 5 minutes) within 1 minute of last data reception
- **SC-011**: All critical operations (license issuance, revocation, configuration changes) are logged and queryable within 1 second
- **SC-012**: System supports graceful degradation when Edge systems are temporarily unreachable, allowing Edge systems to continue operating independently for at least 24 hours without Cloud connectivity

## Assumptions

- Edge systems are capable of connecting to Cloud system via network (HTTP/HTTPS and MQTT)
- Edge systems implement the CloudSyncPayload data structure for data synchronization
- Edge systems can receive and process commands sent via real-time messaging channel
- Each energy storage cabinet corresponds to exactly one Edge system and one gateway device
- Edge systems support 7 types of sensors: CO2, CO, smoke, liquid level, conductivity, temperature, and flow
- Administrators have appropriate authentication credentials to access the system
- Network connectivity between Cloud and Edge systems is generally stable, with occasional temporary outages
- Edge systems can operate independently for at least 24 hours when Cloud is unreachable

## Dependencies

- Edge systems must be deployed and operational before Cloud system can manage them
- Network infrastructure must support HTTP/HTTPS and MQTT communication protocols
- Authentication and authorization mechanisms must be in place before system goes live
- Database systems must be configured and accessible before storing sensor data and cabinet information

## Out of Scope

- Direct sensor data collection (handled by gateway devices)
- Battery Management System (BMS) management (handled by Edge systems)
- Real-time control commands for individual devices (handled by Edge systems, Cloud only manages policies)
- Neural network-based vulnerability detection model (MVP implements rule-based assessment; AI model is future enhancement)
- Firmware upgrade functionality (future enhancement)
- Deep packet inspection (DPI) for traffic analysis (MVP implements statistical monitoring; DPI is future enhancement)

## Frontend Implementation

**注意**: 虽然本规格主要定义后端API功能,但系统包含前端用户界面以提供完整的用户体验。前端与后端并行开发,作为同一feature的一部分交付。

**前端技术栈**:
- Vue.js 3 + TypeScript
- Element Plus (UI组件库)
- ECharts (数据可视化)
- Pinia (状态管理)
- Vue Router (路由)
- Vite (构建工具)

**前端导航菜单结构**(按用户需求定制):

1. **监控大屏** (`/dashboard`)
   - 首页/默认页面
   - 显示储能柜概览统计(总数/在线/离线/故障)
   - 健康评分分布图表
   - 实时告警列表(最新20条)
   - 关键传感器趋势图(24小时)
   - 地理分布地图(如配置位置信息)

2. **储能柜管理** (`/cabinets`)
   - 储能柜列表页面(支持筛选、搜索、分页)
   - 储能柜详情页面(设备信息、传感器数据、健康评分、历史数据图表)
   - 储能柜创建/编辑页面
   - 传感器数据展示(最新数据、历史趋势)

3. **许可控制** (`/licenses`)
   - 许可证列表页面(显示所有许可证状态)
   - 许可证创建页面(绑定cabinet_id和MAC地址)
   - 许可证详情/编辑页面(续期、吊销操作)
   - 许可证吊销列表
   - 指令下发界面(许可证更新/吊销指令)

4. **脆弱性评价** (`/vulnerability`)
   - 脆弱性评分总览(所有储能柜的安全评分分布)
   - 储能柜脆弱性详情(综合评分、各维度评分、风险等级)
   - 安全风险列表(按风险等级排序)
   - 安全加固建议(针对配置安全性问题)
   - 评估历史记录

5. **流量检测** (`/traffic`)
   - 流量统计概览(所有储能柜的流量趋势)
   - 单个储能柜流量详情(连接数趋势、流量量趋势、协议分布)
   - 流量基线展示(当前值 vs 基线对比)
   - 流量异常告警列表
   - 流量阈值配置(基线偏离倍数、固定上限)

6. **监控告警** (`/alerts`)
   - 告警列表页面(支持按严重程度、状态、储能柜、时间范围筛选)
   - 告警详情页面(告警信息、关联储能柜、传感器数据)
   - 告警处理操作(标记已解决、添加备注)
   - 告警统计图表(按类型、严重程度、时间分布)

**前端路由结构**:
```
/login               - 登录页面
/register            - 注册页面
/                    - 重定向到 /dashboard
/dashboard           - 监控大屏
/cabinets            - 储能柜列表
/cabinets/:id        - 储能柜详情
/cabinets/create     - 创建储能柜
/sensors             - 传感器数据(已移除,合并到储能柜详情)
/licenses            - 许可证管理
/licenses/:id        - 许可证详情
/vulnerability       - 脆弱性评价
/vulnerability/:id   - 储能柜脆弱性详情
/traffic             - 流量检测
/traffic/:id         - 储能柜流量详情
/alerts              - 监控告警
/alerts/:id          - 告警详情
```

前端与后端通过RESTful API通信,遵循contracts/openapi.yaml定义的API契约。详细实现任务见tasks.md中的前端任务(标记为*-F)。
