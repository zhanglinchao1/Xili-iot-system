/**
 * API接口封装
 * 统一管理所有API请求
 */

import { request } from '@/utils/request';
import type {
  SuccessResponse,
  PaginatedResponse,
  Cabinet,
  CabinetLocation,
  CabinetStatistics,
  SensorDevice,
  SensorData,
  LatestSensorData,
  License,
  CreateLicenseRequest,
  RenewLicenseRequest,
  RevokeLicenseRequest,
  Alert,
  Command,
  AuditLog,
  FrontendConfig,
  LoginRequest,
  LoginResponse,
  RegisterResponse,
  TrafficSummary,
  TrafficTrend,
  ProtocolSlice,
  TrafficStat,
  TrafficDetailResponse,
  UserInfo,
  UpdateProfileRequest,
  UpdatePasswordRequest,
  CreateUserRequest,
  UpdateUserRequest,
  ResetPasswordRequest,
} from '@/types/api';

// ========== 配置相关 ==========
export const configApi = {
  // 获取前端配置
  getConfig(): Promise<SuccessResponse<FrontendConfig>> {
    return request.get('/config');
  },
};

// ========== 认证相关 ==========
export const authApi = {
  // 注册
  register(data: { username: string; email: string; password: string }): Promise<SuccessResponse<RegisterResponse>> {
    return request.post('/auth/register', data);
  },

  // 登录
  login(data: LoginRequest): Promise<SuccessResponse<LoginResponse>> {
    return request.post('/auth/login', data);
  },

  // 刷新Token
  refreshToken(): Promise<SuccessResponse<{ token: string }>> {
    return request.post('/auth/refresh');
  },

  // 登出
  logout(): Promise<SuccessResponse<null>> {
    return request.post('/auth/logout');
  },
};

// ========== 储能柜相关 ==========
export const cabinetApi = {
  // 获取储能柜列表
  list(params?: { page?: number; page_size?: number; status?: string }): Promise<PaginatedResponse<Cabinet>> {
    return request.get('/cabinets', { params });
  },

  // 获取所有储能柜位置信息（用于地图展示）
  getLocations(): Promise<SuccessResponse<CabinetLocation[]>> {
    return request.get('/cabinets/locations');
  },

  // 获取储能柜统计信息
  getStatistics(): Promise<SuccessResponse<CabinetStatistics>> {
    return request.get('/cabinets/statistics');
  },

  // 获取储能柜详情
  get(cabinetId: string): Promise<SuccessResponse<Cabinet>> {
    return request.get(`/cabinets/${cabinetId}`);
  },

  // 创建储能柜
  create(data: Partial<Cabinet>): Promise<SuccessResponse<Cabinet>> {
    return request.post('/cabinets', data);
  },

  // 预注册储能柜
  preRegister(data: {
    cabinet_id: string;
    name: string;
    location?: string;
    capacity_kwh?: number;
    mac_address: string;
    license_expires_at?: string;
    permissions?: string[];
    ip_address?: string;
    device_model?: string;
    notes?: string;
  }): Promise<SuccessResponse<{
    cabinet_id: string;
    registration_token: string;
    token_expires_at: string;
  }>> {
    return request.post('/cabinets/pre-register', data);
  },

  // 获取激活信息
  getActivationInfo(cabinetId: string): Promise<SuccessResponse<{
    cabinet_id: string;
    name: string;
    mac_address: string;
    registration_token: string;
    token_expires_at: string;
    token_expired: boolean;
    cloud_api_url: string;
  }>> {
    return request.get(`/cabinets/${cabinetId}/activation-info`);
  },

  // 重新生成注册Token
  regenerateToken(cabinetId: string): Promise<SuccessResponse<{
    registration_token: string;
    token_expires_at: string;
  }>> {
    return request.post(`/cabinets/${cabinetId}/regenerate-token`);
  },

  // Edge端激活储能柜
  activate(data: {
    registration_token: string;
    mac_address: string;
  }): Promise<SuccessResponse<{
    cabinet_id: string;
    api_key: string;
    api_secret: string;
  }>> {
    return request.post('/cabinets/activate', data);
  },

  // 更新储能柜
  update(cabinetId: string, data: Partial<Cabinet>): Promise<SuccessResponse<Cabinet>> {
    return request.put(`/cabinets/${cabinetId}`, data);
  },

  // 删除储能柜
  delete(cabinetId: string): Promise<SuccessResponse<null>> {
    return request.delete(`/cabinets/${cabinetId}`);
  },

  // 获取储能柜最新传感器数据
  getLatestSensorData(cabinetId: string): Promise<SuccessResponse<LatestSensorData[]>> {
    return request.get(`/cabinets/${cabinetId}/sensors/latest`);
  },

  // 获取储能柜传感器设备
  getDevices(cabinetId: string): Promise<SuccessResponse<SensorDevice[]>> {
    return request.get(`/cabinets/${cabinetId}/devices`);
  },

  // 获取储能柜告警
  getAlerts(cabinetId: string, params?: { page?: number; page_size?: number }): Promise<PaginatedResponse<Alert>> {
    return request.get(`/cabinets/${cabinetId}/alerts`, { params });
  },

  // 获取储能柜健康评分（实时计算）
  getHealthScore(cabinetId: string): Promise<SuccessResponse<{ cabinet_id: string; health_score: number }>> {
    return request.get(`/cabinets/${cabinetId}/health-score`);
  },

  // ========== API Key管理 ==========
  // 获取API Key信息(脱敏显示)
  getAPIKeyInfo(cabinetId: string): Promise<SuccessResponse<{
    cabinet_id: string;
    activation_status: string;
    has_api_key: boolean;
    api_key_masked?: string;
    generated_at?: string;
  }>> {
    return request.get(`/cabinets/${cabinetId}/api-key`);
  },

  // 重新生成API Key
  regenerateAPIKey(cabinetId: string): Promise<SuccessResponse<{
    cabinet_id: string;
    api_key: string;
  }>> {
    return request.post(`/cabinets/${cabinetId}/api-key/regenerate`);
  },

  // 撤销API Key
  revokeAPIKey(cabinetId: string): Promise<SuccessResponse<{ message: string }>> {
    return request.delete(`/cabinets/${cabinetId}/api-key`);
  },
};

