/*
 * API处理函数
 * 实现各种API端点的处理逻辑
 */
package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/edge/storage-cabinet/internal/auth"
	"github.com/edge/storage-cabinet/internal/collector"
	"github.com/edge/storage-cabinet/internal/device"
	"github.com/edge/storage-cabinet/internal/storage"
	"github.com/edge/storage-cabinet/internal/vulnerability"
	"github.com/edge/storage-cabinet/pkg/models"
	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "edge-system",
	})
}

// ReadyCheck 就绪检查
func ReadyCheck(c *gin.Context) {
	// TODO: 检查各个服务的就绪状态
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"checks": gin.H{
			"database": "ok",
			"zkp":      "ok",
			"services": "ok",
		},
	})
}

// GetChallenge 获取认证挑战
func GetChallenge(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.ChallengeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 验证设备ID格式
		if len(req.DeviceID) == 0 || len(req.DeviceID) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_DEVICE_ID",
				"message": "设备ID长度必须在1-64字符之间",
			})
			return
		}

		challenge, err := authService.GenerateChallenge(req.DeviceID)
		if err != nil {
			// 检查是否是许可证错误
			if strings.Contains(err.Error(), "LICENSE_001") {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "LICENSE_001",
					"message": "许可证校验失败: " + err.Error(),
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "CHALLENGE_FAILED",
				"message": "生成挑战失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, models.ChallengeResponse{
			ChallengeID: challenge.ChallengeID,
			Nonce:       challenge.Nonce,
			ExpiresAt:   challenge.ExpiresAt,
		})
	}
}

// VerifyProof 验证零知识证明
func VerifyProof(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.AuthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		session, err := authService.VerifyProof(&req)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "AUTH_FAILED",
				"message": "认证失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, models.AuthResponse{
			Success:   true,
			SessionID: session.SessionID,
			Token:     session.Token,
			ExpiresAt: session.ExpiresAt,
			Message:   "认证成功",
		})
	}
}

// RefreshSession 刷新会话
func RefreshSession(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Authorization头获取当前token
		token := extractTokenFromHeader(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "缺少认证令牌",
			})
			return
		}

		newSession, err := authService.RefreshSession(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "REFRESH_FAILED",
				"message": "刷新会话失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, models.AuthResponse{
			Success:   true,
			SessionID: newSession.SessionID,
			Token:     newSession.Token,
			ExpiresAt: newSession.ExpiresAt,
			Message:   "会话刷新成功",
		})
	}
}

// GetDeviceStatistics 获取设备统计信息
func GetDeviceStatistics(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := deviceManager.GetStatistics()
		c.JSON(http.StatusOK, stats)
	}
}

// GetDeviceLatestData 获取设备最新传感器数据
func GetDeviceLatestData(dataCollector *collector.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("id")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "设备ID不能为空",
			})
			return
		}

		// 查询最新数据（限制1条）
		data, _, err := dataCollector.QueryData(deviceID, "", "", "", 1, 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询设备数据失败: " + err.Error(),
			})
			return
		}

		if len(data) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"device_id": deviceID,
				"value":     nil,
				"unit":      "",
				"timestamp": nil,
				"message":   "暂无数据",
			})
			return
		}

		latestData := data[0]
		c.JSON(http.StatusOK, gin.H{
			"device_id": latestData.DeviceID,
			"value":     latestData.Value,
			"unit":      latestData.Unit,
			"timestamp": latestData.Timestamp,
			"quality":   latestData.Quality,
		})
	}
}

// GetDevicesByCabinet 按储能柜获取设备列表
// Edge端是单柜系统，返回所有设备
func GetDevicesByCabinet(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		cabinetID := c.Param("cabinet_id")
		if cabinetID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "储能柜ID不能为空",
			})
			return
		}

		// Edge端是单柜系统，返回所有设备
		devices, err := deviceManager.GetAllDevices()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询设备失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"cabinet_id": cabinetID,
			"devices":    devices,
			"total":      len(devices),
		})
	}
}

// GetCabinetList 获取储能柜列表
func GetCabinetList(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		devices, err := deviceManager.GetAllDevices()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询设备列表失败: " + err.Error(),
			})
			return
		}

		// 按储能柜分组统计
		cabinetStats := make(map[string]map[string]interface{})
		for _, device := range devices {
			cabinetID := device.CabinetID
			if _, exists := cabinetStats[cabinetID]; !exists {
				cabinetStats[cabinetID] = map[string]interface{}{
					"cabinet_id":    cabinetID,
					"device_count":  0,
					"online_count":  0,
					"offline_count": 0,
					"sensor_types":  make(map[string]int),
				}
			}

			stats := cabinetStats[cabinetID]
			stats["device_count"] = stats["device_count"].(int) + 1

			if device.Status == "online" {
				stats["online_count"] = stats["online_count"].(int) + 1
			} else {
				stats["offline_count"] = stats["offline_count"].(int) + 1
			}

			sensorTypes := stats["sensor_types"].(map[string]int)
			sensorTypes[string(device.SensorType)]++
		}

		// 转换为数组格式
		cabinets := make([]map[string]interface{}, 0, len(cabinetStats))
		for _, stats := range cabinetStats {
			cabinets = append(cabinets, stats)
		}

		c.JSON(http.StatusOK, gin.H{
			"cabinets": cabinets,
			"total":    len(cabinets),
		})
	}
}

// ListDevices 获取设备列表
func ListDevices(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析查询参数
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		status := c.Query("status")
		sensorType := c.Query("sensor_type")

		devices, total, err := deviceManager.ListDevices(page, limit, status, sensorType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询设备列表失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"devices": devices,
			"total":   total,
			"page":    page,
			"limit":   limit,
		})
	}
}

// GetDevice 获取设备详情
func GetDevice(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("id")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "设备ID不能为空",
			})
			return
		}

		device, err := deviceManager.GetDevice(deviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "DEVICE_NOT_FOUND",
				"message": "设备不存在: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, device)
	}
}

// RegisterDevice 注册设备
func RegisterDevice(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.DeviceRegistration
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 额外的输入验证
		if err := validateDeviceRegistration(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_DATA",
				"message": "设备注册信息验证失败: " + err.Error(),
			})
			return
		}

		device, err := deviceManager.RegisterDevice(&req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "REGISTER_FAILED",
				"message": "设备注册失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, device)
	}
}

