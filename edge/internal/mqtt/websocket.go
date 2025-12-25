/*
 * WebSocket服务器
 * 用于实时推送MQTT数据到前端页面
 */
package mqtt

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type      string      `json:"type"`      // sensor_data, device_status, alert, heartbeat
	Data      interface{} `json:"data"`      // 具体数据
	Timestamp time.Time   `json:"timestamp"` // 消息时间戳
}

// client WebSocket客户端封装
type client struct {
	conn   *websocket.Conn
	send   chan []byte   // 发送消息通道
	hub    *WebSocketHub
	mutex  sync.Mutex    // 写入锁
}

// WebSocketHub WebSocket连接管理器
type WebSocketHub struct {
	logger     *zap.Logger
	clients    map[*client]bool
	broadcast  chan []byte
	register   chan *client
	unregister chan *client
	mutex      sync.RWMutex
}

// WebSocket升级器配置
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许跨域连接（生产环境中应该更严格）
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewWebSocketHub 创建WebSocket管理器
func NewWebSocketHub(logger *zap.Logger) *WebSocketHub {
	return &WebSocketHub{
		logger:     logger,
		clients:    make(map[*client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

// Run 启动WebSocket管理器
func (h *WebSocketHub) Run() {
	h.logger.Info("🚀 WebSocket Hub 已启动")

	for {
		select {
		case c := <-h.register:
			h.mutex.Lock()
			h.clients[c] = true
			h.mutex.Unlock()
			h.logger.Info("🔗 新的WebSocket客户端连接",
				zap.Int("total_clients", len(h.clients)))

		case c := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mutex.Unlock()
			h.logger.Info("🔌 WebSocket客户端断开连接",
				zap.Int("total_clients", len(h.clients)))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for c := range h.clients {
				select {
				case c.send <- message:
					// 消息发送成功
				default:
					// 发送通道已满,关闭客户端
					h.mutex.RUnlock()
					h.mutex.Lock()
					close(c.send)
					delete(h.clients, c)
					h.mutex.Unlock()
					h.mutex.RLock()
					h.logger.Warn("WebSocket客户端发送通道已满，已关闭")
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket升级失败", zap.Error(err))
		return
	}

	// 创建客户端
	c := &client{
		conn: conn,
		send: make(chan []byte, 256),
		hub:  h,
	}

	// 注册新客户端
	h.register <- c

	// 启动读写协程
	go c.writePump()
	go c.readPump()
}

// readPump 从WebSocket连接读取消息
func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// 设置读取参数
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 读取客户端消息（主要用于保持连接）
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Warn("WebSocket连接异常关闭", zap.Error(err))
			}
			break
		}
	}
}

// writePump 向WebSocket连接写入消息
func (c *client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub关闭了发送通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 使用互斥锁保护写入
			c.mutex.Lock()
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			c.mutex.Unlock()

			if err != nil {
				c.hub.logger.Warn("WebSocket写入失败", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			// 使用互斥锁保护写入
			c.mutex.Lock()
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			c.mutex.Unlock()

			if err != nil {
				c.hub.logger.Warn("WebSocket ping失败", zap.Error(err))
				return
			}
		}
	}
}

// BroadcastSensorData 广播传感器数据
func (h *WebSocketHub) BroadcastSensorData(data *SensorData) {
	message := WebSocketMessage{
		Type:      "sensor_data",
		Data:      data,
		Timestamp: time.Now(),
	}
	h.broadcastMessage(message)
}

// BroadcastDeviceStatus 广播设备状态
func (h *WebSocketHub) BroadcastDeviceStatus(status *DeviceStatus) {
	message := WebSocketMessage{
		Type:      "device_status",
		Data:      status,
		Timestamp: time.Now(),
	}
	h.broadcastMessage(message)
}

// BroadcastAlert 广播告警信息
func (h *WebSocketHub) BroadcastAlert(alert *Alert) {
	message := WebSocketMessage{
		Type:      "alert",
		Data:      alert,
		Timestamp: time.Now(),
	}
	h.broadcastMessage(message)
}

// BroadcastHeartbeat 广播心跳信息
func (h *WebSocketHub) BroadcastHeartbeat(heartbeat *Heartbeat) {
	message := WebSocketMessage{
		Type:      "heartbeat",
		Data:      heartbeat,
		Timestamp: time.Now(),
	}
	h.broadcastMessage(message)
}

// broadcastMessage 广播消息到所有客户端
func (h *WebSocketHub) broadcastMessage(message WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("序列化WebSocket消息失败", zap.Error(err))
		return
	}

	select {
	case h.broadcast <- data:
		h.logger.Debug("📤 WebSocket消息已发送", 
			zap.String("type", message.Type),
			zap.Int("clients", len(h.clients)))
	default:
		h.logger.Warn("WebSocket广播通道已满，跳过消息")
	}
}

// GetClientCount 获取当前连接的客户端数量
func (h *WebSocketHub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}