// ========== 传感器设备相关 ==========
export const deviceApi = {
  // 获取设备详情
  get(deviceId: string): Promise<SuccessResponse<SensorDevice>> {
    return request.get(`/devices/${deviceId}`);
  },
  
  // 获取设备历史数据
  getData(deviceId: string, params?: { start_time?: string; end_time?: string }): Promise<SuccessResponse<SensorData[]>> {
    return request.get(`/devices/${deviceId}/data`, { params });
  },
};

// ========== 传感器数据相关 ==========
export const sensorApi = {
  getHistoricalData(params: {
    device_id: string;
    start_time?: string;
    end_time?: string;
    aggregation?: string;
    page?: number;
    page_size?: number;
  }): Promise<PaginatedResponse<SensorData>> {
    return request.get('/devices/data', { params });
  },
};

// ========== 许可证相关 ==========
export const licenseApi = {
  list(params?: { page?: number; page_size?: number; status?: string }): Promise<PaginatedResponse<License>> {
    return request.get('/licenses', { params });
  },

  get(cabinetId: string): Promise<SuccessResponse<License>> {
    return request.get(`/licenses/${cabinetId}`);
  },

  create(data: CreateLicenseRequest): Promise<SuccessResponse<License>> {
    return request.post('/licenses', data);
  },

  renew(cabinetId: string, data: RenewLicenseRequest): Promise<SuccessResponse<null>> {
    return request.put(`/licenses/${cabinetId}`, data);
  },

  revoke(cabinetId: string, data: RevokeLicenseRequest): Promise<SuccessResponse<null>> {
    return request.post(`/licenses/${cabinetId}/revoke`, data);
  },

  delete(cabinetId: string): Promise<SuccessResponse<null>> {
    return request.delete(`/licenses/${cabinetId}`);
  },

  push(cabinetId: string): Promise<SuccessResponse<any>> {
    return request.post(`/licenses/${cabinetId}/push`);
  },

  sync(data?: { cabinet_ids?: string[]; valid_days?: number; max_devices?: number; permissions?: string[] }): Promise<SuccessResponse<{ created: number }>> {
    return request.post('/licenses/sync', data);
  },
};

// ========== 命令相关 ==========
export const commandApi = {
  // 获取命令列表
  list(params?: {
    page?: number;
    page_size?: number;
    cabinet_id?: string;
    status?: string;
    command_type?: string;
  }): Promise<PaginatedResponse<Command>> {
    return request.get('/commands', { params });
  },
  
  // 发送命令
  send(data: { cabinet_id: string; command_type: string; payload: Record<string, any> }): Promise<SuccessResponse<Command>> {
    const { cabinet_id, ...body } = data;
    return request.post(`/commands/${cabinet_id}`, body);
  },
  
  // 获取命令状态
  getStatus(commandId: string): Promise<SuccessResponse<Command>> {
    return request.get(`/commands/${commandId}`);
  },
};

// ========== 告警相关 ==========
export const alertApi = {
  // 获取告警列表
  list(params?: {
    page?: number;
    page_size?: number;
    severity?: string;
    status?: string;
    cabinet_id?: string;
  }): Promise<PaginatedResponse<Alert>> {
    return request.get('/alerts', { params });
  },

  // 获取告警详情
  get(alertId: string): Promise<SuccessResponse<Alert>> {
    return request.get(`/alerts/${alertId}`);
  },

  // 解决单个告警
  resolve(alertId: string, data?: { resolved_by?: string }): Promise<SuccessResponse<Alert>> {
    return request.put(`/alerts/${alertId}/resolve`, data);
  },

  // 批量解决告警
  batchResolve(alertIds: string[]): Promise<SuccessResponse<{ resolved_count: number }>> {
    return request.post('/alerts/batch-resolve', { alert_ids: alertIds });
  },
};

