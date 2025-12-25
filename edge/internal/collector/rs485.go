/*
 * RS485通信模块
 * 负责通过RS485接口接收传感器数据
 */
package collector

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/edge/storage-cabinet/pkg/models"
	"go.uber.org/zap"
)

// RS485Protocol RS485通信协议接口
type RS485Protocol interface {
	ParseFrame(data []byte) (*SensorFrame, error)
	BuildQueryCommand(deviceAddr byte, registerAddr uint16) []byte
}

// ModbusRTU Modbus RTU协议实现
type ModbusRTU struct{}

// SensorFrame 传感器数据帧
type SensorFrame struct {
	DeviceID   string
	SensorType models.SensorType
	Value      float64
	Unit       string
	Timestamp  time.Time
	Quality    int
}

// RS485Collector RS485数据采集器
type RS485Collector struct {
	logger     *zap.Logger
	port       io.ReadWriteCloser
	protocol   RS485Protocol
	devices    map[byte]*RS485Device // 设备地址映射
	dataChan   chan *SensorFrame
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
}

// RS485Device RS485设备信息
type RS485Device struct {
	Address    byte              // Modbus地址
	DeviceID   string            // 设备ID
	SensorType models.SensorType // 传感器类型
	RegisterMap map[uint16]string // 寄存器映射
}

// NewRS485Collector 创建RS485采集器
func NewRS485Collector(logger *zap.Logger, port io.ReadWriteCloser) *RS485Collector {
	return &RS485Collector{
		logger:   logger,
		port:     port,
		protocol: &ModbusRTU{},
		devices:  make(map[byte]*RS485Device),
		dataChan: make(chan *SensorFrame, 100),
		stopChan: make(chan struct{}),
	}
}

// RegisterDevice 注册RS485设备
func (c *RS485Collector) RegisterDevice(device *RS485Device) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.devices[device.Address] = device
	c.logger.Info("RS485 device registered",
		zap.String("device_id", device.DeviceID),
		zap.Uint8("address", device.Address),
		zap.String("sensor_type", string(device.SensorType)))
}

// Start 启动采集
func (c *RS485Collector) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("collector already running")
	}
	c.running = true
	c.mu.Unlock()

	// 启动读取协程
	go c.readLoop()
	
	// 启动轮询协程
	go c.pollLoop()

	c.logger.Info("RS485 collector started")
	return nil
}

// Stop 停止采集
func (c *RS485Collector) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	close(c.stopChan)
	c.logger.Info("RS485 collector stopped")
}

// readLoop 读取数据循环
func (c *RS485Collector) readLoop() {
	buffer := make([]byte, 256)
	
	for {
		select {
		case <-c.stopChan:
			return
		default:
			// 设置读取超时
			if deadliner, ok := c.port.(interface{ SetReadDeadline(time.Time) error }); ok {
				deadliner.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			}
			
			n, err := c.port.Read(buffer)
			if err != nil {
				if err != io.EOF {
					c.logger.Debug("Read error", zap.Error(err))
				}
				continue
			}
			
			if n > 0 {
				// 解析数据帧
				frame, err := c.protocol.ParseFrame(buffer[:n])
				if err != nil {
					c.logger.Debug("Parse frame error", zap.Error(err))
					continue
				}
				
				// 发送到数据通道
				select {
				case c.dataChan <- frame:
				default:
					c.logger.Warn("Data channel full, dropping frame")
				}
			}
		}
	}
}

// pollLoop 轮询设备循环
func (c *RS485Collector) pollLoop() {
	ticker := time.NewTicker(30 * time.Second) // 30秒轮询一次
	defer ticker.Stop()
	
	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.pollAllDevices()
		}
	}
}

