package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud-system/internal/config"
	"cloud-system/internal/models"
	"cloud-system/internal/mqtt"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AlertService 告警服务接口
type AlertService interface {
	// ListAlerts 获取告警列表
	ListAlerts(ctx context.Context, filter *models.AlertListFilter) ([]*models.Alert, int64, error)

	// GetAlert 获取告警详情
	GetAlert(ctx context.Context, alertID string) (*models.Alert, error)

	// ResolveAlert 解决告警
	ResolveAlert(ctx context.Context, alertID string, resolvedBy string) error

	// GetCabinetAlerts 获取储能柜的告警
	GetCabinetAlerts(ctx context.Context, cabinetID string) ([]*models.Alert, error)

	// CalculateHealthScore 计算储能柜健康评分
	CalculateHealthScore(ctx context.Context, cabinetID string) (float64, error)

	// SyncAlerts 接收Edge端同步的告警数据
	SyncAlerts(ctx context.Context, request *models.AlertSyncRequest) error

	// BatchResolveAlerts 批量解决告警
	BatchResolveAlerts(ctx context.Context, alertIDs []string, resolvedBy string) error
}

// alertService 告警服务实现
type alertService struct {
	alertRepo      repository.AlertRepository
	cabinetRepo    repository.CabinetRepository
	httpClient     *http.Client
	edgeAPIConfig  config.EdgeAPIConfig
	edgeMQTTClient *mqtt.EdgeClient // MQTT客户端用于下发命令到Edge
}

// NewAlertService 创建告警服务实例
func NewAlertService(
	alertRepo repository.AlertRepository,
	cabinetRepo repository.CabinetRepository,
	edgeCfg config.EdgeAPIConfig,
) AlertService {
	timeout := 5 * time.Second
	if d, err := config.ParseDuration(edgeCfg.Timeout); err == nil {
		timeout = d
	}

	return &alertService{
		alertRepo:     alertRepo,
		cabinetRepo:   cabinetRepo,
		httpClient:    &http.Client{Timeout: timeout},
		edgeAPIConfig: edgeCfg,
	}
}

// SetEdgeMQTTClient 设置Edge MQTT客户端(用于下发命令)
func (s *alertService) SetEdgeMQTTClient(client *mqtt.EdgeClient) {
	s.edgeMQTTClient = client
	utils.Info("Edge MQTT client set for alert service command dispatch")
}

// ListAlerts 获取告警列表
func (s *alertService) ListAlerts(ctx context.Context, filter *models.AlertListFilter) ([]*models.Alert, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}

	var err error
	filter.PageSize, err = utils.ValidatePageSize(filter.PageSize, 100)
	if err != nil {
		return nil, 0, err
	}

	alerts, total, err := s.alertRepo.List(ctx, filter)
	if err != nil {
		utils.Error("Failed to list alerts", zap.Error(err))
		return nil, 0, err
	}

	return alerts, total, nil
}

// GetAlert 获取告警详情
func (s *alertService) GetAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	alert, err := s.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		return nil, err
	}

	return alert, nil
}

// ResolveAlert 解决告警
func (s *alertService) ResolveAlert(ctx context.Context, alertID string, resolvedBy string) error {
	return s.resolveSingleAlert(ctx, alertID, resolvedBy)
}

