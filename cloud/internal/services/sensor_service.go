package services

import (
	"context"
	"fmt"

	"cloud-system/internal/models"
	"cloud-system/internal/repository"
	"cloud-system/internal/utils"
	"cloud-system/pkg/errors"

	"go.uber.org/zap"
)

// SensorService 传感器服务接口
type SensorService interface {
	// SaveSensorDataFromMQTT 保存来自MQTT的传感器数据
	SaveSensorDataFromMQTT(ctx context.Context, msg *MQTTSensorMessage) error

	// SyncSensorData 同步传感器数据（Edge端调用）
	SyncSensorData(ctx context.Context, cabinetID string, request *models.SyncDataRequest) (int, error)

	// GetLatestSensorData 获取储能柜的最新传感器数据
	GetLatestSensorData(ctx context.Context, cabinetID string) ([]*models.LatestSensorData, error)

	// GetHistoricalData 获取历史数据
	GetHistoricalData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.SensorData, int64, error)

	// GetAggregatedData 获取聚合数据
	GetAggregatedData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.AggregatedData, error)

	// ListDevices 获取储能柜下的传感器设备
	ListDevices(ctx context.Context, cabinetID string) ([]*models.SensorDevice, error)
}

// sensorService 传感器服务实现
type sensorService struct {
	sensorDataRepo   repository.SensorDataRepository
	sensorDeviceRepo repository.SensorDeviceRepository
	cabinetRepo      repository.CabinetRepository
	alertRepo        repository.AlertRepository
}

// NewSensorService 创建传感器服务实例
func NewSensorService(
	sensorDataRepo repository.SensorDataRepository,
	sensorDeviceRepo repository.SensorDeviceRepository,
	cabinetRepo repository.CabinetRepository,
	alertRepo repository.AlertRepository,
) SensorService {
	return &sensorService{
		sensorDataRepo:   sensorDataRepo,
		sensorDeviceRepo: sensorDeviceRepo,
		cabinetRepo:      cabinetRepo,
		alertRepo:        alertRepo,
	}
}

// SaveSensorDataFromMQTT 保存来自MQTT的传感器数据
func (s *sensorService) SaveSensorDataFromMQTT(ctx context.Context, msg *MQTTSensorMessage) error {
	// 获取设备信息以获取cabinet_id和sensor_type
	device, err := s.sensorDeviceRepo.GetByID(ctx, msg.DeviceID)
	if err != nil {
		// 如果设备不存在，尝试自动创建
		appErr, ok := err.(*errors.AppError)
		if !ok || appErr.Code != errors.ErrNotFound {
			// 如果不是"不存在"错误，直接返回
			return errors.Wrap(err, errors.ErrNotFound, "设备不存在")
		}

		// 设备不存在，尝试自动创建
		utils.Info("设备不存在，尝试自动创建",
			zap.String("device_id", msg.DeviceID),
			zap.String("sensor_type", msg.SensorType),
		)

		// 获取默认的cabinet_id（优先从已有设备中获取，否则使用默认值）
		cabinetID := s.getDefaultCabinetID(ctx)
		if cabinetID == "" {
			cabinetID = "CABINET_A1" // 默认值
			utils.Warn("使用默认cabinet_id",
				zap.String("device_id", msg.DeviceID),
				zap.String("cabinet_id", cabinetID),
			)
		}

		// 创建设备
		newDevice := &models.SensorDevice{
			DeviceID:   msg.DeviceID,
			CabinetID:  cabinetID,
			SensorType: msg.SensorType,
			Name:       s.generateDeviceName(msg.DeviceID, msg.SensorType),
			Unit:       msg.Unit,
			Status:     "active",
		}

		if err := s.sensorDeviceRepo.Create(ctx, newDevice); err != nil {
			utils.Error("自动创建设备失败",
				zap.String("device_id", msg.DeviceID),
				zap.String("sensor_type", msg.SensorType),
				zap.Error(err),
			)
			return errors.Wrap(err, errors.ErrDatabaseQuery, "自动创建设备失败")
		}

		utils.Info("设备自动创建成功",
			zap.String("device_id", msg.DeviceID),
			zap.String("sensor_type", msg.SensorType),
			zap.String("cabinet_id", cabinetID),
		)

		device = newDevice
	}

	// 转换为SensorData模型
	sensorData := &models.SensorData{
		DeviceID:  msg.DeviceID,
		Timestamp: msg.Timestamp,
		Value:     msg.Value,
		Quality:   msg.Quality,
		Status: func() string {
			if msg.Quality < 50 {
				return "error"
			} else if msg.Quality < 80 {
				return "warning"
			}
			return "normal"
		}(),
	}

	// 插入数据（需要传入cabinetID和sensorType）
	if err := s.sensorDataRepo.Insert(ctx, sensorData, device.CabinetID, msg.SensorType); err != nil {
		utils.Error("Failed to insert sensor data",
			zap.String("device_id", msg.DeviceID),
			zap.String("sensor_type", msg.SensorType),
			zap.Error(err),
		)
		return errors.Wrap(err, errors.ErrDatabaseQuery, "保存传感器数据失败")
	}

	return nil
}