// pollAllDevices 轮询所有设备
func (c *RS485Collector) pollAllDevices() {
	c.mu.RLock()
	devices := make([]*RS485Device, 0, len(c.devices))
	for _, dev := range c.devices {
		devices = append(devices, dev)
	}
	c.mu.RUnlock()
	
	for _, device := range devices {
		// 根据传感器类型查询不同的寄存器
		registers := c.getRegistersByType(device.SensorType)
		
		for _, reg := range registers {
			cmd := c.protocol.BuildQueryCommand(device.Address, reg)
			if _, err := c.port.Write(cmd); err != nil {
				c.logger.Error("Failed to send query command",
					zap.String("device_id", device.DeviceID),
					zap.Error(err))
				continue
			}
			
			// 等待响应
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// getRegistersByType 根据传感器类型获取寄存器地址
func (c *RS485Collector) getRegistersByType(sensorType models.SensorType) []uint16 {
	switch sensorType {
	case models.SensorCO2:
		return []uint16{0x0000} // CO2浓度寄存器
	case models.SensorCO:
		return []uint16{0x0001} // CO浓度寄存器
	case models.SensorSmoke:
		return []uint16{0x0002} // 烟雾浓度寄存器
	case models.SensorLiquidLevel:
		return []uint16{0x0003} // 液位寄存器
	case models.SensorConductivity: // 电导率
		return []uint16{0x0004} // 电导率寄存器
	case models.SensorTemperature:
		return []uint16{0x0005} // 温度寄存器
	case models.SensorFlow:
		return []uint16{0x0006} // 流速寄存器
	default:
		return []uint16{}
	}
}

// GetDataChannel 获取数据通道
func (c *RS485Collector) GetDataChannel() <-chan *SensorFrame {
	return c.dataChan
}

// ParseFrame 解析Modbus RTU数据帧
func (m *ModbusRTU) ParseFrame(data []byte) (*SensorFrame, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("frame too short")
	}
	
	// 检查CRC
	if !m.checkCRC(data) {
		return nil, fmt.Errorf("CRC check failed")
	}
	
	// 解析Modbus响应
	deviceAddr := data[0]
	functionCode := data[1]
	
	if functionCode != 0x03 && functionCode != 0x04 {
		return nil, fmt.Errorf("unsupported function code: %02x", functionCode)
	}
	
	byteCount := data[2]
	if len(data) < int(3+byteCount+2) {
		return nil, fmt.Errorf("incomplete frame")
	}
	
	// 提取数据值（假设是16位整数）
	value := binary.BigEndian.Uint16(data[3:5])
	
	// 创建传感器帧
	frame := &SensorFrame{
		DeviceID:  fmt.Sprintf("RS485_%02X", deviceAddr),
		Value:     float64(value),
		Timestamp: time.Now(),
		Quality:   100,
	}
	
	return frame, nil
}

// BuildQueryCommand 构建Modbus查询命令
func (m *ModbusRTU) BuildQueryCommand(deviceAddr byte, registerAddr uint16) []byte {
	// 构建Modbus RTU读保持寄存器命令（功能码03）
	cmd := make([]byte, 8)
	cmd[0] = deviceAddr                          // 设备地址
	cmd[1] = 0x03                                // 功能码：读保持寄存器
	binary.BigEndian.PutUint16(cmd[2:4], registerAddr) // 起始地址
	binary.BigEndian.PutUint16(cmd[4:6], 1)           // 寄存器数量
	
	// 计算CRC
	crc := m.calculateCRC(cmd[:6])
	binary.LittleEndian.PutUint16(cmd[6:8], crc)
	
	return cmd
}

// checkCRC 检查CRC
func (m *ModbusRTU) checkCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	
	dataLen := len(data) - 2
	calculatedCRC := m.calculateCRC(data[:dataLen])
	receivedCRC := binary.LittleEndian.Uint16(data[dataLen:])
	
	return calculatedCRC == receivedCRC
}

// calculateCRC 计算Modbus CRC16
func (m *ModbusRTU) calculateCRC(data []byte) uint16 {
	crc := uint16(0xFFFF)
	
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc = crc >> 1
			}
		}
	}
	
	return crc
}

// DefaultSensorRegisters 默认传感器寄存器配置
var DefaultSensorRegisters = map[models.SensorType]map[string]uint16{
	models.SensorCO2: {
		"concentration": 0x0000,
		"status":       0x0010,
	},
	models.SensorCO: {
		"concentration": 0x0001,
		"status":       0x0011,
	},
	models.SensorSmoke: {
		"concentration": 0x0002,
		"alarm":        0x0012,
	},
	models.SensorLiquidLevel: {
		"level":  0x0003,
		"status": 0x0013,
	},
	models.SensorConductivity: { // 电导率
		"conductivity": 0x0004,
		"temperature":  0x0014,
	},
	models.SensorTemperature: {
		"temperature": 0x0005,
		"humidity":    0x0015, // 可能同时测量湿度
	},
	models.SensorFlow: {
		"flow_rate": 0x0006,
		"total":     0x0016,
	},
}