// UpdateDevice 更新设备信息
func UpdateDevice(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("id")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "设备ID不能为空",
			})
			return
		}

		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		device, err := deviceManager.UpdateDevice(deviceID, updates)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "UPDATE_FAILED",
				"message": "更新设备失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, device)
	}
}

// UnregisterDevice 注销设备
func UnregisterDevice(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("id")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "设备ID不能为空",
			})
			return
		}

		err := deviceManager.UnregisterDevice(deviceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "UNREGISTER_FAILED",
				"message": "注销设备失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "设备注销成功",
		})
	}
}

// DeviceHeartbeat 设备心跳
func DeviceHeartbeat(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Param("id")
		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "设备ID不能为空",
			})
			return
		}

		var heartbeat models.Heartbeat
		if err := c.ShouldBindJSON(&heartbeat); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 确保设备ID一致
		heartbeat.DeviceID = deviceID

		err := deviceManager.ProcessHeartbeat(&heartbeat)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "HEARTBEAT_FAILED",
				"message": "处理心跳失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "心跳处理成功",
		})
	}
}

// CollectData 数据采集
func CollectData(dataCollector *collector.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.DataCollectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 验证数据范围
		if err := validateSensorData(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_DATA",
				"message": "传感器数据验证失败: " + err.Error(),
			})
			return
		}

		// 设置时间戳
		if req.Timestamp.IsZero() {
			req.Timestamp = time.Now()
		}

		err := dataCollector.CollectData(&req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "COLLECT_FAILED",
				"message": "数据采集失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "数据采集成功",
		})
	}
}

// QueryData 查询历史数据
func QueryData(dataCollector *collector.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Query("device_id")
		sensorType := c.Query("sensor_type")
		startTime := c.Query("start_time")
		endTime := c.Query("end_time")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100000")) // 提高到100000，确保能获取所有历史数据

		data, total, err := dataCollector.QueryData(deviceID, sensorType, startTime, endTime, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询数据失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  data,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

// GetStatistics 获取数据统计
func GetStatistics(dataCollector *collector.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.Query("device_id")
		sensorType := c.Query("sensor_type")
		period := c.DefaultQuery("period", "24h") // 1h, 24h, 7d, 30d

		stats, err := dataCollector.GetStatisticsByPeriod(deviceID, sensorType, period)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "STATS_FAILED",
				"message": "获取统计数据失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, stats)
	}
}

// ListAlerts 获取告警列表
func ListAlerts(dataCollector *collector.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		severity := c.Query("severity")
		resolved := c.Query("resolved")

		alerts, total, err := dataCollector.ListAlerts(page, limit, severity, resolved)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询告警失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"alerts": alerts,
			"total":  total,
			"page":   page,
			"limit":  limit,
		})
	}
}

// CreateAlert 创建告警
func CreateAlert(dataCollector *collector.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var alert models.Alert
		if err := c.ShouldBindJSON(&alert); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 设置时间戳
		alert.Timestamp = time.Now()

		err := dataCollector.CreateAlert(&alert)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "CREATE_FAILED",
				"message": "创建告警失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, alert)
	}
}

// ResolveAlert 解决告警
func ResolveAlert(dataCollector *collector.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		alertIDStr := c.Param("id")
		alertID, err := strconv.ParseInt(alertIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "无效的告警ID",
			})
			return
		}

		err = dataCollector.ResolveAlert(alertID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "RESOLVE_FAILED",
				"message": "解决告警失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "告警已解决",
		})
	}
}

// extractTokenFromHeader 从Authorization头提取token
func extractTokenFromHeader(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// validateSensorData 验证传感器数据
func validateSensorData(req *models.DataCollectRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}
	// 验证设备ID
	if len(req.DeviceID) == 0 || len(req.DeviceID) > 64 {
		return fmt.Errorf("设备ID长度必须在1-64字符之间")
	}

	// 验证传感器类型
	validTypes := map[models.SensorType]bool{
		models.SensorCO2:          true,
		models.SensorCO:           true,
		models.SensorSmoke:        true,
		models.SensorLiquidLevel:  true,
		models.SensorConductivity: true,
		models.SensorTemperature:  true,
		models.SensorFlow:         true,
	}
	if !validTypes[req.SensorType] {
		return fmt.Errorf("不支持的传感器类型: %s", req.SensorType)
	}

	// 验证数值范围
	switch req.SensorType {
	case models.SensorCO2:
		if req.Value < 0 || req.Value > 50000 {
			return fmt.Errorf("CO2浓度值超出范围(0-50000ppm): %.2f", req.Value)
		}
	case models.SensorCO:
		if req.Value < 0 || req.Value > 1000 {
			return fmt.Errorf("CO浓度值超出范围(0-1000ppm): %.2f", req.Value)
		}
	case models.SensorSmoke:
		if req.Value < 0 || req.Value > 10000 {
			return fmt.Errorf("烟雾浓度值超出范围(0-10000ppm): %.2f", req.Value)
		}
	case models.SensorLiquidLevel:
		if req.Value < 0 || req.Value > 2000 {
			return fmt.Errorf("液位值超出范围(0-2000mm): %.2f", req.Value)
		}
	case models.SensorConductivity:
		if req.Value < 0 || req.Value > 100 {
			return fmt.Errorf("电导率值超出范围(0-100mS/cm): %.2f", req.Value)
		}
	case models.SensorTemperature:
		if req.Value < -50 || req.Value > 150 {
			return fmt.Errorf("温度值超出范围(-50-150°C): %.2f", req.Value)
		}
	case models.SensorFlow:
		if req.Value < 0 || req.Value > 1000 {
			return fmt.Errorf("流速值超出范围(0-1000L/min): %.2f", req.Value)
		}
	}

	// 验证数据质量
	if req.Quality < 0 || req.Quality > 100 {
		return fmt.Errorf("数据质量值超出范围(0-100): %d", req.Quality)
	}

	return nil
}