// getDefaultCabinetID 获取默认的cabinet_id
// 优先从cabinets表中获取第一个，如果没有任何储能柜，返回空字符串
func (s *sensorService) getDefaultCabinetID(ctx context.Context) string {
	// 尝试从cabinets表中获取第一个cabinet_id
	if s.cabinetRepo != nil {
		cabinets, _, err := s.cabinetRepo.List(ctx, &models.CabinetListFilter{
			Page:     1,
			PageSize: 1,
		})
		if err == nil && len(cabinets) > 0 {
			return cabinets[0].CabinetID
		}
	}

	// 如果都没有，返回空字符串，让调用者使用默认值
	return ""
}

// generateDeviceName 根据device_id和sensor_type生成设备名称
func (s *sensorService) generateDeviceName(deviceID, sensorType string) string {
	// 如果device_id已经包含可读的名称，直接使用
	// 否则根据sensor_type生成名称
	sensorTypeNames := map[string]string{
		"co2":          "二氧化碳传感器",
		"co":           "一氧化碳传感器",
		"smoke":        "烟雾传感器",
		"liquid_level": "液位传感器",
		"conductivity": "电导率传感器",
		"temperature":  "温度传感器",
		"flow":         "流速传感器",
	}

	if name, ok := sensorTypeNames[sensorType]; ok {
		return name
	}

	return deviceID
}

