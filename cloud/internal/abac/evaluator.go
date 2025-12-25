package abac

import (
	"fmt"
	"reflect"
	"strings"
)

// Evaluator 策略评估引擎
type Evaluator struct {
	scorer *TrustScorer
}

// NewEvaluator 创建策略评估引擎
func NewEvaluator() *Evaluator {
	return &Evaluator{
		scorer: NewTrustScorer(),
	}
}

// EvaluateRequest 评估访问请求
type EvaluateRequest struct {
	SubjectAttrs Attributes // 主体属性
	Resource     string     // 请求的资源
	Action       string     // 请求的动作 (GET, POST, PUT, DELETE)
	Policies     []*AccessPolicy // 策略列表
}

// EvaluateResponse 评估响应
type EvaluateResponse struct {
	Allowed       bool          // 是否允许访问
	MatchedPolicy *AccessPolicy // 匹配的策略
	TrustScore    float64       // 信任度分数
	Permissions   []string      // 授予的权限
	Reason        string        // 拒绝原因
}

// Evaluate 执行策略评估
func (e *Evaluator) Evaluate(req *EvaluateRequest) *EvaluateResponse {
	resp := &EvaluateResponse{
		Allowed:     false,
		Permissions: []string{},
	}

	// 1. 计算信任度
	resp.TrustScore = e.scorer.CalculateTrustScore(req.SubjectAttrs)

	// 2. 按优先级筛选并评估策略
	var matchedPolicy *AccessPolicy
	for _, policy := range req.Policies {
		// 只评估启用的策略且主体类型匹配的策略
		if !policy.Enabled || string(req.SubjectAttrs.GetType()) != policy.SubjectType {
			continue
		}

		// 检查策略条件
		if e.matchConditions(req.SubjectAttrs, policy.Conditions) {
			matchedPolicy = policy
			break // 找到第一个匹配的策略就停止 (策略已按优先级排序)
		}
	}

	// 3. 没有匹配的策略
	if matchedPolicy == nil {
		resp.Reason = "无匹配的访问策略"
		return resp
	}

	// 4. 检查权限
	resp.MatchedPolicy = matchedPolicy
	resp.Permissions = matchedPolicy.Permissions

	// 检查是否有通配符权限
	for _, perm := range matchedPolicy.Permissions {
		if perm == "*" {
			resp.Allowed = true
			return resp
		}
	}

	// 检查资源和动作是否在权限列表中
	requiredPerm := e.makePermission(req.Resource, req.Action)
	for _, perm := range matchedPolicy.Permissions {
		if e.matchPermission(perm, requiredPerm) {
			resp.Allowed = true
			return resp
		}
	}

	resp.Reason = fmt.Sprintf("权限不足，需要权限: %s", requiredPerm)
	return resp
}

// matchConditions 匹配策略条件
func (e *Evaluator) matchConditions(attrs Attributes, conditions []PolicyCondition) bool {
	// 所有条件都必须满足 (AND逻辑)
	for _, cond := range conditions {
		if !e.matchCondition(attrs, cond) {
			return false
		}
	}
	return true
}

// matchCondition 匹配单个条件
func (e *Evaluator) matchCondition(attrs Attributes, cond PolicyCondition) bool {
	// 获取属性值
	attrValue := e.getAttributeValue(attrs, cond.Attribute)
	if attrValue == nil {
		return false
	}

	// 根据操作符进行比较
	switch cond.Operator {
	case "eq":
		return e.equals(attrValue, cond.Value)
	case "ne":
		return !e.equals(attrValue, cond.Value)
	case "gt":
		return e.greaterThan(attrValue, cond.Value)
	case "lt":
		return e.lessThan(attrValue, cond.Value)
	case "gte":
		return e.greaterThanOrEqual(attrValue, cond.Value)
	case "lte":
		return e.lessThanOrEqual(attrValue, cond.Value)
	case "in":
		return e.in(attrValue, cond.Value)
	case "contains":
		return e.contains(attrValue, cond.Value)
	default:
		return false
	}
}

// getAttributeValue 获取属性值 (支持嵌套属性)
func (e *Evaluator) getAttributeValue(attrs Attributes, attrName string) interface{} {
	// 特殊属性：trust_score
	if attrName == "trust_score" {
		return attrs.GetTrustScore()
	}

	// 使用反射获取结构体字段值
	val := reflect.ValueOf(attrs)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 将snake_case转换为PascalCase
	fieldName := e.snakeToPascal(attrName)
	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}

	return field.Interface()
}

// 比较操作符实现