// validateDeviceRegistration 验证设备注册信息
func validateDeviceRegistration(req *models.DeviceRegistration) error {
	if req == nil {
		return fmt.Errorf("设备注册信息不能为空")
	}

	// 验证设备ID
	if len(req.DeviceID) == 0 || len(req.DeviceID) > 64 {
		return fmt.Errorf("设备ID长度必须在1-64字符之间")
	}

	// 验证设备类型
	if req.DeviceType == "" {
		return fmt.Errorf("设备类型不能为空")
	}

	// 验证传感器类型
	validTypes := map[models.SensorType]bool{
		models.SensorCO2:          true,
		models.SensorCO:           true,
		models.SensorSmoke:        true,
		models.SensorLiquidLevel:  true,
		models.SensorConductivity: true,
		models.SensorTemperature:  true,
		models.SensorFlow:         true,
	}
	if !validTypes[req.SensorType] {
		return fmt.Errorf("不支持的传感器类型: %s", req.SensorType)
	}

	// 验证储能柜ID
	if len(req.CabinetID) == 0 || len(req.CabinetID) > 32 {
		return fmt.Errorf("储能柜ID长度必须在1-32字符之间")
	}

	// 验证公钥格式
	if len(req.PublicKey) == 0 {
		return fmt.Errorf("公钥不能为空")
	}

	// 验证承诺值格式
	if len(req.Commitment) == 0 {
		return fmt.Errorf("承诺值不能为空")
	}

	return nil
}

// GetAlertLogs 获取告警日志
func GetAlertLogs(db *storage.SQLiteDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取底层sql.DB
		sqlDB := db.GetDB()

		// 获取查询参数
		startDate := c.DefaultQuery("start_date", "")
		endDate := c.DefaultQuery("end_date", "")
		severity := c.DefaultQuery("severity", "")
		resolved := c.DefaultQuery("resolved", "")
		deviceID := c.DefaultQuery("device_id", "")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}

		offset := (page - 1) * limit

		// 构建查询条件
		where := []string{"1=1"}
		args := []interface{}{}

		if startDate != "" {
			where = append(where, "timestamp >= ?")
			args = append(args, startDate+" 00:00:00")
		}
		if endDate != "" {
			where = append(where, "timestamp <= ?")
			args = append(args, endDate+" 23:59:59")
		}
		if severity != "" {
			where = append(where, "severity = ?")
			args = append(args, severity)
		}
		if resolved != "" {
			if resolved == "true" {
				where = append(where, "resolved = 1")
			} else if resolved == "false" {
				where = append(where, "resolved = 0")
			}
		}
		if deviceID != "" {
			where = append(where, "device_id LIKE ?")
			args = append(args, "%"+deviceID+"%")
		}

		whereClause := strings.Join(where, " AND ")

		// 查询总数
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM alerts WHERE %s", whereClause)
		var total int
		if err := sqlDB.QueryRow(countQuery, args...).Scan(&total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询告警日志总数失败: " + err.Error(),
			})
			return
		}

		// 查询日志列表
		query := fmt.Sprintf(`
			SELECT id, device_id, alert_type, severity, message, value, threshold,
			       timestamp, resolved, resolved_at
			FROM alerts
			WHERE %s
			ORDER BY timestamp DESC
			LIMIT ? OFFSET ?
		`, whereClause)

		args = append(args, limit, offset)
		rows, err := sqlDB.Query(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询告警日志失败: " + err.Error(),
			})
			return
		}
		defer rows.Close()

		logs := []gin.H{}
		for rows.Next() {
			var id int
			var deviceID, alertType, severity, message string
			var value, threshold float64
			var timestamp string
			var resolved bool
			var resolvedAt interface{}

			if err := rows.Scan(&id, &deviceID, &alertType, &severity, &message,
				&value, &threshold, &timestamp, &resolved, &resolvedAt); err != nil {
				continue
			}

			log := gin.H{
				"id":          id,
				"device_id":   deviceID,
				"alert_type":  alertType,
				"severity":    severity,
				"message":     message,
				"value":       value,
				"threshold":   threshold,
				"timestamp":   timestamp,
				"resolved":    resolved,
				"resolved_at": resolvedAt,
			}
			logs = append(logs, log)
		}

		c.JSON(http.StatusOK, gin.H{
			"logs":  logs,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

// GetAuthLogs 获取认证日志
func GetAuthLogs(db *storage.SQLiteDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取底层sql.DB
		sqlDB := db.GetDB()

		// 获取查询参数
		startDate := c.DefaultQuery("start_date", "")
		endDate := c.DefaultQuery("end_date", "")
		status := c.DefaultQuery("status", "")
		deviceID := c.DefaultQuery("device_id", "")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}

		offset := (page - 1) * limit

		// 构建查询条件
		where := []string{"1=1"}
		args := []interface{}{}

		if startDate != "" {
			where = append(where, "created_at >= ?")
			args = append(args, startDate+" 00:00:00")
		}
		if endDate != "" {
			where = append(where, "created_at <= ?")
			args = append(args, endDate+" 23:59:59")
		}
		if deviceID != "" {
			where = append(where, "device_id LIKE ?")
			args = append(args, "%"+deviceID+"%")
		}

		whereClause := strings.Join(where, " AND ")

		// 合并查询挑战和会话记录
		// 策略：挑战表示"请求认证"，会话表示"认证成功"
		var logs []gin.H
		var total int

		// 1. 查询挑战记录（认证请求）
		challengeQuery := fmt.Sprintf(`
			SELECT challenge_id, device_id, created_at, used
			FROM challenges
			WHERE %s
			ORDER BY created_at DESC
		`, whereClause)

		challengeRows, err := sqlDB.Query(challengeQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询认证日志失败: " + err.Error(),
			})
			return
		}
		defer challengeRows.Close()

		for challengeRows.Next() {
			var challengeID, deviceID, createdAt string
			var used bool
			if err := challengeRows.Scan(&challengeID, &deviceID, &createdAt, &used); err != nil {
				continue
			}

			action := "challenge_requested"
			authStatus := "pending"
			details := "生成认证挑战"

			if used {
				action = "challenge_used"
				authStatus = "success"
				details = "挑战已使用（认证成功）"
			}

			// 应用状态筛选
			if status != "" && status != authStatus {
				continue
			}

			logs = append(logs, gin.H{
				"id":         challengeID,
				"device_id":  deviceID,
				"action":     action,
				"status":     authStatus,
				"timestamp":  createdAt,
				"session_id": nil,
				"details":    details,
			})
		}

		// 2. 查询会话记录（认证成功）
		sessionQuery := fmt.Sprintf(`
			SELECT session_id, device_id, created_at, ip_address
			FROM sessions
			WHERE %s
			ORDER BY created_at DESC
		`, whereClause)

		sessionRows, err := sqlDB.Query(sessionQuery, args...)
		if err == nil {
			defer sessionRows.Close()

			for sessionRows.Next() {
				var sessionID, deviceID, createdAt string
				var ipAddress interface{}
				if err := sessionRows.Scan(&sessionID, &deviceID, &createdAt, &ipAddress); err != nil {
					continue
				}

				// 应用状态筛选
				if status != "" && status != "success" {
					continue
				}

				ipStr := ""
				if ipAddress != nil {
					ipStr = ipAddress.(string)
				}

				logs = append(logs, gin.H{
					"id":         sessionID,
					"device_id":  deviceID,
					"action":     "session_created",
					"status":     "success",
					"timestamp":  createdAt,
					"session_id": sessionID,
					"ip_address": ipStr,
					"details":    "会话创建成功",
				})
			}
		}

		// 排序（按时间倒序）
		// 注意：这里简化处理，实际应该在SQL层面合并
		total = len(logs)

		// 分页处理
		start := offset
		end := offset + limit
		if start > total {
			logs = []gin.H{}
		} else {
			if end > total {
				end = total
			}
			logs = logs[start:end]
		}

		c.JSON(http.StatusOK, gin.H{
			"logs":  logs,
			"total": total,
			"page":  page,
			"limit": limit,
		})
	}
}