// SyncSensorData 同步传感器数据（Edge端调用）
func (s *sensorService) SyncSensorData(ctx context.Context, cabinetID string, request *models.SyncDataRequest) (int, error) {
	// 验证储能柜ID匹配
	if request.CabinetID != cabinetID {
		return 0, errors.New(errors.ErrBadRequest, "请求体中的cabinet_id与URL路径不匹配")
	}

	// 验证储能柜是否存在
	exists, err := s.cabinetRepo.Exists(ctx, cabinetID)
	if err != nil {
		utils.Error("Failed to check cabinet existence",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return 0, err
	}

	if !exists {
		return 0, errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	// 验证数据量（告警通过/alerts/sync端点单独同步，这里只验证传感器数据）
	if len(request.SensorData) == 0 {
		return 0, errors.New(errors.ErrBadRequest, "传感器数据不能为空")
	}

	if len(request.SensorData) > 1000 {
		return 0, errors.New(errors.ErrBadRequest, "传感器数据量超过最大限制（1000条）")
	}

	totalSynced := 0

	// 处理传感器数据
	if len(request.SensorData) > 0 {
		// 验证并转换传感器数据
		sensorData := make([]models.SensorData, 0, len(request.SensorData))
		deviceMap := make(map[string]*models.SensorDevice) // 缓存设备信息

		for i, point := range request.SensorData {
			// 验证传感器类型
			if !models.IsValidSensorType(point.SensorType) {
				return 0, errors.New(
					errors.ErrValidation,
					fmt.Sprintf("第%d条数据：无效的传感器类型（%s）", i+1, point.SensorType),
				)
			}

			// 查询设备信息（缓存）
			device, exists := deviceMap[point.DeviceID]
			if !exists {
				dev, err := s.sensorDeviceRepo.GetByID(ctx, point.DeviceID)
				if err != nil {
					utils.Warn("Device not found, skipping sensor data",
						zap.String("device_id", point.DeviceID),
						zap.Error(err),
					)
					continue // 跳过该条数据
				}
				device = dev
				deviceMap[point.DeviceID] = device
			}

			// 验证设备属于当前储能柜
			if device.CabinetID != cabinetID {
				utils.Warn("Device belongs to different cabinet, skipping",
					zap.String("device_id", point.DeviceID),
					zap.String("device_cabinet_id", device.CabinetID),
					zap.String("expected_cabinet_id", cabinetID),
				)
				continue // 跳过该条数据
			}

			// 计算状态（根据质量）
			status := "normal"
			if point.Quality < 50 {
				status = "error"
			} else if point.Quality < 80 {
				status = "warning"
			}

			// 转换为SensorData
			sensorData = append(sensorData, models.SensorData{
				DeviceID:  point.DeviceID,
				Timestamp: point.Timestamp,
				Value:     point.Value,
				Quality:   point.Quality,
				Status:    status,
			})
		}

		// 批量插入传感器数据（需要包含cabinet_id和sensor_type）
		insertedCount := 0
		for i, d := range sensorData {
			device := deviceMap[d.DeviceID]
			err := s.sensorDataRepo.Insert(ctx, &d, device.CabinetID, request.SensorData[i].SensorType)
			if err != nil {
				utils.Warn("Failed to insert sensor data",
					zap.String("device_id", d.DeviceID),
					zap.String("sensor_type", request.SensorData[i].SensorType),
					zap.Error(err),
				)
				// 继续插入其他数据，不中断
				continue
			}
			insertedCount++
		}

		totalSynced += insertedCount
		utils.Info("Sensor data synced",
			zap.String("cabinet_id", cabinetID),
			zap.Int("total", len(request.SensorData)),
			zap.Int("inserted", insertedCount),
		)
	}

	// 告警数据通过 /alerts/sync 端点单独同步，这里不再处理

	// 更新储能柜最后同步时间
	if err := s.cabinetRepo.UpdateLastSyncTime(ctx, cabinetID); err != nil {
		utils.Warn("Failed to update cabinet last sync time",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		// 不影响主流程
	}

	utils.Info("Sensor data sync completed",
		zap.String("cabinet_id", cabinetID),
		zap.Int("sensor_data_count", len(request.SensorData)),
		zap.Int("total_synced", totalSynced),
	)

	return totalSynced, nil
}

// GetLatestSensorData 获取储能柜的最新传感器数据
func (s *sensorService) GetLatestSensorData(ctx context.Context, cabinetID string) ([]*models.LatestSensorData, error) {
	// 验证储能柜是否存在
	exists, err := s.cabinetRepo.Exists(ctx, cabinetID)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	// 从TimescaleDB获取最新数据
	data, err := s.sensorDataRepo.GetLatestByCabinetID(ctx, cabinetID)
	if err != nil {
		utils.Error("Failed to get latest sensor data",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "获取最新传感器数据失败")
	}

	return data, nil
}

// GetHistoricalData 获取历史数据
func (s *sensorService) GetHistoricalData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.SensorData, int64, error) {
	data, total, err := s.sensorDataRepo.GetHistoricalData(ctx, query)
	if err != nil {
		utils.Error("Failed to get historical sensor data",
			zap.String("device_id", query.DeviceID),
			zap.Error(err),
		)
		return nil, 0, errors.Wrap(err, errors.ErrDatabaseQuery, "获取历史传感器数据失败")
	}

	return data, total, nil
}

// GetAggregatedData 获取聚合数据
func (s *sensorService) GetAggregatedData(ctx context.Context, query *models.HistoricalDataQuery) ([]*models.AggregatedData, error) {
	data, err := s.sensorDataRepo.GetAggregatedData(ctx, query)
	if err != nil {
		utils.Error("Failed to get aggregated sensor data",
			zap.String("device_id", query.DeviceID),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "获取聚合传感器数据失败")
	}

	return data, nil
}

// ListDevices 获取储能柜下的传感器设备
func (s *sensorService) ListDevices(ctx context.Context, cabinetID string) ([]*models.SensorDevice, error) {
	// 验证储能柜是否存在
	exists, err := s.cabinetRepo.Exists(ctx, cabinetID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New(errors.ErrCabinetNotFound, "储能柜不存在")
	}

	devices, err := s.sensorDeviceRepo.ListByCabinetID(ctx, cabinetID)
	if err != nil {
		utils.Error("Failed to list sensor devices",
			zap.String("cabinet_id", cabinetID),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, errors.ErrDatabaseQuery, "获取传感器设备失败")
	}

	return devices, nil
}
