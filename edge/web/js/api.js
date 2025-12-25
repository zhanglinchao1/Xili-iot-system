/**
 * API调用模块
 * 封装所有后端API接口调用
 * @version 2.1.0 - 支持内网穿透访问
 */

const API = {
    // 自动检测访问地址，支持内网穿透
    baseURL: (() => {
        const hostname = window.location.hostname;
        const port = window.location.port;
        const protocol = window.location.protocol;
        
        // 本地开发环境
        if (hostname === 'localhost' || hostname === '127.0.0.1') {
            return 'http://localhost:8001';
        }
        
        // IDE端口转发环境
        if (port === '63443' || parseInt(port) > 60000) {
            return 'http://localhost:8001';
        }
        
        // 内网穿透或生产环境：使用相同的origin（不指定端口）
        // 内网穿透通常将后端8001端口映射到80端口
        if (!port || port === '80' || port === '443') {
            return `${protocol}//${hostname}`;
        }
        
        // 其他情况：带端口访问（如直接访问IP:8001）
        return `${protocol}//${hostname}:${port}`;
    })(),
    version: '2.1.0',
    requestCount: 0,

    /**
     * 获取请求头
     */
    getHeaders() {
        return {
            'Content-Type': 'application/json'
        };
    },

    /**
     * 通用请求方法
     */
    async request(url, options = {}) {
        const requestId = ++this.requestCount;
        const fullURL = `${this.baseURL}${url}`;

        console.log(`%c[API #${requestId}] 请求`, 'color: #f59e0b; font-weight: bold');
        console.log(`[API #${requestId}] URL: ${fullURL}`);
        console.log(`[API #${requestId}] Method: ${options.method || 'GET'}`);
        console.log(`[API #${requestId}] Headers:`, this.getHeaders());
        if (options.body) {
            console.log(`[API #${requestId}] Body:`, options.body);
        }

        try {
            const startTime = performance.now();

            const response = await fetch(fullURL, {
                ...options,
                headers: {
                    ...this.getHeaders(),
                    ...options.headers
                }
            });

            const endTime = performance.now();
            const duration = (endTime - startTime).toFixed(2);

            console.log(`[API #${requestId}] 响应状态: ${response.status} ${response.statusText}`);
            console.log(`[API #${requestId}] 响应时间: ${duration}ms`);

            const data = await response.json();
            console.log(`[API #${requestId}] 响应数据:`, data);

            if (!response.ok) {
                const errorMsg = data.message || `HTTP ${response.status}: ${response.statusText}`;
                console.error(`[API #${requestId}] ✗ 请求失败:`, errorMsg);
                throw new Error(errorMsg);
            }

            console.log(`[API #${requestId}] ✓ 请求成功`);
            return data;
        } catch (error) {
            console.error(`[API #${requestId}] ✗ 请求异常:`, error);
            console.error(`[API #${requestId}] 错误类型:`, error.name);
            console.error(`[API #${requestId}] 错误信息:`, error.message);

            if (error.name === 'TypeError' && error.message.includes('Failed to fetch')) {
                console.error(`[API #${requestId}] 可能原因: 1) 后端服务未启动 2) CORS配置问题 3) 网络连接问题`);
            }

            throw error;
        }
    },

    /**
     * 健康检查
     */
    async healthCheck() {
        return this.request('/health');
    },

    /**
     * 获取设备列表
     */
    async getDevices(params = {}) {
        const queryParams = new URLSearchParams();
        if (params.page) queryParams.append('page', params.page);
        if (params.limit) queryParams.append('limit', params.limit);
        if (params.status) queryParams.append('status', params.status);
        if (params.sensor_type) queryParams.append('sensor_type', params.sensor_type);

        const queryString = queryParams.toString();
        const url = `/api/v1/devices${queryString ? '?' + queryString : ''}`;
        
        return this.request(url);
    },

    /**
     * 获取单个设备信息
     */
    async getDevice(deviceId) {
        return this.request(`/api/v1/devices/${deviceId}`);
    },

    /**
     * 获取设备最新传感器数据
     */
    async getDeviceLatestData(deviceId) {
        return this.request(`/api/v1/devices/${deviceId}/latest-data`);
    },

    /**
     * 注册新设备
     */
    async registerDevice(deviceData) {
        return this.request('/api/v1/devices', {
            method: 'POST',
            body: JSON.stringify({
                device_type: 'sensor',
                ...deviceData
            })
        });
    },

    /**
     * 更新设备信息
     */
    async updateDevice(deviceId, deviceData) {
        return this.request(`/api/v1/devices/${deviceId}`, {
            method: 'PUT',
            body: JSON.stringify(deviceData)
        });
    },

    /**
     * 注销设备
     */
    async deleteDevice(deviceId) {
        return this.request(`/api/v1/devices/${deviceId}`, {
            method: 'DELETE'
        });
    },

    /**
     * 设备心跳
     */
    async sendHeartbeat(deviceId, data = {}) {
        return this.request(`/api/v1/devices/${deviceId}/heartbeat`, {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    /**
     * 获取设备统计信息
     */
    async getDeviceStatistics() {
        return this.request('/api/v1/devices/statistics');
    },

    /**
     * 生成认证挑战（用于测试）
     */
    async getChallenge(deviceId) {
        return this.request('/api/v1/auth/challenge', {
            method: 'POST',
            body: JSON.stringify({ device_id: deviceId })
        });
    },

    /**
     * 验证证明（用于测试）
     */
    async verifyProof(data) {
        return this.request('/api/v1/auth/verify', {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    /**
     * 获取统计数据
     * @param {string} deviceId - 设备ID（可选）
     * @param {string} sensorType - 传感器类型（可选）
     * @param {string} period - 统计周期 (1h/24h/7d/30d)
     */
    async getStatistics(deviceId = '', sensorType = '', period = '24h') {
        const queryParams = new URLSearchParams();
        if (deviceId) queryParams.append('device_id', deviceId);
        if (sensorType) queryParams.append('sensor_type', sensorType);
        if (period) queryParams.append('period', period);

        const queryString = queryParams.toString();
        const url = `/api/v1/data/statistics${queryString ? '?' + queryString : ''}`;

        return this.request(url);
    },

    /**
     * 获取告警列表
     * @param {number} page - 页码
     * @param {number} limit - 每页数量
     * @param {string} severity - 严重程度筛选（可选）
     * @param {string} resolved - 是否已解决（可选）
     */
    async getAlerts(page = 1, limit = 20, severity = '', resolved = '') {
        const queryParams = new URLSearchParams();
        queryParams.append('page', page);
        queryParams.append('limit', limit);
        if (severity) queryParams.append('severity', severity);
        if (resolved) queryParams.append('resolved', resolved);

        const queryString = queryParams.toString();
        const url = `/api/v1/alerts?${queryString}`;

        return this.request(url);
    },

    /**
     * 解决告警
     * @param {number} alertId - 告警ID
     */
    async resolveAlert(alertId) {
        return this.request(`/api/v1/alerts/${alertId}/resolve`, {
            method: 'PUT'
        });
    },

    /**
     * 查询历史数据
     * @param {Object|string} paramsOrDeviceId - 参数对象或设备ID（向后兼容）
     * @param {string} sensorType - 传感器类型（可选，旧API）
     * @param {string} startTime - 开始时间（可选，旧API）
     * @param {string} endTime - 结束时间（可选，旧API）
     * @param {number} page - 页码（旧API）
     * @param {number} limit - 每页数量（旧API）
     */
    async queryData(paramsOrDeviceId, sensorType = '', startTime = '', endTime = '', page = 1, limit = 100) {
        const queryParams = new URLSearchParams();

        // 支持对象参数或传统位置参数
        if (typeof paramsOrDeviceId === 'object') {
            // 新API: 使用对象参数
            const params = paramsOrDeviceId;
            if (params.device_id) queryParams.append('device_id', params.device_id);
            if (params.sensor_type) queryParams.append('sensor_type', params.sensor_type);
            if (params.start_time) queryParams.append('start_time', params.start_time);
            if (params.end_time) queryParams.append('end_time', params.end_time);
            queryParams.append('page', params.page || 1);
            queryParams.append('limit', params.limit || 100);
        } else {
            // 旧API: 使用位置参数（向后兼容）
            queryParams.append('device_id', paramsOrDeviceId);
            if (sensorType) queryParams.append('sensor_type', sensorType);
            if (startTime) queryParams.append('start_time', startTime);
            if (endTime) queryParams.append('end_time', endTime);
            queryParams.append('page', page);
            queryParams.append('limit', limit);
        }

        const queryString = queryParams.toString();
        const url = `/api/v1/data/query?${queryString}`;

        return this.request(url);
    },

    /**
     * 获取告警日志
     * @param {Object} filters - 筛选条件
     * @returns {Promise<Object>} 日志数据
     */
    async getAlertLogs(filters = {}) {
        const queryParams = new URLSearchParams();

        if (filters.startDate) queryParams.append('start_date', filters.startDate);
        if (filters.endDate) queryParams.append('end_date', filters.endDate);
        if (filters.severity) queryParams.append('severity', filters.severity);
        if (filters.resolved !== undefined && filters.resolved !== '') {
            queryParams.append('resolved', filters.resolved);
        }
        if (filters.deviceID) queryParams.append('device_id', filters.deviceID);
        queryParams.append('page', filters.page || 1);
        queryParams.append('limit', filters.limit || 20);

        const queryString = queryParams.toString();
        const url = `/api/v1/logs/alerts?${queryString}`;

        return this.request(url);
    },

    /**
     * 获取认证日志
     * @param {Object} filters - 筛选条件
     * @returns {Promise<Object>} 日志数据
     */
    async getAuthLogs(filters = {}) {
        const queryParams = new URLSearchParams();

        if (filters.startDate) queryParams.append('start_date', filters.startDate);
        if (filters.endDate) queryParams.append('end_date', filters.endDate);
        if (filters.status) queryParams.append('status', filters.status);
        if (filters.deviceID) queryParams.append('device_id', filters.deviceID);
        queryParams.append('page', filters.page || 1);
        queryParams.append('limit', filters.limit || 20);

        const queryString = queryParams.toString();
        const url = `/api/v1/logs/auth?${queryString}`;

        return this.request(url);
    },

    /**
     * 批量删除告警日志
     * @param {Array<number>} ids - 要删除的日志ID数组
     */
    async batchDeleteAlertLogs(ids) {
        return this.request('/api/v1/logs/alerts/batch', {
            method: 'DELETE',
            body: JSON.stringify({ ids })
        });
    },

    /**
     * 批量删除认证日志
     * @param {Array<string>} ids - 日志ID数组
     */
    async batchDeleteAuthLogs(ids) {
        return this.request('/api/v1/logs/auth/batch', {
            method: 'DELETE',
            body: JSON.stringify({ ids })
        });
    },

    /**
     * 一键清空所有认证日志
     */
    async clearAllAuthLogs() {
        return this.request('/api/v1/logs/auth/clear', {
            method: 'DELETE'
        });
    },

    /**
     * 获取许可证信息
     */
    async getLicenseInfo() {
        return this.request('/api/v1/license/info');
    },

    /**
     * 获取告警配置(阈值)
     */
    async getAlertConfig() {
        return this.request('/api/v1/alerts/config');
    },

    /**
     * 通用GET请求方法
     * @param {string} url - API路径（会自动添加/api/v1前缀）
     */
    async get(url) {
        // 如果URL已经包含/api/v1，直接使用；否则自动添加前缀
        const fullUrl = url.startsWith('/api/v1') ? url : `/api/v1${url}`;
        return this.request(fullUrl);
    },

    /**
     * 通用POST请求方法
     * @param {string} url - API路径
     * @param {Object} data - 请求数据
     */
    async post(url, data = {}) {
        const fullUrl = url.startsWith('/api/v1') ? url : `/api/v1${url}`;
        return this.request(fullUrl, {
            method: 'POST',
            body: JSON.stringify(data)
        });
    },

    /**
     * 通用PUT请求方法
     * @param {string} url - API路径
     * @param {Object} data - 请求数据
     */
    async put(url, data = {}) {
        const fullUrl = url.startsWith('/api/v1') ? url : `/api/v1${url}`;
        return this.request(fullUrl, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
    },

    /**
     * 通用DELETE请求方法
     * @param {string} url - API路径
     */
    async delete(url) {
        const fullUrl = url.startsWith('/api/v1') ? url : `/api/v1${url}`;
        return this.request(fullUrl, {
            method: 'DELETE'
        });
    },

    /**
     * 获取服务器时间
     * @returns {Promise<Date>} 服务器时间
     */
    async getServerTime() {
        try {
            const response = await this.request('/health');
            if (response && response.timestamp) {
                // timestamp 是Unix时间戳(秒),转换为毫秒
                const serverTime = new Date(response.timestamp * 1000);
                console.log(`[API] 服务器时间: ${serverTime.toISOString()} (timestamp: ${response.timestamp})`);
                return serverTime;
            }
            // 如果获取失败,fallback到浏览器时间
            console.warn('[API] 获取服务器时间失败,使用浏览器时间');
            return new Date();
        } catch (error) {
            console.error('[API] 获取服务器时间异常:', error);
            return new Date();
        }
    }
};

// 导出API对象
window.API = API;