// BatchDeleteAlertLogs 批量删除告警日志
func BatchDeleteAlertLogs(db *storage.SQLiteDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取底层sql.DB
		sqlDB := db.GetDB()

		// 解析请求体
		var req struct {
			IDs []int `json:"ids" binding:"required,min=1"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 构建批量删除SQL
		if len(req.IDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "至少需要选择一条记录",
			})
			return
		}

		// 构建占位符
		placeholders := make([]string, len(req.IDs))
		args := make([]interface{}, len(req.IDs))
		for i, id := range req.IDs {
			placeholders[i] = "?"
			args[i] = id
		}

		query := fmt.Sprintf("DELETE FROM alerts WHERE id IN (%s)", strings.Join(placeholders, ","))

		result, err := sqlDB.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DELETE_FAILED",
				"message": "删除告警日志失败: " + err.Error(),
			})
			return
		}

		affected, _ := result.RowsAffected()

		c.JSON(http.StatusOK, gin.H{
			"message": "批量删除成功",
			"deleted": affected,
		})
	}
}

// BatchDeleteAuthLogs 批量删除认证日志
func BatchDeleteAuthLogs(db *storage.SQLiteDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取底层sql.DB
		sqlDB := db.GetDB()

		// 解析请求体
		var req struct {
			IDs []string `json:"ids" binding:"required,min=1"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 构建批量删除SQL
		if len(req.IDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "至少需要选择一条记录",
			})
			return
		}

		// 开始事务
		tx, err := sqlDB.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "TRANSACTION_FAILED",
				"message": "开始事务失败: " + err.Error(),
			})
			return
		}
		defer tx.Rollback()

		// 构建占位符
		placeholders := make([]string, len(req.IDs))
		args := make([]interface{}, len(req.IDs))
		for i, id := range req.IDs {
			placeholders[i] = "?"
			args[i] = id
		}

		var totalAffected int64

		// 从 challenges 表删除（认证日志实际存储在 challenges 和 sessions 表中）
		challengeQuery := fmt.Sprintf("DELETE FROM challenges WHERE challenge_id IN (%s)", strings.Join(placeholders, ","))
		result1, err := tx.Exec(challengeQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DELETE_FAILED",
				"message": "删除认证日志失败: " + err.Error(),
			})
			return
		}
		affected1, _ := result1.RowsAffected()
		totalAffected += affected1

		// 从 sessions 表删除
		sessionQuery := fmt.Sprintf("DELETE FROM sessions WHERE session_id IN (%s)", strings.Join(placeholders, ","))
		result2, err := tx.Exec(sessionQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DELETE_FAILED",
				"message": "删除认证日志失败: " + err.Error(),
			})
			return
		}
		affected2, _ := result2.RowsAffected()
		totalAffected += affected2

		// 提交事务
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "COMMIT_FAILED",
				"message": "提交事务失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "批量删除成功",
			"deleted": totalAffected,
		})
	}
}

// ClearAllAuthLogs 一键清空所有认证日志
func ClearAllAuthLogs(db *storage.SQLiteDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取底层sql.DB
		sqlDB := db.GetDB()

		// 开始事务
		tx, err := sqlDB.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "TRANSACTION_FAILED",
				"message": "开始事务失败: " + err.Error(),
			})
			return
		}
		defer tx.Rollback()

		// 统计要删除的记录数
		var challengeCount, sessionCount int64

		// 统计 challenges 表记录数
		err = tx.QueryRow("SELECT COUNT(*) FROM challenges").Scan(&challengeCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "COUNT_FAILED",
				"message": "统计记录失败: " + err.Error(),
			})
			return
		}

		// 统计 sessions 表记录数
		err = tx.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "COUNT_FAILED",
				"message": "统计记录失败: " + err.Error(),
			})
			return
		}

		totalCount := challengeCount + sessionCount

		// 如果没有记录，直接返回
		if totalCount == 0 {
			c.JSON(http.StatusOK, gin.H{
				"message": "没有需要清空的日志",
				"deleted": 0,
			})
			return
		}

		// 清空 challenges 表
		_, err = tx.Exec("DELETE FROM challenges")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DELETE_FAILED",
				"message": "清空认证日志失败: " + err.Error(),
			})
			return
		}

		// 清空 sessions 表
		_, err = tx.Exec("DELETE FROM sessions")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DELETE_FAILED",
				"message": "清空认证日志失败: " + err.Error(),
			})
			return
		}

		// 重置自增ID（SQLite使用sqlite_sequence表）
		_, _ = tx.Exec("DELETE FROM sqlite_sequence WHERE name='challenges'")
		_, _ = tx.Exec("DELETE FROM sqlite_sequence WHERE name='sessions'")

		// 提交事务
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "COMMIT_FAILED",
				"message": "提交事务失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "成功清空所有认证日志",
			"deleted": totalCount,
		})
	}
}

