package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud-system/internal/models"
	"cloud-system/internal/mqtt"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CommandService 命令服务接口
type CommandService interface {
	// SendCommand 发送命令到Edge端
	SendCommand(ctx context.Context, cabinetID string, request *models.SendCommandRequest, createdBy string) (*models.Command, error)

	// GetCommand 获取命令详情
	GetCommand(ctx context.Context, commandID string) (*models.Command, error)

	// ListCommands 获取命令列表
	ListCommands(ctx context.Context, filter *models.CommandListFilter) ([]*models.Command, int64, error)

	// AckCommand Edge端确认命令执行结果
	AckCommand(ctx context.Context, commandID string, apiKey string, status string, message string) error
}

// commandService 命令服务实现
type commandService struct {
	commandRepo repository.CommandRepository
	cabinetRepo repository.CabinetRepository
	mqttClient  *mqtt.Client
}

// NewCommandService 创建命令服务实例
func NewCommandService(
	commandRepo repository.CommandRepository,
	cabinetRepo repository.CabinetRepository,
	mqttClient *mqtt.Client,
) CommandService {
	return &commandService{
		commandRepo: commandRepo,
		cabinetRepo: cabinetRepo,
		mqttClient:  mqttClient,
	}
}

// SendCommand 发送命令到Edge端
func (s *commandService) SendCommand(ctx context.Context, cabinetID string, request *models.SendCommandRequest, createdBy string) (*models.Command, error) {
	// 验证储能柜是否存在
	exists, err := s.cabinetRepo.Exists(ctx, cabinetID)
	if err != nil {
		utils.Error("Failed to check cabinet existence",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return nil, err
	}

	if !exists {
		return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	// 验证命令类型
	if !models.IsValidCommandType(request.CommandType) {
		return nil, errors.New(errors.ErrValidation, "无效的命令类型")
	}

	// 序列化Payload
	payloadBytes, err := json.Marshal(request.Payload)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrBadRequest, "命令payload格式错误")
	}

	// 创建命令记录
	commandID := uuid.New().String()
	command := &models.Command{
		CommandID:   commandID,
		CabinetID:   cabinetID,
		CommandType: request.CommandType,
		Payload:     string(payloadBytes),
		CreatedBy:   createdBy,
	}

	if err := s.commandRepo.Create(ctx, command); err != nil {
		utils.Error("Failed to create command",
			zap.String("command_id", commandID),
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return nil, err
	}

	// 异步发送MQTT命令
	go func() {
		if err := s.sendMQTTCommand(cabinetID, commandID, request.CommandType, request.Payload); err != nil {
			utils.Error("Failed to send MQTT command",
				zap.String("command_id", commandID),
				zap.String("cabinet_id", cabinetID),
				zap.Error(err),
			)

			// 更新命令状态为失败
			failResult := fmt.Sprintf("MQTT发送失败: %v", err)
			_ = s.commandRepo.UpdateStatus(context.Background(), commandID, "failed", &failResult)
		} else {
			// 标记为已发送
			_ = s.commandRepo.MarkAsSent(context.Background(), commandID)

			utils.Info("MQTT command sent successfully",
				zap.String("command_id", commandID),
				zap.String("cabinet_id", cabinetID),
				zap.String("command_type", request.CommandType),
			)
		}
	}()

	// 立即返回命令记录
	return command, nil
}

// sendMQTTCommand 通过MQTT发送命令
func (s *commandService) sendMQTTCommand(cabinetID string, commandID string, commandType string, payload map[string]interface{}) error {
	// 构建MQTT消息
	message := map[string]interface{}{
		"command_id":   commandID,
		"command_type": commandType,
		"payload":      payload,
		"timestamp":    time.Now().Unix(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化MQTT消息失败: %w", err)
	}

	// 发送到对应的MQTT主题
	topicType := commandType
	switch commandType {
	case "license_push":
		topicType = "license_update"
	case "license_revoke":
		topicType = "license_revoke"
	}
	topic := mqtt.GetCommandTopic(cabinetID, topicType)
	if err := s.mqttClient.Publish(topic, messageBytes); err != nil {
		return fmt.Errorf("MQTT发布失败: %w", err)
	}

	return nil
}

// GetCommand 获取命令详情
func (s *commandService) GetCommand(ctx context.Context, commandID string) (*models.Command, error) {
	command, err := s.commandRepo.GetByID(ctx, commandID)
	if err != nil {
		utils.Warn("Failed to get command",
			zap.String("command_id", commandID),
			zap.Error(err),
		)
		return nil, err
	}

	return command, nil
}

// ListCommands 获取命令列表
func (s *commandService) ListCommands(ctx context.Context, filter *models.CommandListFilter) ([]*models.Command, int64, error) {
	// 验证和设置默认值
	if filter.Page < 1 {
		filter.Page = 1
	}

	var err error
	filter.PageSize, err = utils.ValidatePageSize(filter.PageSize, 100)
	if err != nil {
		return nil, 0, err
	}

	// 验证命令类型（如果提供）
	if filter.CommandType != nil && *filter.CommandType != "" {
		if !models.IsValidCommandType(*filter.CommandType) {
			return nil, 0, errors.New(errors.ErrValidation, "无效的命令类型")
		}
	}

	// 验证状态（如果提供）
	if filter.Status != nil && *filter.Status != "" {
		if !models.IsValidCommandStatus(*filter.Status) {
			return nil, 0, errors.New(errors.ErrValidation, "无效的命令状态")
		}
	}

	// 从数据库获取
	commands, total, err := s.commandRepo.List(ctx, filter)
	if err != nil {
		utils.Error("Failed to list commands", zap.Error(err))
		return nil, 0, err
	}

	return commands, total, nil
}

// AckCommand Edge端确认命令执行结果
func (s *commandService) AckCommand(ctx context.Context, commandID string, apiKey string, status string, message string) error {
	if apiKey == "" {
		return errors.New(errors.ErrUnauthorized, "缺少API Key")
	}

	// 获取储能柜
	cabinet, err := s.cabinetRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return err
	}

	// 获取命令并验证归属
	command, err := s.commandRepo.GetByID(ctx, commandID)
	if err != nil {
		return err
	}

	if command.CabinetID != cabinet.CabinetID {
		return errors.New(errors.ErrForbidden, "命令不属于当前储能柜")
	}

	result := message
	if err := s.commandRepo.MarkAsCompleted(ctx, commandID, status, result); err != nil {
		return err
	}

	utils.Info("Command acknowledged",
		zap.String("command_id", commandID),
		zap.String("cabinet_id", cabinet.CabinetID),
		zap.String("status", status))
	return nil
}