// GetCabinetAlerts 获取储能柜的告警
func (s *alertService) GetCabinetAlerts(ctx context.Context, cabinetID string) ([]*models.Alert, error) {
	// 验证储能柜是否存在
	exists, err := s.cabinetRepo.Exists(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	alerts, err := s.alertRepo.GetRecentByCabinet(ctx, cabinetID, 5)
	if err != nil {
		utils.Error("Failed to get cabinet alerts",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return nil, err
	}

	return alerts, nil
}

// CalculateHealthScore 已废弃：改用脆弱性综合评分(latest_vulnerability_score)
// 脆弱性评分由Edge端计算并同步到Cloud端，更全面准确
// 该方法保留用于兼容性，直接返回最新的脆弱性评分
func (s *alertService) CalculateHealthScore(ctx context.Context, cabinetID string) (float64, error) {
	cabinet, err := s.cabinetRepo.GetByID(ctx, cabinetID)
	if err != nil {
		return 0, err
	}

	// 直接返回脆弱性评分
	return cabinet.LatestVulnerabilityScore, nil
}

func (s *alertService) resolveSingleAlert(ctx context.Context, alertID string, resolvedBy string) error {
	alert, err := s.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		return err
	}

	if alert.Resolved {
		return errors.New(errors.ErrBadRequest, "告警已解决")
	}

	if alert.EdgeAlertID != nil {
		if err := s.notifyEdgeResolve(ctx, alert); err != nil {
			return err
		}
	} else {
		utils.Warn("Edge alert id missing, skipping edge sync",
			zap.String("alert_id", alertID),
			zap.String("cabinet_id", alert.CabinetID),
		)
	}

	if err := s.alertRepo.Resolve(ctx, alertID, resolvedBy); err != nil {
		utils.Error("Failed to resolve alert",
			zap.String("alert_id", alertID),
			zap.Error(err),
		)
		return err
	}

	utils.Info("Alert resolved successfully",
		zap.String("alert_id", alertID),
		zap.String("resolved_by", resolvedBy),
	)

	return nil
}

func (s *alertService) notifyEdgeResolve(ctx context.Context, alert *models.Alert) error {
	// 优先使用MQTT下发命令,如果MQTT不可用则回退到HTTP
	if s.edgeMQTTClient != nil && s.edgeMQTTClient.IsConnected() {
		return s.notifyEdgeResolveViaMQTT(ctx, alert)
	}

	// 回退到HTTP方式
	utils.Warn("Edge MQTT not available, falling back to HTTP",
		zap.String("cabinet_id", alert.CabinetID))
	return s.notifyEdgeResolveViaHTTP(ctx, alert)
}

// notifyEdgeResolveViaMQTT 通过MQTT下发告警解决命令
func (s *alertService) notifyEdgeResolveViaMQTT(_ context.Context, alert *models.Alert) error {
	if alert.EdgeAlertID == nil {
		return nil
	}

	// 构建MQTT命令消息
	command := map[string]interface{}{
		"command_id":   uuid.New().String(),
		"command_type": "resolve_alert",
		"payload": map[string]interface{}{
			"alert_id":  *alert.EdgeAlertID,
			"device_id": alert.DeviceID,
		},
		"timestamp": time.Now().Unix(),
	}

	payload, err := json.Marshal(command)
	if err != nil {
		return errors.Wrap(err, errors.ErrSyncFailed, "序列化告警解决命令失败")
	}

	// 发布到 cloud/cabinets/{cabinet_id}/commands/control topic
	// 根据 senddata.md 规范，resolve_alert 属于 control 类别
	topic := fmt.Sprintf("cloud/cabinets/%s/commands/control", alert.CabinetID)
	if err := s.edgeMQTTClient.Publish(topic, 1, false, payload); err != nil {
		utils.Error("Failed to publish alert resolve command via MQTT",
			zap.String("cabinet_id", alert.CabinetID),
			zap.Int64("edge_alert_id", *alert.EdgeAlertID),
			zap.Error(err))
		return errors.Wrap(err, errors.ErrSyncFailed, "发布告警解决命令失败")
	}

	utils.Info("Alert resolve command published via MQTT",
		zap.String("cabinet_id", alert.CabinetID),
		zap.String("topic", topic),
		zap.Int64("edge_alert_id", *alert.EdgeAlertID))

	return nil
}

// notifyEdgeResolveViaHTTP 通过HTTP调用Edge端解决告警
func (s *alertService) notifyEdgeResolveViaHTTP(ctx context.Context, alert *models.Alert) error {
	if alert.EdgeAlertID == nil {
		return nil
	}

	cabinet, err := s.cabinetRepo.GetByID(ctx, alert.CabinetID)
	if err != nil {
		return err
	}

	baseURL := strings.TrimSuffix(s.edgeAPIConfig.BaseURL, "/")
	if cabinet.IPAddress != nil && *cabinet.IPAddress != "" {
		scheme := s.edgeAPIConfig.Scheme
		if scheme == "" {
			scheme = "http"
		}
		port := s.edgeAPIConfig.Port
		if port == 0 {
			port = 8001
		}
		baseURL = fmt.Sprintf("%s://%s:%d/api/v1", scheme, *cabinet.IPAddress, port)
	}

	if baseURL == "" {
		return errors.New(errors.ErrInternalServer, "Edge API地址未配置")
	}

	url := fmt.Sprintf("%s/alerts/%d/resolve", strings.TrimSuffix(baseURL, "/"), *alert.EdgeAlertID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return errors.Wrap(err, errors.ErrSyncFailed, "构建Edge告警请求失败")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, errors.ErrSyncFailed, "调用Edge告警接口失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return errors.New(errors.ErrSyncFailed, fmt.Sprintf("Edge接口返回状态码: %d", resp.StatusCode))
	}

	return nil
}

// SyncAlerts 接收Edge端同步的告警数据
func (s *alertService) SyncAlerts(ctx context.Context, request *models.AlertSyncRequest) error {
	// 验证储能柜是否存在
	exists, err := s.cabinetRepo.Exists(ctx, request.CabinetID)
	if err != nil {
		return errors.Wrap(err, errors.ErrDatabaseQuery, "验证储能柜失败")
	}
	if !exists {
		return errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	// 批量创建告警
	successCount := 0
	skippedCount := 0

	for _, alertData := range request.Alerts {
		alert := &models.Alert{
			CabinetID:   request.CabinetID,
			EdgeAlertID: alertData.AlertID,
			AlertType:   alertData.AlertType,
			Severity:    alertData.Severity,
			Message:     alertData.Message,
			CreatedAt:   alertData.Timestamp,
			Resolved:    alertData.Resolved,
			ResolvedAt:  alertData.ResolvedAt,
		}

		if alert.Resolved {
			resolvedBy := "edge"
			alert.ResolvedBy = &resolvedBy
		}

		if alertData.DeviceID != "" {
			alert.DeviceID = &alertData.DeviceID
		}
		if alertData.Value != nil {
			alert.SensorValue = alertData.Value
		}

		alert.PopulateCalculatedFields()

		if err := s.alertRepo.CreateOrUpdate(ctx, alert); err != nil {
			utils.Error("Failed to create or update synced alert",
				zap.String("cabinet_id", request.CabinetID),
				zap.String("device_id", alertData.DeviceID),
				zap.String("alert_type", alertData.AlertType),
				zap.Error(err),
			)
			skippedCount++
			continue
		}
		successCount++
	}

	utils.Info("Alert sync completed",
		zap.String("cabinet_id", request.CabinetID),
		zap.Int("total", len(request.Alerts)),
		zap.Int("success", successCount),
		zap.Int("skipped", skippedCount))

	// 更新储能柜最后同步时间（将status从offline更新为active）
	if err := s.cabinetRepo.UpdateLastSyncTime(ctx, request.CabinetID); err != nil {
		utils.Warn("Failed to update cabinet last sync time after alert sync",
			zap.String("cabinet_id", request.CabinetID),
			zap.Error(err),
		)
		// 不影响主流程，继续返回成功
	}

	utils.Info("Alerts synced from Edge successfully",
		zap.String("cabinet_id", request.CabinetID),
		zap.Int("count", len(request.Alerts)),
	)

	return nil
}

// BatchResolveAlerts 批量解决告警
func (s *alertService) BatchResolveAlerts(ctx context.Context, alertIDs []string, resolvedBy string) error {
	successCount := 0
	failCount := 0
	var lastErr error

	for _, alertID := range alertIDs {
		if err := s.resolveSingleAlert(ctx, alertID, resolvedBy); err != nil {
			failCount++
			lastErr = err
			continue
		}
		successCount++
	}

	utils.Info("Batch resolve alerts completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
		zap.String("resolved_by", resolvedBy),
	)

	if failCount > 0 {
		if successCount == 0 && lastErr != nil {
			return lastErr
		}
		return errors.New(errors.ErrSyncFailed, fmt.Sprintf("部分告警解决失败：%d条", failCount))
	}

	return nil
}

// CreateAlert 创建告警（内部使用或Edge端调用）
func CreateAlert(ctx context.Context, alertRepo repository.AlertRepository, cabinetID string, alertType string, severity string, message string, deviceID *string, sensorValue *float64) error {
	alert := &models.Alert{
		AlertID:     uuid.New().String(),
		CabinetID:   cabinetID,
		AlertType:   alertType,
		Severity:    severity,
		Message:     message,
		DeviceID:    deviceID,
		SensorValue: sensorValue,
	}

	return alertRepo.Create(ctx, alert)
}