// GetLicenseInfo 获取许可证信息
func GetLicenseInfo(licenseService interface {
	IsEnabled() bool
	GetLicenseInfo() map[string]interface{}
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		info := licenseService.GetLicenseInfo()
		c.JSON(http.StatusOK, info)
	}
}

// GetAlertConfig 获取告警配置(包括阈值)
func GetAlertConfig(cfg interface {
	GetSensorThreshold(string) (float64, float64, bool)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取所有传感器类型的阈值
		sensorTypes := []string{"co2", "co", "smoke", "liquid_level", "conductivity", "temperature", "flow"}
		thresholds := make(map[string]interface{})

		for _, sensorType := range sensorTypes {
			min, max, enabled := cfg.GetSensorThreshold(sensorType)
			if enabled {
				thresholds[sensorType] = gin.H{
					"min": min,
					"max": max,
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"enabled":    len(thresholds) > 0,
			"thresholds": thresholds,
		})
	}
}

// CloudCredentialsStore 定义Cloud凭证数据库操作接口
type CloudCredentialsStore interface {
	GetCloudCredentials(cabinetID string) (*storage.CloudCredential, error)
	GetFirstCloudCredentials() (*storage.CloudCredential, error)
	SaveCloudCredentials(cabinetID, apiKey, apiSecret, cloudEndpoint string) error
}

// GetConfig 获取系统配置信息（用于前端）
// 从配置文件读取基本配置，从数据库优先读取API Key和Endpoint
func GetConfig(cfg interface {
	GetCloudConfig() (enabled bool, endpoint, apiKey, adminToken, cabinetID string)
	GetCabinetInfo() (name, location string, latitude, longitude, capacityKWh *float64, deviceModel string)
}, db CloudCredentialsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, endpoint, _, adminToken, cabinetID := cfg.GetCloudConfig()
		cabinetName, location, latitude, longitude, capacityKWh, deviceModel := cfg.GetCabinetInfo()

		// 优先从数据库读取API Key和Endpoint
		apiKey := ""
		if db != nil {
			// 先尝试按cabinet_id查询
			if cred, err := db.GetCloudCredentials(cabinetID); err == nil && cred != nil {
				apiKey = cred.APIKey
				if cred.CloudEndpoint != "" {
					endpoint = cred.CloudEndpoint
				}
			} else {
				// 如果找不到，尝试获取第一个凭证
				if cred, err := db.GetFirstCloudCredentials(); err == nil && cred != nil {
					apiKey = cred.APIKey
					if cred.CloudEndpoint != "" {
						endpoint = cred.CloudEndpoint
					}
				}
			}
		}

		cloudConfig := gin.H{
			"enabled":     enabled,
			"endpoint":    endpoint,
			"api_key":     apiKey,
			"admin_token": adminToken,
			"cabinet_id":  cabinetID,
		}

		// 添加储能柜详细信息（如果有的话）
		if cabinetName != "" {
			cloudConfig["cabinet_name"] = cabinetName
		}
		if location != "" {
			cloudConfig["location"] = location
		}
		if latitude != nil {
			cloudConfig["latitude"] = *latitude
		}
		if longitude != nil {
			cloudConfig["longitude"] = *longitude
		}
		if capacityKWh != nil {
			cloudConfig["capacity_kwh"] = *capacityKWh
		}
		if deviceModel != "" {
			cloudConfig["device_model"] = deviceModel
		}

		c.JSON(http.StatusOK, gin.H{
			"cloud": cloudConfig,
		})
	}
}

