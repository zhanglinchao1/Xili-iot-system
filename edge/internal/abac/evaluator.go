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
	SubjectAttrs Attributes
	Resource     string
	Action       string
	Policies     []*AccessPolicy
}

// EvaluateResponse 评估响应
type EvaluateResponse struct {
	Allowed       bool
	MatchedPolicy *AccessPolicy
	TrustScore    float64
	Permissions   []string
	Reason        string
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
		if !policy.Enabled || string(req.SubjectAttrs.GetType()) != policy.SubjectType {
			continue
		}

		if e.matchConditions(req.SubjectAttrs, policy.Conditions) {
			matchedPolicy = policy
			break
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

	// 通配符权限
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

func (e *Evaluator) matchConditions(attrs Attributes, conditions []PolicyCondition) bool {
	for _, cond := range conditions {
		if !e.matchCondition(attrs, cond) {
			return false
		}
	}
	return true
}

func (e *Evaluator) matchCondition(attrs Attributes, cond PolicyCondition) bool {
	attrValue := e.getAttributeValue(attrs, cond.Attribute)
	if attrValue == nil {
		return false
	}

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

func (e *Evaluator) getAttributeValue(attrs Attributes, attrName string) interface{} {
	if attrName == "trust_score" {
		return attrs.GetTrustScore()
	}

	val := reflect.ValueOf(attrs)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	fieldName := e.snakeToPascal(attrName)
	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}

	return field.Interface()
}

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
	case string:
		// 尝试解析字符串数字
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func (e *Evaluator) snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func (e *Evaluator) makePermission(resource, action string) string {
	operation := "read"
	switch strings.ToUpper(action) {
	case "GET":
		operation = "read"
	case "POST", "PUT", "PATCH":
		operation = "write"
	case "DELETE":
		operation = "delete"
	}

	parts := strings.Split(strings.Trim(resource, "/"), "/")
	if len(parts) > 0 {
		for i := len(parts) - 1; i >= 0; i-- {
			if !e.isID(parts[i]) {
				return operation + ":" + parts[i]
			}
		}
	}

	return operation + ":unknown"
}

func (e *Evaluator) matchPermission(granted, required string) bool {
	if granted == required || granted == "*" {
		return true
	}

	if strings.HasSuffix(granted, ":*") {
		grantedOp := strings.TrimSuffix(granted, ":*")
		requiredOp := strings.Split(required, ":")[0]
		return grantedOp == requiredOp
	}

	return false
}

func (e *Evaluator) isID(s string) bool {
	if len(s) == 0 {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			if strings.Contains(s, "-") {
				return true
			}
			return false
		}
	}
	return true
}
