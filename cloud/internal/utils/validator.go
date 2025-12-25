package utils

import (
	"fmt"
	"regexp"
	"strings"

	"cloud-system/pkg/errors"
)

// 验证规则正则表达式
var (
	// CabinetID格式: 大写字母、数字、连字符和下划线
	CabinetIDRegex = regexp.MustCompile(`^[A-Z0-9_-]+$`)
	// MAC地址格式
	MACAddressRegex = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
	// Email格式
	EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateCabinetID 验证储能柜ID格式
func ValidateCabinetID(cabinetID string) error {
	if cabinetID == "" {
		return errors.NewValidationError("cabinet_id不能为空")
	}
	if len(cabinetID) > 50 {
		return errors.NewValidationError("cabinet_id长度不能超过50个字符")
	}
	if !CabinetIDRegex.MatchString(cabinetID) {
		return errors.NewValidationError("cabinet_id格式无效，只能包含大写字母、数字、连字符和下划线")
	}
	return nil
}

// ValidateMACAddress 验证MAC地址格式
func ValidateMACAddress(macAddress string) error {
	if macAddress == "" {
		return errors.NewValidationError("mac_address不能为空")
	}
	if !MACAddressRegex.MatchString(macAddress) {
		return errors.NewValidationError("mac_address格式无效，应为XX:XX:XX:XX:XX:XX格式")
	}
	return nil
}

// ValidateEmail 验证Email格式
func ValidateEmail(email string) error {
	if email == "" {
		return nil // Email可选
	}
	if !EmailRegex.MatchString(email) {
		return errors.NewValidationError("email格式无效")
	}
	return nil
}

// ValidateRequired 验证必填字段
func ValidateRequired(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.NewValidationError(fmt.Sprintf("%s不能为空", fieldName))
	}
	return nil
}

// ValidateLength 验证字符串长度
func ValidateLength(fieldName, value string, min, max int) error {
	length := len(value)
	if length < min {
		return errors.NewValidationError(
			fmt.Sprintf("%s长度不能少于%d个字符", fieldName, min),
		)
	}
	if max > 0 && length > max {
		return errors.NewValidationError(
			fmt.Sprintf("%s长度不能超过%d个字符", fieldName, max),
		)
	}
	return nil
}

// ValidateRange 验证数值范围
func ValidateRange(fieldName string, value, min, max float64) error {
	if value < min {
		return errors.NewValidationError(
			fmt.Sprintf("%s不能小于%f", fieldName, min),
		)
	}
	if value > max {
		return errors.NewValidationError(
			fmt.Sprintf("%s不能大于%f", fieldName, max),
		)
	}
	return nil
}

// ValidateEnum 验证枚举值
func ValidateEnum(fieldName, value string, validValues []string) error {
	for _, valid := range validValues {
		if value == valid {
			return nil
		}
	}
	return errors.NewValidationError(
		fmt.Sprintf("%s必须是以下值之一: %s", fieldName, strings.Join(validValues, ", ")),
	)
}

// ValidatePageSize 验证分页大小
func ValidatePageSize(pageSize, maxPageSize int) (int, error) {
	if pageSize <= 0 {
		return 20, nil // 默认值
	}
	if pageSize > maxPageSize {
		return 0, errors.NewValidationError(
			fmt.Sprintf("page_size不能超过%d", maxPageSize),
		)
	}
	return pageSize, nil
}

// ValidatePage 验证页码
func ValidatePage(page int) (int, error) {
	if page < 1 {
		return 1, nil // 默认第一页
	}
	return page, nil
}

// ValidateSensorType 验证传感器类型
func ValidateSensorType(sensorType string) error {
	validTypes := []string{
		"co2", "co", "smoke", "liquid_level",
		"conductivity", "temperature", "flow",
	}
	return ValidateEnum("sensor_type", sensorType, validTypes)
}

// ValidateStatus 验证状态
func ValidateStatus(status string, validStatuses []string) error {
	return ValidateEnum("status", status, validStatuses)
}

// ValidateSeverity 验证告警严重度
func ValidateSeverity(severity string) error {
	validSeverities := []string{"info", "warning", "error", "critical"}
	return ValidateEnum("severity", severity, validSeverities)
}
