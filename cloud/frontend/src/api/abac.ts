// ABAC访问控制API封装
import type {
  AccessPolicy,
  CreatePolicyRequest,
  UpdatePolicyRequest,
  PolicyListFilter,
  PolicyListResponse,
  AccessLogFilter,
  AccessLogListResponse,
  AccessStats,
  EvaluationRequest,
  EvaluationResult
} from '@/types/abac'
import { request } from '@/utils/request'

// ==================== 策略管理API ====================

/**
 * 获取策略列表
 */
export function listPolicies(params: PolicyListFilter) {
  return request.get<PolicyListResponse>('/abac/policies', { params })
}

/**
 * 获取策略详情
 */
export function getPolicy(id: string) {
  return request.get<{ policy: AccessPolicy }>(`/abac/policies/${id}`)
}

/**
 * 创建策略
 */
export function createPolicy(data: CreatePolicyRequest) {
  return request.post<{ policy: AccessPolicy }>('/abac/policies', data)
}

/**
 * 更新策略
 */
export function updatePolicy(id: string, data: UpdatePolicyRequest) {
  return request.put<{ message: string }>(`/abac/policies/${id}`, data)
}

/**
 * 删除策略
 */
export function deletePolicy(id: string) {
  return request.delete<{ message: string }>(`/abac/policies/${id}`)
}

/**
 * 切换策略启用状态
 */
export function togglePolicy(id: string) {
  return request.post<{ message: string }>(`/abac/policies/${id}/toggle`)
}

// ==================== 访问日志API ====================

/**
 * 获取访问日志列表
 */
export function listAccessLogs(params: AccessLogFilter) {
  return request.get<AccessLogListResponse>('/abac/access-logs', { params })
}

/**
 * 获取访问统计
 */
export function getAccessStats(params?: { start_time?: string; end_time?: string }) {
  return request.get<{ stats: AccessStats }>('/abac/access-stats', { params })
}

// ==================== 策略评估API ====================

/**
 * 测试策略评估
 */
export function evaluatePolicy(data: EvaluationRequest) {
  return request.post<{ result: EvaluationResult }>('/abac/evaluate', data)
}
