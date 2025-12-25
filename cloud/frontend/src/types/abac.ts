// ABAC访问控制相关类型定义

// 主体类型
export type SubjectType = 'user' | 'cabinet' | 'device'

// 策略条件操作符
export type ConditionOperator = 'eq' | 'ne' | 'gt' | 'lt' | 'gte' | 'lte' | 'in' | 'contains'

// 策略条件
export interface PolicyCondition {
  attribute: string
  operator: ConditionOperator
  value: any
}

// 访问策略
export interface AccessPolicy {
  id: string
  name: string
  description?: string
  subject_type: SubjectType
  conditions: PolicyCondition[]
  permissions: string[]
  priority: number
  enabled: boolean
  created_at: string
  updated_at: string
}

// 创建策略请求
export interface CreatePolicyRequest {
  id: string
  name: string
  description?: string
  subject_type: SubjectType
  conditions: PolicyCondition[]
  permissions: string[]
  priority: number
}

// 更新策略请求
export interface UpdatePolicyRequest {
  name?: string
  description?: string
  conditions?: PolicyCondition[]
  permissions?: string[]
  priority?: number
  enabled?: boolean
}

// 策略列表筛选
export interface PolicyListFilter {
  page: number
  page_size: number
  subject_type?: SubjectType
  enabled?: boolean
  search?: string
}

// 策略列表响应
export interface PolicyListResponse {
  success: boolean
  data: AccessPolicy[]
  total: number
  page: number
  page_size: number
}

// 访问日志
export interface AccessLog {
  id: number
  subject_type: string
  subject_id: string
  resource: string
  action: string
  allowed: boolean
  policy_id?: string
  trust_score?: number
  ip_address?: string
  timestamp: string
  attributes?: Record<string, any>
}

// 访问日志筛选
export interface AccessLogFilter {
  page: number
  page_size: number
  subject_type?: string
  subject_id?: string
  resource?: string
  allowed?: boolean
  start_time?: string
  end_time?: string
}

// 访问日志列表响应
export interface AccessLogListResponse {
  success: boolean
  data: AccessLog[]
  total: number
  page: number
  page_size: number
}

// 信任度分布
export interface TrustScoreDistribution {
  range_0_30: number
  range_30_60: number
  range_60_80: number
  range_80_100: number
}

// 资源统计
export interface ResourceStat {
  resource: string
  count: number
}

// 拒绝原因统计
export interface DenyReasonStat {
  reason: string
  count: number
}

// 访问统计
export interface AccessStats {
  total_requests: number
  allowed_requests: number
  denied_requests: number
  allow_rate: number
  deny_rate: number
  trust_score_distribution: TrustScoreDistribution
  top_resources: ResourceStat[]
  deny_reasons: DenyReasonStat[]
}

// 策略评估请求
export interface EvaluationRequest {
  subject_type: SubjectType
  attributes: Record<string, any>
  resource: string
  action: string
}

// 策略评估结果
export interface EvaluationResult {
  allowed: boolean
  matched_policy?: AccessPolicy
  trust_score: number
  permissions: string[]
  reason: string
}