// ========== 审计日志相关 ==========
export const auditApi = {
  // 获取审计日志列表
  list(params?: { page?: number; page_size?: number; user_id?: string; action?: string }): Promise<PaginatedResponse<AuditLog>> {
    return request.get('/audit-logs', { params });
  },
};

// ========== 脆弱性评价相关 ==========
export const vulnerabilityApi = {
  // 获取脆弱性评估列表(所有储能柜)
  listAssessments(params?: {
    page?: number;
    page_size?: number;
    risk_level?: string;
    cabinet_id?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<SuccessResponse<{
    items: any[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
  }>> {
    return request.get('/vulnerability/assessments', { params });
  },

  // 获取评估详情
  getAssessmentDetail(assessmentId: number): Promise<SuccessResponse<{
    assessment: any;
    events: any[];
  }>> {
    return request.get(`/vulnerability/assessments/${assessmentId}`);
  },

  // 获取储能柜最新评估
  getLatestAssessment(cabinetId: string): Promise<SuccessResponse<{
    assessment: any;
    events: any[];
  }>> {
    return request.get(`/cabinets/${cabinetId}/vulnerability/latest`);
  },

  // 获取储能柜评估历史
  getHistory(cabinetId: string, params?: {
    start_time?: string;
    end_time?: string;
    limit?: number;
  }): Promise<SuccessResponse<any[]>> {
    return request.get(`/cabinets/${cabinetId}/vulnerability/history`, { params });
  },

  // 获取储能柜评估统计
  getStats(cabinetId: string, params?: { days?: number }): Promise<SuccessResponse<{
    total_assessments: number;
    avg_score: number;
    min_score: number;
    max_score: number;
    risk_distribution: {
      critical: number;
      high: number;
      medium: number;
      low: number;
      healthy: number;
    };
  }>> {
    return request.get(`/cabinets/${cabinetId}/vulnerability/stats`, { params });
  },
};

// ========== 流量检测相关 ==========
export const trafficApi = {
  getSummary(params?: { range?: string }): Promise<SuccessResponse<{ summary: TrafficSummary; trend: TrafficTrend; protocol: ProtocolSlice[] }>> {
    return request.get('/traffic/summary', { params });
  },

  listCabinets(): Promise<SuccessResponse<TrafficStat[]>> {
    return request.get('/traffic/cabinets');
  },

  getCabinetDetail(cabinetId: string, params?: { range?: string }): Promise<SuccessResponse<TrafficDetailResponse>> {
    return request.get(`/traffic/cabinets/${cabinetId}`, { params });
  },
};

// ========== 地图相关 ==========
export const mapApi = {
  // 搜索地点
  searchLocation(keyword: string): Promise<SuccessResponse<{
    results: Array<{
      id: string;
      title: string;
      address: string;
      location: { lat: number; lng: number };
    }>;
  }>> {
    return request.get('/map/search', { params: { keyword } });
  },

  // 逆地理编码(坐标转地址)
  reverseGeocode(latitude: number, longitude: number): Promise<SuccessResponse<{
    title: string;
    address: string;
  }>> {
    return request.get('/map/geocode/reverse', { params: { latitude, longitude } });
  },
};

// ========== 用户管理相关 ==========
export const userApi = {
  // 获取个人信息
  getProfile(): Promise<SuccessResponse<UserInfo>> {
    return request.get('/users/profile');
  },

  // 更新个人信息
  updateProfile(data: UpdateProfileRequest): Promise<SuccessResponse<null>> {
    return request.put('/users/profile', data);
  },

  // 修改密码
  updatePassword(data: UpdatePasswordRequest): Promise<SuccessResponse<null>> {
    return request.put('/users/password', data);
  },

  // 获取用户列表（管理员）
  listUsers(params?: { page?: number; page_size?: number; role?: string; status?: string }): Promise<PaginatedResponse<UserInfo>> {
    return request.get('/users', { params });
  },

  // 获取用户详情（管理员）
  getUser(userId: number): Promise<SuccessResponse<UserInfo>> {
    return request.get(`/users/${userId}`);
  },

  // 创建用户（管理员）
  createUser(data: CreateUserRequest): Promise<SuccessResponse<UserInfo>> {
    return request.post('/users', data);
  },

  // 更新用户（管理员）
  updateUser(userId: number, data: UpdateUserRequest): Promise<SuccessResponse<null>> {
    return request.put(`/users/${userId}`, data);
  },

  // 重置用户密码（管理员）
  resetUserPassword(userId: number, data: ResetPasswordRequest): Promise<SuccessResponse<null>> {
    return request.put(`/users/${userId}/reset-password`, data);
  },

  // 删除用户（管理员）
  deleteUser(userId: number): Promise<SuccessResponse<null>> {
    return request.delete(`/users/${userId}`);
  },
};

// 导出所有API
export default {
  config: configApi,
  auth: authApi,
  cabinet: cabinetApi,
  device: deviceApi,
  sensor: sensorApi,
  license: licenseApi,
  command: commandApi,
  alert: alertApi,
  audit: auditApi,
  vulnerability: vulnerabilityApi,
  traffic: trafficApi,
  map: mapApi,
  user: userApi,
};