// UpdateConfig 更新系统配置（用于前端）
// API Key保存到数据库，其他配置保存到配置文件
func UpdateConfig(cfg interface {
	UpdateCloudConfig(enabled bool, endpoint string) error
	GetCloudConfig() (enabled bool, endpoint, apiKey, adminToken, cabinetID string)
}, db CloudCredentialsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Cloud struct {
				Enabled  *bool   `json:"enabled"`
				Endpoint *string `json:"endpoint"`
				APIKey   *string `json:"api_key"`
			} `json:"cloud"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数格式错误: " + err.Error(),
			})
			return
		}

		// 获取当前配置
		_, configEndpoint, _, _, cabinetID := cfg.GetCloudConfig()

		// 优先从数据库获取endpoint（数据库存储的是用户通过Web界面配置的正确值）
		currentEndpoint := configEndpoint
		if db != nil {
			if cred, err := db.GetCloudCredentials(cabinetID); err == nil && cred != nil && cred.CloudEndpoint != "" {
				currentEndpoint = cred.CloudEndpoint
			} else if cred, err := db.GetFirstCloudCredentials(); err == nil && cred != nil && cred.CloudEndpoint != "" {
				currentEndpoint = cred.CloudEndpoint
			}
		}

		// 验证输入
		if input.Cloud.Endpoint != nil && *input.Cloud.Endpoint != "" {
			// 简单的URL格式验证
			endpoint := *input.Cloud.Endpoint
			if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "INVALID_ENDPOINT",
					"message": "Cloud端地址必须以http://或https://开头",
				})
				return
			}
		}

		// 更新配置
		enabled := false
		if input.Cloud.Enabled != nil {
			enabled = *input.Cloud.Enabled
		}

		endpoint := currentEndpoint
		if input.Cloud.Endpoint != nil {
			endpoint = *input.Cloud.Endpoint
		}

		apiKey := ""
		if input.Cloud.APIKey != nil {
			apiKey = *input.Cloud.APIKey
		}

		if err := cfg.UpdateCloudConfig(enabled, endpoint); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "UPDATE_FAILED",
				"message": "更新配置失败: " + err.Error(),
			})
			return
		}

		// 如果提供了API Key字段，保存到数据库
		if input.Cloud.APIKey != nil && db != nil {
			if err := db.SaveCloudCredentials(cabinetID, apiKey, "", endpoint); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "UPDATE_CREDENTIALS_FAILED",
					"message": "更新API Key失败: " + err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "配置更新成功",
			"cloud": gin.H{
				"enabled":  enabled,
				"endpoint": endpoint,
				"api_key":  apiKey != "",
			},
		})
	}
}

// TestCloudConnection 测试Cloud端连接（代理请求，避免浏览器跨域问题）
// 优先从数据库获取endpoint，如果数据库中没有则使用配置文件中的值
func TestCloudConnection(cfg interface {
	GetCloudConfig() (enabled bool, endpoint, apiKey, adminToken, cabinetID string)
}, db CloudCredentialsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, endpoint, _, _, cabinetID := cfg.GetCloudConfig()

		// 优先从数据库获取endpoint（数据库中存储的是用户通过Web界面配置的值）
		if db != nil {
			if cred, err := db.GetCloudCredentials(cabinetID); err == nil && cred != nil && cred.CloudEndpoint != "" {
				endpoint = cred.CloudEndpoint
			} else if cred, err := db.GetFirstCloudCredentials(); err == nil && cred != nil && cred.CloudEndpoint != "" {
				endpoint = cred.CloudEndpoint
			}
		}

		if !enabled || endpoint == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "CLOUD_NOT_CONFIGURED",
				"message": "Cloud端未配置或未启用",
			})
			return
		}

		// 构建健康检查URL
		healthURL := strings.TrimSuffix(endpoint, "/api/v1") + "/health"

		// 创建HTTP客户端，设置超时
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		// 发送请求
		startTime := time.Now()
		resp, err := client.Get(healthURL)
		latency := time.Since(startTime).Milliseconds()

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "CONNECTION_FAILED",
				"message": fmt.Sprintf("无法连接到Cloud端: %v", err),
				"details": gin.H{
					"endpoint": healthURL,
					"latency":  latency,
				},
			})
			return
		}
		defer resp.Body.Close()

		// 读取响应
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "INVALID_RESPONSE",
				"message": fmt.Sprintf("响应格式错误: %v", err),
				"details": gin.H{
					"endpoint":    healthURL,
					"status_code": resp.StatusCode,
					"latency":     latency,
				},
			})
			return
		}

		// 返回成功结果
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "连接成功",
			"data":    result,
			"details": gin.H{
				"endpoint":    healthURL,
				"status_code": resp.StatusCode,
				"latency":     latency,
			},
		})
	}
}

// RegisterToCloud 注册到Cloud端（代理请求，避免浏览器跨域问题）
// 注册成功后自动将API Key保存到数据库
func RegisterToCloud(cfg interface {
	GetCloudConfig() (enabled bool, endpoint, apiKey, adminToken, cabinetID string)
}, db CloudCredentialsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, endpoint, _, _, cabinetID := cfg.GetCloudConfig()

		// 优先从数据库获取endpoint（数据库中存储的是用户通过Web界面配置的值）
		if db != nil {
			if cred, err := db.GetCloudCredentials(cabinetID); err == nil && cred != nil && cred.CloudEndpoint != "" {
				endpoint = cred.CloudEndpoint
			} else if cred, err := db.GetFirstCloudCredentials(); err == nil && cred != nil && cred.CloudEndpoint != "" {
				endpoint = cred.CloudEndpoint
			}
		}

		if !enabled || endpoint == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "CLOUD_NOT_CONFIGURED",
				"message": "Cloud端未配置或未启用",
			})
			return
		}

		// 获取注册信息
		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数格式错误: " + err.Error(),
			})
			return
		}

		// 从payload中获取cabinet_id（如果前端传递了）
		if id, ok := payload["cabinet_id"].(string); ok && id != "" {
			cabinetID = id
		}

		// 构建Cloud端注册URL
		registerURL := strings.TrimSuffix(endpoint, "/") + "/cabinets/register"

		// 将payload转为JSON
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "MARSHAL_FAILED",
				"message": fmt.Sprintf("数据序列化失败: %v", err),
			})
			return
		}

		// 创建HTTP客户端
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		// 发送POST请求到Cloud端
		startTime := time.Now()
		resp, err := client.Post(registerURL, "application/json", strings.NewReader(string(payloadBytes)))
		latency := time.Since(startTime).Milliseconds()

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "CONNECTION_FAILED",
				"message": fmt.Sprintf("无法连接到Cloud端: %v", err),
				"details": gin.H{
					"endpoint": registerURL,
					"latency":  latency,
				},
			})
			return
		}
		defer resp.Body.Close()

		// 读取响应
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "INVALID_RESPONSE",
				"message": fmt.Sprintf("响应格式错误: %v", err),
				"details": gin.H{
					"endpoint":    registerURL,
					"status_code": resp.StatusCode,
					"latency":     latency,
				},
			})
			return
		}

		// 如果注册成功且包含API凭证，自动保存到数据库
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			if data, ok := result["data"].(map[string]interface{}); ok {
				if apiKey, ok := data["api_key"].(string); ok && apiKey != "" {
					apiSecret := ""
					if s, ok := data["api_secret"].(string); ok {
						apiSecret = s
					}
					// 保存凭证到数据库
					if db != nil {
						if err := db.SaveCloudCredentials(cabinetID, apiKey, apiSecret, endpoint); err != nil {
							// 保存失败不影响注册结果，只记录警告
							result["warning"] = fmt.Sprintf("凭证保存到数据库失败: %v", err)
						} else {
							result["credentials_saved"] = true
							result["credentials_storage"] = "database"
						}
					} else {
						result["warning"] = "数据库未配置，凭证未保存"
					}
				}
			}
		}

		// 返回完整结果
		c.JSON(resp.StatusCode, result)
	}
}

// UpdateCloudCredentials 更新Cloud API凭证（注册成功后调用）
// 凭证保存到数据库
func UpdateCloudCredentials(cfg interface {
	GetCloudConfig() (enabled bool, endpoint, apiKey, adminToken, cabinetID string)
}, db CloudCredentialsStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			APIKey    string `json:"api_key" binding:"required"`
			APISecret string `json:"api_secret"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数格式错误: " + err.Error(),
			})
			return
		}

		// 获取配置中的cabinet_id和endpoint
		_, configEndpoint, _, _, cabinetID := cfg.GetCloudConfig()

		// 优先从数据库获取endpoint（避免覆盖用户配置的正确endpoint）
		endpoint := configEndpoint
		if db != nil {
			if cred, err := db.GetCloudCredentials(cabinetID); err == nil && cred != nil && cred.CloudEndpoint != "" {
				endpoint = cred.CloudEndpoint
			} else if cred, err := db.GetFirstCloudCredentials(); err == nil && cred != nil && cred.CloudEndpoint != "" {
				endpoint = cred.CloudEndpoint
			}
		}

		// 保存凭证到数据库
		if db != nil {
			if err := db.SaveCloudCredentials(cabinetID, input.APIKey, input.APISecret, endpoint); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "UPDATE_FAILED",
					"message": "更新凭证失败: " + err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DATABASE_NOT_CONFIGURED",
				"message": "数据库未配置，无法保存凭证",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "API凭证已保存到数据库",
			"api_key": input.APIKey,
			"storage": "database",
			"note":    "api_secret已存储，请妥善保管原始密钥",
		})
	}
}

