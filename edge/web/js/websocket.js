/**
 * WebSocket统一管理器
 * 负责WebSocket连接管理、消息订阅/发布、自动重连
 * @version 1.0.0
 */

const WebSocketManager = {
    // WebSocket实例
    ws: null,
    
    // 连接状态
    connected: false,
    connecting: false,
    
    // 重连配置
    reconnectAttempts: 0,
    maxReconnectAttempts: 10,
    reconnectDelay: 1000,        // 初始重连延迟(毫秒)
    maxReconnectDelay: 30000,    // 最大重连延迟(毫秒)
    reconnectTimer: null,
    
    // 心跳配置
    heartbeatInterval: 30000,    // 心跳间隔(毫秒)
    heartbeatTimer: null,
    lastPongTime: null,
    
    // 消息订阅者
    subscribers: {
        sensor_data: [],
        device_status: [],
        alert: [],
        heartbeat: [],
        connection: []           // 连接状态变化订阅
    },
    
    // WebSocket URL
    wsUrl: null,

    /**
     * 初始化WebSocket管理器
     */
    init() {
        console.log('%c[WebSocketManager] 初始化', 'color: #10b981; font-weight: bold');
        
        // 构建WebSocket URL
        this.wsUrl = this.buildWebSocketUrl();
        console.log('[WebSocketManager] WebSocket URL:', this.wsUrl);
        
        // 建立连接
        this.connect();
        
        // 页面卸载时关闭连接
        window.addEventListener('beforeunload', () => {
            this.destroy();
        });
        
        // 页面可见性变化时处理连接
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'visible') {
                // 页面变为可见时，检查连接状态
                if (!this.connected && !this.connecting) {
                    console.log('[WebSocketManager] 页面可见，检查连接...');
                    this.connect();
                }
            }
        });
    },

    /**
     * 构建WebSocket URL
     * 支持本地开发、IDE端口转发、内网穿透等多种环境
     * @returns {string} WebSocket URL
     */
    buildWebSocketUrl() {
        const hostname = window.location.hostname;
        const port = window.location.port;
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        
        // 本地开发环境
        if (hostname === 'localhost' || hostname === '127.0.0.1') {
            return `ws://${hostname}:8001/ws`;
        }
        
        // IDE端口转发环境
        if (port === '63443' || parseInt(port) > 60000) {
            return 'ws://localhost:8001/ws';
        }
        
        // 内网穿透或生产环境：使用相同的host（不额外指定端口）
        // 内网穿透会将WebSocket请求路由到后端
        if (!port || port === '80' || port === '443') {
            return `${protocol}//${hostname}/ws`;
        }
        
        // 其他情况：带端口访问（如直接访问IP:8001）
        return `${protocol}//${hostname}:${port}/ws`;
    },

    /**
     * 建立WebSocket连接
     */
    connect() {
        if (this.connecting || this.connected) {
            console.log('[WebSocketManager] 已连接或正在连接中，跳过');
            return;
        }
        
        this.connecting = true;
        console.log('[WebSocketManager] 正在连接...');
        
        try {
            this.ws = new WebSocket(this.wsUrl);
            
            this.ws.onopen = this.onOpen.bind(this);
            this.ws.onmessage = this.onMessage.bind(this);
            this.ws.onclose = this.onClose.bind(this);
            this.ws.onerror = this.onError.bind(this);
            
        } catch (error) {
            console.error('[WebSocketManager] 连接失败:', error);
            this.connecting = false;
            this.scheduleReconnect();
        }
    },

    /**
     * 连接成功回调
     */
    onOpen(event) {
        console.log('%c[WebSocketManager] ✅ 连接成功', 'color: #10b981; font-weight: bold');
        
        this.connected = true;
        this.connecting = false;
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        
        // 启动心跳
        this.startHeartbeat();
        
        // 通知订阅者连接状态变化
        this.notifySubscribers('connection', { status: 'connected' });
        
        // 更新UI状态
        this.updateConnectionStatus('connected');
    },

    /**
     * 收到消息回调
     */
    onMessage(event) {
        try {
            const message = JSON.parse(event.data);
            
            // 记录收到的消息(调试用)
            if (message.type !== 'heartbeat') {
                console.log('[WebSocketManager] 📥 收到消息:', message.type, message.data);
            }
            
            // 更新最后pong时间(用于心跳检测)
            this.lastPongTime = Date.now();
            
            // 分发消息给订阅者
            this.notifySubscribers(message.type, message.data);
            
        } catch (error) {
            console.error('[WebSocketManager] 解析消息失败:', error, event.data);
        }
    },

    /**
     * 连接关闭回调
     */
    onClose(event) {
        console.warn('[WebSocketManager] 🔌 连接关闭:', event.code, event.reason);
        
        this.connected = false;
        this.connecting = false;
        this.ws = null;
        
        // 停止心跳
        this.stopHeartbeat();
        
        // 通知订阅者连接状态变化
        this.notifySubscribers('connection', { status: 'disconnected', code: event.code });
        
        // 更新UI状态
        this.updateConnectionStatus('disconnected');
        
        // 计划重连
        this.scheduleReconnect();
    },

    /**
     * 连接错误回调
     */
    onError(error) {
        console.error('[WebSocketManager] ❌ 连接错误:', error);
        // onClose会在onError之后被调用，所以这里不需要做太多处理
    },

    /**
     * 计划重连
     */
    scheduleReconnect() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
        }
        
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.warn('[WebSocketManager] ⚠️ 达到最大重连次数，停止重连');
            this.notifySubscribers('connection', { status: 'failed' });
            this.updateConnectionStatus('failed');
            return;
        }
        
        this.reconnectAttempts++;
        
        // 指数退避
        const delay = Math.min(
            this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1),
            this.maxReconnectDelay
        );
        
        console.log(`[WebSocketManager] ⏳ ${(delay/1000).toFixed(1)}秒后重连 (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
        
        this.reconnectTimer = setTimeout(() => {
            this.connect();
        }, delay);
    },

    /**
     * 启动心跳
     */
    startHeartbeat() {
        this.stopHeartbeat();
        
        this.heartbeatTimer = setInterval(() => {
            if (this.connected && this.ws) {
                // 发送ping消息
                try {
                    this.ws.send(JSON.stringify({ type: 'ping' }));
                } catch (error) {
                    console.warn('[WebSocketManager] 发送心跳失败:', error);
                }
            }
        }, this.heartbeatInterval);
    },

    /**
     * 停止心跳
     */
    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    },

    /**
     * 订阅消息
     * @param {string} type - 消息类型 (sensor_data, device_status, alert, heartbeat, connection)
     * @param {Function} callback - 回调函数
     * @returns {Function} 取消订阅的函数
     */
    subscribe(type, callback) {
        if (!this.subscribers[type]) {
            this.subscribers[type] = [];
        }
        
        this.subscribers[type].push(callback);
        console.log(`[WebSocketManager] 订阅 ${type}, 当前订阅者数: ${this.subscribers[type].length}`);
        
        // 返回取消订阅的函数
        return () => {
            this.unsubscribe(type, callback);
        };
    },

    /**
     * 取消订阅
     * @param {string} type - 消息类型
     * @param {Function} callback - 回调函数
     */
    unsubscribe(type, callback) {
        if (!this.subscribers[type]) return;
        
        const index = this.subscribers[type].indexOf(callback);
        if (index > -1) {
            this.subscribers[type].splice(index, 1);
            console.log(`[WebSocketManager] 取消订阅 ${type}, 剩余订阅者数: ${this.subscribers[type].length}`);
        }
    },

    /**
     * 通知订阅者
     * @param {string} type - 消息类型
     * @param {any} data - 消息数据
     */
    notifySubscribers(type, data) {
        const subscribers = this.subscribers[type];
        if (!subscribers || subscribers.length === 0) return;
        
        subscribers.forEach(callback => {
            try {
                callback(data);
            } catch (error) {
                console.error(`[WebSocketManager] 订阅者回调执行失败 (${type}):`, error);
            }
        });
    },

    /**
     * 发送消息
     * @param {string} type - 消息类型
     * @param {any} data - 消息数据
     */
    send(type, data) {
        if (!this.connected || !this.ws) {
            console.warn('[WebSocketManager] 未连接，无法发送消息');
            return false;
        }
        
        try {
            this.ws.send(JSON.stringify({ type, data }));
            return true;
        } catch (error) {
            console.error('[WebSocketManager] 发送消息失败:', error);
            return false;
        }
    },

    /**
     * 更新连接状态UI显示
     * @param {string} status - 连接状态 (connected, disconnected, failed)
     */
    updateConnectionStatus(status) {
        const statusEl = document.querySelector('.system-status');
        if (!statusEl) return;
        
        const iconEl = statusEl.querySelector('i');
        const textEl = statusEl.querySelector('span');
        
        if (!iconEl || !textEl) return;
        
        switch (status) {
            case 'connected':
                iconEl.className = 'fas fa-circle status-online';
                textEl.textContent = '实时连接';
                break;
            case 'disconnected':
                iconEl.className = 'fas fa-circle status-warning';
                textEl.textContent = '重连中...';
                break;
            case 'failed':
                iconEl.className = 'fas fa-circle';
                iconEl.style.color = 'var(--danger-color)';
                textEl.textContent = '连接失败';
                break;
        }
    },

    /**
     * 获取连接状态
     * @returns {boolean} 是否已连接
     */
    isConnected() {
        return this.connected;
    },

    /**
     * 手动重连
     */
    reconnect() {
        console.log('[WebSocketManager] 手动触发重连');
        this.reconnectAttempts = 0;
        
        if (this.ws) {
            this.ws.close();
        }
        
        this.connect();
    },

    /**
     * 销毁WebSocket管理器
     */
    destroy() {
        console.log('[WebSocketManager] 销毁');
        
        // 停止心跳
        this.stopHeartbeat();
        
        // 取消重连计时器
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        
        // 关闭连接
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        
        // 清空订阅者
        Object.keys(this.subscribers).forEach(type => {
            this.subscribers[type] = [];
        });
        
        this.connected = false;
        this.connecting = false;
    }
};

// 导出到全局
window.WebSocketManager = WebSocketManager;

