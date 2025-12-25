/**
 * 实时传感器监控模块
 * 负责获取和显示7种传感器的实时数据
 * 使用统一WebSocket管理器接收实时数据
 * @version 2.0.0
 */

const RealtimeMonitor = {
    // 传感器类型配置
    sensorTypes: {
        co2: {
            name: 'CO2',
            unit: 'ppm',
            timeId: 'co2Time',
            threshold: { max: 5000 }
        },
        co: {
            name: 'CO',
            unit: 'ppm',
            timeId: 'coTime',
            threshold: { max: 50 }
        },
        smoke: {
            name: '烟雾',
            unit: 'AD值',
            timeId: 'smokeTime',
            threshold: { max: 1000 }
        },
        liquid_level: {
            name: '液位',
            unit: 'mm',
            timeId: 'liquidLevelTime',
            threshold: { min: 0, max: 160 }
        },
        conductivity: {
            name: '电导率',
            unit: 'mS/cm',
            timeId: 'conductivityTime',
            threshold: { min: 0.5, max: 10 }
        },
        temperature: {
            name: '温度',
            unit: '°C',
            timeId: 'temperatureTime',
            threshold: { min: -10, max: 60 }
        },
        flow: {
            name: '流速',
            unit: 'L/min',
            timeId: 'flowTime',
            threshold: { min: 0.5, max: 100 }
        }
    },

    // 缓存每个传感器的最后一次数据
    lastSensorData: {},
    
    // 订阅取消函数
    unsubscribeSensorData: null,
    unsubscribeDeviceStatus: null,
    unsubscribeConnection: null,

    /**
     * 初始化实时监控
     */
    async init() {
        console.log('[RealtimeMonitor] 初始化实时传感器监控...');

        // 加载阈值配置
        if (typeof ThresholdManager !== 'undefined') {
            await ThresholdManager.loadThresholds();
            this.updateThresholdDisplays();
        }

        // 订阅WebSocket消息
        this.subscribeWebSocket();

        // 加载初始数据(HTTP请求获取最后一次数据)
        await this.loadInitialData();
        
        console.log('[RealtimeMonitor] 初始化完成');
    },

    /**
     * 订阅WebSocket消息
     */
    subscribeWebSocket() {
        // 确保WebSocketManager已初始化
        if (typeof WebSocketManager === 'undefined') {
            console.error('[RealtimeMonitor] WebSocketManager未定义');
            return;
        }

        // 订阅传感器数据
        this.unsubscribeSensorData = WebSocketManager.subscribe('sensor_data', (data) => {
            this.handleSensorData(data);
        });

        // 订阅设备状态变化
        this.unsubscribeDeviceStatus = WebSocketManager.subscribe('device_status', (data) => {
            this.handleDeviceStatus(data);
        });

        // 订阅连接状态变化
        this.unsubscribeConnection = WebSocketManager.subscribe('connection', (data) => {
            this.handleConnectionChange(data);
        });

        console.log('[RealtimeMonitor] WebSocket订阅完成');
    },

    /**
     * 处理传感器数据
     * @param {Object} data - 传感器数据
     */
    handleSensorData(data) {
        if (data && data.sensor_type) {
            console.log(`[RealtimeMonitor] 收到传感器数据: ${data.sensor_type} = ${data.value}`);
            this.updateSensorPanel(data.sensor_type, data);
            this.updateRefreshTime();
        }
    },

    /**
     * 处理设备状态变化
     * @param {Object} data - 设备状态数据
     */
    handleDeviceStatus(data) {
        console.log('[RealtimeMonitor] 收到设备状态更新:', data);
        // 设备状态变化可以触发统计信息更新
        if (typeof DeviceManager !== 'undefined' && DeviceManager.loadStatistics) {
            DeviceManager.loadStatistics();
        }
    },

    /**
     * 处理连接状态变化
     * @param {Object} data - 连接状态
     */
    handleConnectionChange(data) {
        console.log('[RealtimeMonitor] 连接状态变化:', data.status);
        
        if (data.status === 'connected') {
            // 重新连接后加载最新数据
            this.loadInitialData();
        }
    },

    /**
     * 加载初始数据(通过HTTP获取最后一次数据)
     */
    async loadInitialData() {
        console.log('[RealtimeMonitor] 加载初始数据...');
        
        const sensorTypes = Object.keys(this.sensorTypes);
        
        // 串行请求,避免触发后端限流
        for (let i = 0; i < sensorTypes.length; i++) {
            await this.fetchLatestData(sensorTypes[i]);
            
            // 每个请求之间延迟100ms
            if (i < sensorTypes.length - 1) {
                await new Promise(resolve => setTimeout(resolve, 100));
            }
        }
        
        this.updateRefreshTime();
        console.log('[RealtimeMonitor] 初始数据加载完成');
    },

    /**
     * 获取指定传感器类型的最新数据
     * @param {string} sensorType - 传感器类型
     */
    async fetchLatestData(sensorType) {
        try {
            const response = await API.queryData({
                sensor_type: sensorType,
                limit: 1,
                page: 1
            });

            if (response && response.data && response.data.length > 0) {
                const latestData = response.data[0];
                this.updateSensorPanel(sensorType, latestData);
            } else {
                this.updateSensorPanel(sensorType, null);
            }
        } catch (error) {
            console.error(`[RealtimeMonitor] 获取${sensorType}数据失败:`, error);
            this.updateSensorPanel(sensorType, null);
        }
    },

    /**
     * 更新页面中的阈值显示
     */
    updateThresholdDisplays() {
        if (typeof ThresholdManager === 'undefined' || !ThresholdManager.loaded) {
            console.warn('[RealtimeMonitor] 阈值未加载,使用默认值');
            return;
        }

        const thresholdIds = {
            'co2': 'co2ThresholdText',
            'co': 'coThresholdText',
            'smoke': 'smokeThresholdText',
            'liquid_level': 'liquidLevelThresholdText',
            'conductivity': 'conductivityThresholdText',
            'temperature': 'temperatureThresholdText',
            'flow': 'flowThresholdText'
        };

        Object.keys(thresholdIds).forEach(sensorType => {
            const el = document.getElementById(thresholdIds[sensorType]);
            if (el) {
                el.textContent = ThresholdManager.getThresholdText(sensorType);
            }
        });

        console.log('[RealtimeMonitor] 阈值显示已更新');
    },

    /**
     * 更新传感器面板
     * @param {string} sensorType - 传感器类型
     * @param {Object|null} data - 传感器数据
     */
    updateSensorPanel(sensorType, data) {
        const config = this.sensorTypes[sensorType];
        const panelId = `sensor${this.capitalize(this.camelCase(sensorType))}`;
        const panel = document.getElementById(panelId);

        if (!panel) {
            console.warn(`[RealtimeMonitor] 面板未找到: ${panelId}`);
            return;
        }

        const statusEl = panel.querySelector('.sensor-status');
        const valueEl = panel.querySelector('.reading-value');
        const timeEl = document.getElementById(config.timeId);

        // 如果有新数据,缓存到lastSensorData
        if (data) {
            this.lastSensorData[sensorType] = data;
        }

        // 优先使用新数据,如果没有新数据则使用缓存的最后数据
        const displayData = data || this.lastSensorData[sensorType];

        if (displayData) {
            // 有数据 - 更新显示
            const value = parseFloat(displayData.value).toFixed(1);
            valueEl.textContent = value;

            // 检查数据时效性
            const dataTime = new Date(displayData.timestamp);
            const now = new Date();
            const diffMinutes = (now - dataTime) / (1000 * 60);
            const isDataStale = diffMinutes > 60;

            // 检查是否超过阈值
            const isAlert = this.checkThreshold(sensorType, parseFloat(displayData.value));

            if (isAlert) {
                statusEl.textContent = '告警';
                statusEl.className = 'sensor-status alert';
                panel.classList.add('alert-state');
                panel.style.animation = 'pulse 2s infinite';
            } else if (isDataStale) {
                statusEl.textContent = '过期';
                statusEl.className = 'sensor-status offline';
                panel.classList.remove('alert-state');
                panel.style.animation = '';
            } else {
                statusEl.textContent = '正常';
                statusEl.className = 'sensor-status online';
                panel.classList.remove('alert-state');
                panel.style.animation = '';
            }

            // 更新时间
            if (timeEl) {
                timeEl.textContent = this.formatTime(displayData.timestamp);
            }

            // 如果是新数据,添加数据更新动画效果
            if (data) {
                valueEl.style.transform = 'scale(1.1)';
                setTimeout(() => {
                    valueEl.style.transform = 'scale(1)';
                }, 200);
            }

        } else {
            // 完全无数据 - 显示离线
            valueEl.textContent = '--';
            statusEl.textContent = '离线';
            statusEl.className = 'sensor-status offline';
            panel.classList.remove('alert-state');
            panel.style.animation = '';

            if (timeEl) {
                timeEl.textContent = '-';
            }
        }
    },

    /**
     * 检查是否超过阈值
     * @param {string} sensorType - 传感器类型
     * @param {number} value - 传感器值
     * @returns {boolean} 是否超过阈值
     */
    checkThreshold(sensorType, value) {
        // 优先使用ThresholdManager
        if (typeof ThresholdManager !== 'undefined' && ThresholdManager.loaded) {
            return ThresholdManager.isExceeded(sensorType, value);
        }

        // 兜底使用硬编码阈值
        const threshold = this.sensorTypes[sensorType].threshold;

        if (threshold.max !== undefined && value > threshold.max) {
            return true;
        }

        if (threshold.min !== undefined && value < threshold.min) {
            return true;
        }

        return false;
    },

    /**
     * 更新刷新时间显示
     */
    updateRefreshTime() {
        const timeEl = document.getElementById('lastRefreshTime');
        if (timeEl) {
            const now = new Date();
            timeEl.textContent = `最后更新: ${now.toLocaleTimeString('zh-CN')}`;
        }
    },

    /**
     * 格式化时间
     * @param {string} timestamp - 时间戳
     * @returns {string} 格式化的时间字符串
     */
    formatTime(timestamp) {
        if (!timestamp) return '-';

        const date = new Date(timestamp);
        const now = new Date();
        const diff = now - date;

        // 如果是今天，只显示时间
        if (diff < 24 * 60 * 60 * 1000 && date.getDate() === now.getDate()) {
            return date.toLocaleTimeString('zh-CN');
        }

        // 否则显示日期+时间
        return date.toLocaleString('zh-CN');
    },

    /**
     * 转换为驼峰命名
     * @param {string} str - 下划线命名字符串
     * @returns {string} 驼峰命名字符串
     */
    camelCase(str) {
        return str.replace(/_([a-z])/g, (match, letter) => letter.toUpperCase());
    },

    /**
     * 首字母大写
     * @param {string} str - 字符串
     * @returns {string} 首字母大写的字符串
     */
    capitalize(str) {
        return str.charAt(0).toUpperCase() + str.slice(1);
    },

    /**
     * 清理资源
     */
    destroy() {
        console.log('[RealtimeMonitor] 清理资源...');
        
        // 取消WebSocket订阅
        if (this.unsubscribeSensorData) {
            this.unsubscribeSensorData();
            this.unsubscribeSensorData = null;
        }
        
        if (this.unsubscribeDeviceStatus) {
            this.unsubscribeDeviceStatus();
            this.unsubscribeDeviceStatus = null;
        }
        
        if (this.unsubscribeConnection) {
            this.unsubscribeConnection();
            this.unsubscribeConnection = null;
        }
        
        console.log('[RealtimeMonitor] 资源清理完成');
    }
};

// 导出模块
if (typeof module !== 'undefined' && module.exports) {
    module.exports = RealtimeMonitor;
}