// UpdateConfigCabinetID 更新配置文件中的储能柜ID（保存基本信息时调用）
func UpdateConfigCabinetID(cfg interface {
	UpdateCabinetID(cabinetID string) error
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			CabinetID string `json:"cabinet_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数格式错误: " + err.Error(),
			})
			return
		}

		// 验证储能柜ID格式
		if len(input.CabinetID) == 0 || len(input.CabinetID) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_CABINET_ID",
				"message": "储能柜ID长度必须在1-64字符之间",
			})
			return
		}

		// 更新储能柜ID
		if err := cfg.UpdateCabinetID(input.CabinetID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "UPDATE_FAILED",
				"message": "更新储能柜ID失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "储能柜ID已保存到配置文件",
			"cabinet_id": input.CabinetID,
		})
	}
}

// GetSystemMAC 获取系统MAC地址
func GetSystemMAC() gin.HandlerFunc {
	return func(c *gin.Context) {
		macAddress := getSystemMACAddress()
		c.JSON(http.StatusOK, gin.H{
			"mac_address": macAddress,
		})
	}
}

// GetSystemIP 获取系统IP地址（API处理器）
func GetSystemIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		ipAddress := getSystemIPAddress()
		c.JSON(http.StatusOK, gin.H{
			"ip_address": ipAddress,
		})
	}
}

// getSystemMACAddress 获取系统MAC地址（内部函数）
func getSystemMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "00:00:00:00:00:00"
	}

	for _, iface := range interfaces {
		// 跳过回环接口和未启用的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 跳过虚拟网卡（docker、veth等）
		if strings.HasPrefix(iface.Name, "docker") ||
			strings.HasPrefix(iface.Name, "veth") ||
			strings.HasPrefix(iface.Name, "br-") ||
			strings.HasPrefix(iface.Name, "virbr") {
			continue
		}

		mac := iface.HardwareAddr.String()
		if mac != "" && mac != "00:00:00:00:00:00" {
			return strings.ToUpper(mac)
		}
	}

	return "00:00:00:00:00:00"
}

// getSystemIPAddress 获取系统主网卡IP地址（IPv4）
func getSystemIPAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "0.0.0.0"
	}

	for _, iface := range interfaces {
		// 跳过回环接口和未启用的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 跳过虚拟网卡（docker、veth等）
		if strings.HasPrefix(iface.Name, "docker") ||
			strings.HasPrefix(iface.Name, "veth") ||
			strings.HasPrefix(iface.Name, "br-") ||
			strings.HasPrefix(iface.Name, "virbr") {
			continue
		}

		// 获取接口的地址列表
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			// 检查是否为IP地址（排除非IP地址类型）
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// 只返回IPv4地址
			ip := ipNet.IP.To4()
			if ip != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}

	return "0.0.0.0"
}

// UpdateCabinetID 更新储能柜ID（同时更新所有设备的cabinet_id）
func UpdateCabinetID(deviceManager *device.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			OldCabinetID string `json:"old_cabinet_id" binding:"required"`
			NewCabinetID string `json:"new_cabinet_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 验证新的cabinet_id格式
		if len(req.NewCabinetID) == 0 || len(req.NewCabinetID) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_CABINET_ID",
				"message": "储能柜ID长度必须在1-64字符之间",
			})
			return
		}

		// TODO: 实现批量更新设备的cabinet_id功能
		// affectedCount, err := deviceManager.UpdateCabinetID(req.OldCabinetID, req.NewCabinetID)

		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"message":        "储能柜ID更新功能待实现",
			"affected_count": 0,
			"old_cabinet_id": req.OldCabinetID,
			"new_cabinet_id": req.NewCabinetID,
		})
	}
}

// ========== 脆弱性评估API ==========

// GetCurrentVulnerability 获取当前脆弱性评估结果
func GetCurrentVulnerability(vulnService interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if vulnService == nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "SERVICE_UNAVAILABLE",
				"message": "脆弱性评估服务未启用",
			})
			return
		}

		// 类型断言获取服务实例
		service, ok := vulnService.(*vulnerability.Service)
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"error":   "INTERNAL_ERROR",
				"message": "服务类型错误",
			})
			return
		}

		assessment := service.GetCurrentAssessment()
		if assessment == nil {
			// 如果没有数据，尝试触发一次评估
			service.TriggerAssessment()

			// 等待一小段时间后再次检查
			time.Sleep(500 * time.Millisecond)
			assessment = service.GetCurrentAssessment()

			if assessment == nil {
				// 仍然没有数据，返回友好的错误信息
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"error":   "NO_DATA",
					"message": "暂无评估数据，正在执行首次评估，请稍后刷新",
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    assessment,
		})
	}
}

// GetVulnerabilityHistory 获取脆弱性评估历史
func GetVulnerabilityHistory(vulnService interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if vulnService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "SERVICE_UNAVAILABLE",
				"message": "脆弱性评估服务未启用",
			})
			return
		}

		service, ok := vulnService.(*vulnerability.Service)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "INTERNAL_ERROR",
				"message": "服务类型错误",
			})
			return
		}

		// 解析查询参数
		cabinetID := c.Query("cabinet_id")
		if cabinetID == "" {
			cabinetID = "CABINET-001" // 默认值
		}

		// 时间范围（默认最近24小时）
		startTimeStr := c.Query("start_time")
		endTimeStr := c.Query("end_time")

		var startTime, endTime time.Time
		var err error

		if startTimeStr != "" {
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "INVALID_TIME_FORMAT",
					"message": "起始时间格式错误，请使用RFC3339格式",
				})
				return
			}
		} else {
			startTime = time.Now().Add(-24 * time.Hour)
		}

		if endTimeStr != "" {
			endTime, err = time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "INVALID_TIME_FORMAT",
					"message": "结束时间格式错误，请使用RFC3339格式",
				})
				return
			}
		} else {
			endTime = time.Now()
		}

		// 限制条数
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		if limit <= 0 || limit > 1000 {
			limit = 100
		}

		// 查询历史数据
		assessments, err := service.GetAssessmentHistory(cabinetID, startTime, endTime, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "QUERY_FAILED",
				"message": "查询历史数据失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"assessments": assessments,
				"total":       len(assessments),
			},
		})
	}
}