func (e *Evaluator) equals(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func (e *Evaluator) greaterThan(a, b interface{}) bool {
	av, aok := e.toFloat64(a)
	bv, bok := e.toFloat64(b)
	return aok && bok && av > bv
}

func (e *Evaluator) lessThan(a, b interface{}) bool {
	av, aok := e.toFloat64(a)
	bv, bok := e.toFloat64(b)
	return aok && bok && av < bv
}

func (e *Evaluator) greaterThanOrEqual(a, b interface{}) bool {
	av, aok := e.toFloat64(a)
	bv, bok := e.toFloat64(b)
	return aok && bok && av >= bv
}

func (e *Evaluator) lessThanOrEqual(a, b interface{}) bool {
	av, aok := e.toFloat64(a)
	bv, bok := e.toFloat64(b)
	return aok && bok && av <= bv
}

func (e *Evaluator) in(a, b interface{}) bool {
	// b应该是一个数组
	bVal := reflect.ValueOf(b)
	if bVal.Kind() != reflect.Slice && bVal.Kind() != reflect.Array {
		return false
	}

	for i := 0; i < bVal.Len(); i++ {
		if e.equals(a, bVal.Index(i).Interface()) {
			return true
		}
	}
	return false
}

func (e *Evaluator) contains(a, b interface{}) bool {
	aStr, ok := a.(string)
	if !ok {
		return false
	}
	bStr, ok := b.(string)
	if !ok {
		return false
	}
	return strings.Contains(aStr, bStr)
}

// toFloat64 尝试将值转换为float64
func (e *Evaluator) toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

// snakeToPascal 将snake_case转换为PascalCase
func (e *Evaluator) snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// makePermission 生成权限字符串
func (e *Evaluator) makePermission(resource, action string) string {
	// 从resource提取操作类型
	// 例如: /api/v1/cabinets -> read:cabinets
	// POST /api/v1/cabinets/xxx/sync -> write:sensor_data

	// 根据HTTP方法映射到操作
	operation := "read"
	switch strings.ToUpper(action) {
	case "GET":
		operation = "read"
	case "POST", "PUT", "PATCH":
		operation = "write"
	case "DELETE":
		operation = "delete"
	}

	// 从路径中提取资源名称
	parts := strings.Split(strings.Trim(resource, "/"), "/")
	if len(parts) > 0 {
		// 特殊处理：某些子资源应该映射到父资源权限
		// 例如: /api/v1/cabinets/locations -> read:cabinets
		//      /api/v1/cabinets/statistics -> read:cabinets
		//      /api/v1/cabinets/:id/vulnerability/* -> read:cabinets
		subResourceMappings := map[string]string{
			"locations":     "cabinets", // 储能柜位置信息属于储能柜资源
			"statistics":    "cabinets", // 储能柜统计信息属于储能柜资源
			"vulnerability": "cabinets", // 脆弱性评估属于储能柜资源
			"traffic":       "cabinets", // 流量监测属于储能柜相关数据
		}

		// 获取最后一个非ID的部分
		for i := len(parts) - 1; i >= 0; i-- {
			if !e.isID(parts[i]) {
				resourceName := parts[i]
				// 检查是否是应该映射到父资源的子资源
				if mappedResource, ok := subResourceMappings[resourceName]; ok {
					// 检查父资源是否存在（即前一个部分）
					if i > 0 && parts[i-1] == mappedResource {
						return operation + ":" + mappedResource
					}
					// 特殊处理：/api/v1/cabinets/:id/vulnerability/* 的情况
					// 需要检查路径中是否包含cabinets
					for j := i - 1; j >= 0; j-- {
						if parts[j] == mappedResource {
							return operation + ":" + mappedResource
						}
					}
					// 特殊处理：对于traffic等独立资源，直接映射到父资源
					// 因为流量监测本质上是储能柜相关的数据
					if resourceName == "traffic" {
						return operation + ":" + mappedResource
					}
				}
				return operation + ":" + resourceName
			}
		}
	}

	return operation + ":unknown"
}

// matchPermission 匹配权限
func (e *Evaluator) matchPermission(granted, required string) bool {
	// 精确匹配
	if granted == required {
		return true
	}

	// 通配符匹配
	if granted == "*" {
		return true
	}

	// 操作通配符: read:* 匹配所有读操作
	if strings.HasSuffix(granted, ":*") {
		grantedOp := strings.TrimSuffix(granted, ":*")
		requiredOp := strings.Split(required, ":")[0]
		return grantedOp == requiredOp
	}

	return false
}

// isID 判断是否为ID (简单判断：纯数字或UUID格式)
func (e *Evaluator) isID(s string) bool {
	// 简单判断：如果全是数字或包含-（UUID），认为是ID
	if len(s) == 0 {
		return false
	}

	// 全是数字
	for _, c := range s {
		if c < '0' || c > '9' {
			// 检查是否包含-（UUID）
			if strings.Contains(s, "-") {
				return true
			}
			return false
		}
	}
	return true
}
