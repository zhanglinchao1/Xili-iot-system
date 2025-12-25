/**
 * API响应类型定义
 */

// 统一成功响应格式
export interface SuccessResponse<T = any> {
  success: true;
  data: T;
  message?: string;
}

// 统一错误响应格式
export interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Record<string, any>;
  };
}

// 分页响应格式
export interface PaginatedResponse<T = any> {
  success: true;
  data: T[];
  page: number;
  page_size: number;
  total: number;
  message?: string;
}

// 储能柜类型
export interface Cabinet {
  cabinet_id: string;
  name: string;
  location?: string;
  latitude?: number;    // 纬度坐标
  longitude?: number;   // 经度坐标
  capacity_kwh?: number;
  mac_address: string;
  status: 'pending' | 'active' | 'inactive' | 'offline' | 'maintenance';
  latest_vulnerability_score: number;  // 脆弱性评分
  latest_risk_level: string;  // 风险等级
  vulnerability_updated_at?: string;  // 脆弱性评分更新时间
  last_sync_at?: string;
  activation_status?: string;  // 激活状态
  created_at: string;
  updated_at: string;
}

export interface PreRegisterResponse {
  cabinet_id: string;
  registration_token: string;
  token_expires_at: string;
}

// 储能柜位置信息（用于地图展示）
export interface CabinetLocation {
  cabinet_id: string;
  name: string;
  location?: string;
  latitude?: number;
  longitude?: number;
  status: 'pending' | 'active' | 'inactive' | 'offline' | 'maintenance';
}

// 储能柜统计信息
export interface CabinetStatistics {
  total_cabinets: number;
  active_cabinets: number;       // 激活且同步中的设备数
  offline_cabinets: number;
  inactive_cabinets: number;      // 已停用储能柜数
  pending_cabinets: number;
  maintenance_cabinets: number;
  activated_cabinets: number;     // 已激活储能柜数（activation_status='activated'）
}

// 传感器设备类型
export interface SensorDevice {
  device_id: string;
  cabinet_id: string;
  sensor_type: 'co2' | 'co' | 'smoke' | 'liquid_level' | 'conductivity' | 'temperature' | 'flow';
  name?: string;
  unit?: string;
  status: 'active' | 'inactive' | 'error';
  last_reading_at?: string;
  last_value?: number;
  created_at: string;
  updated_at: string;
}

// 传感器数据类型
export interface SensorData {
  time: string;
  cabinet_id: string;
  device_id: string;
  sensor_type: string;
  value: number;
  unit?: string;
  quality: number;
}

// 最新传感器数据类型（包含设备信息）
export interface LatestSensorData {
  device_id: string;
  sensor_type: string;
  name: string;
  unit: string;
  value: number;
  quality: number;
  status: string;
  timestamp: string;
}

// 流量类型
export interface TrafficSummary {
  cabinet_count: number;
  total_flow_kbps: number;
  avg_latency_ms: number;
  avg_packet_loss: number;
  avg_mqtt_success: number;
  anomaly_count: number;
}

export interface ProtocolSlice {
  name: string;
  value: number;
}

export interface TrafficTrend {
  labels: string[];
  total: number[];
  average: number[];
}

export interface TrafficStat {
  cabinet_id: string;
  location: string;
  timestamp: string;
  flow_kbps: number;
  latency_ms: number;
  packet_loss_rate: number;
  mqtt_success_rate: number;
  reconnection_count: number;
  risk_level: string;
  baseline_deviation: string;
}

export interface TrafficSample {
  cabinet_id: string;
  timestamp: string;
  flow_kbps: number;
}

export interface TrafficDetailResponse {
  stat: TrafficStat;
  history: TrafficSample[];
  protocol: ProtocolSlice[];
}

// 许可证类型
export interface License {
  license_id: string;
  cabinet_id: string;
  mac_address: string;
  issued_at: string;
  expires_at: string;
  revoked_at?: string;
  revoke_reason?: string;
  status: 'active' | 'expired' | 'revoked';
  permissions: string | string[];
  max_devices: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface CreateLicenseRequest {
  cabinet_id: string;
  valid_days: number;
  max_devices: number;
  permissions: string[];
}

export interface RenewLicenseRequest {
  extend_days: number;
}

export interface RevokeLicenseRequest {
  reason: string;
}

// 告警类型
export interface Alert {
  alert_id: number | string;
  cabinet_id: string;
  location?: string;
  edge_alert_id?: number;
  device_id?: string;
  sensor_value?: number;
  alert_type: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  message: string;
  details?: Record<string, any>;
  resolved: boolean;
  resolved_at?: string;
  resolved_by?: string;
  created_at: string;
  status?: 'active' | 'resolved';
}

// 命令类型
export interface Command {
  command_id: string;
  cabinet_id: string;
  command_type: string;
  payload: Record<string, any> | string;
  status: 'pending' | 'sent' | 'success' | 'failed' | 'timeout';
  result?: string;
  sent_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

// 审计日志类型
export interface AuditLog {
  log_id: number;
  user_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  result: 'success' | 'failure';
  details?: Record<string, any>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

// 健康评分类型
export interface HealthScore {
  time: string;
  cabinet_id: string;
  score: number;
  online_rate: number;
  data_quality: number;
  alert_severity_score: number;
  sensor_normalcy: number;
  details?: Record<string, any>;
}

// 前端配置类型
export interface FrontendConfig {
  api_base_url: string;
  polling_interval: number;
  chart_refresh_interval: number;
  page_size: number;
  max_page_size: number;
  frontend?: {
    api_base_url: string;
    polling_interval: number;
    chart_refresh_interval: number;
    page_size: number;
    max_page_size: number;
  };
  map?: MapConfig;
}

// 地图配置类型
export interface MapConfig {
  provider: string;
  tencent_map_key: string;
  tencent_webservice_key: string;
  default_center: {
    latitude: number;
    longitude: number;
  };
  default_zoom: number;
}

// 用户类型
export interface User {
  user_id: string;
  username: string;
  role: string;
}

// 登录请求
export interface LoginRequest {
  username: string;
  password: string;
}

// 登录响应
export interface LoginResponse {
  token: string;
  user: UserInfo;
}

export interface RegisterResponse {
  user_id: number;
}

// 用户信息类型（包含邮箱）
export interface UserInfo {
  id: number;
  username: string;
  email: string;
  role: 'user' | 'admin';
  status?: 'active' | 'disabled';
}

// 更新个人信息请求
export interface UpdateProfileRequest {
  email?: string;
}

// 修改密码请求
export interface UpdatePasswordRequest {
  old_password: string;
  new_password: string;
}

// 创建用户请求（管理员）
export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  role: 'user' | 'admin';
}

// 更新用户请求（管理员）
export interface UpdateUserRequest {
  email?: string;
  role?: 'user' | 'admin';
  status?: 'active' | 'disabled';
}

// 重置密码请求（管理员）
export interface ResetPasswordRequest {
  new_password: string;
}