// GetTransmissionMetrics 获取传输指标详情
func GetTransmissionMetrics(vulnService interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if vulnService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "SERVICE_UNAVAILABLE",
				"message": "脆弱性评估服务未启用",
			})
			return
		}

		service, ok := vulnService.(*vulnerability.Service)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "INTERNAL_ERROR",
				"message": "服务类型错误",
			})
			return
		}

		assessment := service.GetCurrentAssessment()
		if assessment == nil || assessment.TransmissionMetrics == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NO_DATA",
				"message": "暂无传输指标数据",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    assessment.TransmissionMetrics,
		})
	}
}

// TriggerAssessment 手动触发脆弱性评估
func TriggerAssessment(vulnService interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if vulnService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "SERVICE_UNAVAILABLE",
				"message": "脆弱性评估服务未启用",
			})
			return
		}

		service, ok := vulnService.(*vulnerability.Service)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "INTERNAL_ERROR",
				"message": "服务类型错误",
			})
			return
		}

		// 调用服务触发评估
		service.TriggerAssessment()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "脆弱性评估已触发，请稍后查看结果",
		})
	}
}

// DismissVulnerability 消除指定漏洞
func DismissVulnerability(vulnService interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if vulnService == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "SERVICE_UNAVAILABLE",
				"message": "脆弱性评估服务未启用",
			})
			return
		}

		service, ok := vulnService.(*vulnerability.Service)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "INTERNAL_ERROR",
				"message": "服务类型错误",
			})
			return
		}

		// 解析请求
		var req struct {
			VulnerabilityType string `json:"vulnerability_type" binding:"required"`
			Reason            string `json:"reason"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 消除漏洞
		if err := service.DismissVulnerability(req.VulnerabilityType, req.Reason); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "DISMISS_FAILED",
				"message": "消除漏洞失败: " + err.Error(),
			})
			return
		}

		// 重新触发评估以更新评分
		go service.TriggerAssessment()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "漏洞已消除，正在重新评估",
		})
	}
}

// CloudSyncInterface 定义Cloud同步服务接口
// 用于API handler依赖注入，避免循环依赖
type CloudSyncInterface interface {
	SyncCabinetInfo(cabinetID, name, location string, latitude, longitude, capacityKWh *float64) error
}

// SaveCabinetInfo 保存储能柜信息并同步到Cloud端
// @Summary 保存储能柜信息
// @Description 保存储能柜基本信息到本地配置，并自动同步到Cloud端
// @Tags Cabinet
// @Accept json
// @Produce json
// @Param input body object true "储能柜信息"
// @Success 200 {object} object
// @Failure 400 {object} object
// @Router /api/v1/cabinets/info [put]
func SaveCabinetInfo(cfg interface {
	UpdateCabinetInfo(cabinetID, name, location string, latitude, longitude, capacityKWh *float64, deviceModel string) error
}, cloudSync CloudSyncInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析请求参数
		var input struct {
			CabinetID   string   `json:"cabinet_id" binding:"required"`
			Name        string   `json:"name" binding:"required"`
			Location    string   `json:"location"`
			Latitude    *float64 `json:"latitude"`
			Longitude   *float64 `json:"longitude"`
			CapacityKWh *float64 `json:"capacity_kwh"`
			DeviceModel string   `json:"device_model"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "INVALID_REQUEST",
				"message": "请求参数错误: " + err.Error(),
			})
			return
		}

		// 验证cabinet_id格式
		if len(input.CabinetID) == 0 || len(input.CabinetID) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "INVALID_CABINET_ID",
				"message": "储能柜ID长度必须在1-64字符之间",
			})
			return
		}

		// 1. 先同步到Cloud端（使用配置文件中的旧cabinet_id）
		cloudSyncSuccess := false
		var cloudSyncError string

		if cloudSync != nil {
			err := cloudSync.SyncCabinetInfo(
				input.CabinetID,
				input.Name,
				input.Location,
				input.Latitude,
				input.Longitude,
				input.CapacityKWh,
			)

			if err != nil {
				cloudSyncError = err.Error()
				// Cloud同步失败，返回错误，不更新本地配置
				c.JSON(http.StatusOK, gin.H{
					"success":            false,
					"cloud_sync_success": false,
					"cloud_sync_error":   cloudSyncError,
					"message":            "同步到Cloud端失败，本地配置未更新",
				})
				return
			}
			cloudSyncSuccess = true
		}

		// 2. Cloud同步成功后，再更新本地配置文件中的储能柜信息
		configUpdateSuccess := false
		if err := cfg.UpdateCabinetInfo(
			input.CabinetID,
			input.Name,
			input.Location,
			input.Latitude,
			input.Longitude,
			input.CapacityKWh,
			input.DeviceModel,
		); err != nil {
			// 本地配置更新失败，但Cloud已同步成功
			c.JSON(http.StatusOK, gin.H{
				"success":               false,
				"config_update_success": false,
				"cloud_sync_success":    cloudSyncSuccess,
				"error":                 "CONFIG_UPDATE_FAILED",
				"message":               "Cloud端已同步成功，但本地配置更新失败: " + err.Error(),
			})
			return
		}
		configUpdateSuccess = true

		// 3. 全部成功，返回响应
		response := gin.H{
			"success":               true,
			"config_update_success": configUpdateSuccess,
			"cloud_sync_success":    cloudSyncSuccess,
			"message":               "储能柜信息已保存到Edge端",
		}

		if cloudSyncSuccess {
			response["message"] = "储能柜信息已保存到Edge端并成功同步到Cloud端"
		} else {
			// cloudSync为nil的情况（Cloud未启用）
			response["message"] = "储能柜信息已保存到Edge端（Cloud未启用）"
		}

		c.JSON(http.StatusOK, response)
	}
}
