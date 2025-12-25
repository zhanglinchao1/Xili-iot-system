/**
 * 设备管理模块
 * 处理设备相关的业务逻辑
 * 使用WebSocket接收设备状态实时更新
 * @version 3.0.0
 */

const DeviceManager = {
    currentPage: 1,
    pageSize: 20,
    currentFilters: {},
    devices: [],
    statistics: null,
    version: '3.0.0',
    
    // WebSocket订阅取消函数
    unsubscribeDeviceStatus: null,
    unsubscribeConnection: null,

    /**
     * 初始化设备管理模块
     */
    init() {
        console.log('%c[DeviceManager] 初始化', 'color: #8b5cf6; font-weight: bold');
        console.log('[DeviceManager] 版本:', this.version);

        try {
            console.log('[DeviceManager] 绑定事件处理器...');
            this.bindEvents();
            console.log('[DeviceManager] ✓ 事件绑定完成');

            // 订阅WebSocket消息
            console.log('[DeviceManager] 订阅WebSocket消息...');
            this.subscribeWebSocket();
            console.log('[DeviceManager] ✓ WebSocket订阅完成');

            console.log('[DeviceManager] 加载设备列表...');
            this.loadDevices();
        } catch (error) {
            console.error('[DeviceManager] ✗ 初始化失败:', error);
            throw error;
        }
    },

    /**
     * 订阅WebSocket消息
     */
    subscribeWebSocket() {
        if (typeof WebSocketManager === 'undefined') {
            console.warn('[DeviceManager] WebSocketManager未定义，跳过WebSocket订阅');
            return;
        }

        // 订阅设备状态变化
        this.unsubscribeDeviceStatus = WebSocketManager.subscribe('device_status', (data) => {
            this.handleDeviceStatusUpdate(data);
        });

        // 订阅连接状态变化
        this.unsubscribeConnection = WebSocketManager.subscribe('connection', (data) => {
            if (data.status === 'connected') {
                // 重新连接后刷新数据
                console.log('[DeviceManager] WebSocket重连，刷新设备数据');
                this.loadStatistics();
            }
        });
    },

    /**
     * 处理设备状态更新
     * @param {Object} data - 设备状态数据
     */
    handleDeviceStatusUpdate(data) {
        console.log('[DeviceManager] 收到设备状态更新:', data);
        
        if (!data || !data.device_id) return;

        // 更新内存中的设备列表
        const deviceIndex = this.devices.findIndex(d => d.device_id === data.device_id);
        if (deviceIndex > -1) {
            // 更新设备状态
            this.devices[deviceIndex].status = data.status;
            this.devices[deviceIndex].last_seen_at = data.timestamp || new Date().toISOString();
            
            // 更新表格中的对应行
            this.updateDeviceRow(data.device_id, data.status);
        }

        // 更新统计信息
        this.loadStatistics();
    },

    /**
     * 更新表格中的设备行
     * @param {string} deviceId - 设备ID
     * @param {string} status - 新状态
     */
    updateDeviceRow(deviceId, status) {
        const row = document.querySelector(`tr[data-device-id="${deviceId}"]`);
        if (!row) return;

        // 更新状态徽章
        const statusCell = row.querySelector('td:nth-child(4)');
        if (statusCell && typeof UI !== 'undefined') {
            statusCell.innerHTML = UI.getStatusBadge(status);
            
            // 添加更新动画
            statusCell.style.transition = 'background-color 0.3s';
            statusCell.style.backgroundColor = 'rgba(59, 130, 246, 0.1)';
            setTimeout(() => {
                statusCell.style.backgroundColor = '';
            }, 1000);
        }

        // 更新最后在线时间
        const lastSeenCell = row.querySelector('td:nth-child(7)');
        if (lastSeenCell && typeof UI !== 'undefined') {
            lastSeenCell.textContent = UI.formatDateTime(new Date().toISOString());
        }

        console.log(`[DeviceManager] 已更新设备 ${deviceId} 的显示状态为 ${status}`);
    },

    /**
     * 绑定事件
     */
    bindEvents() {
        // 添加设备按钮
        document.getElementById('addDeviceBtn').addEventListener('click', () => {
            this.showDeviceModal();
        });

        // 设备表单提交
        document.getElementById('deviceForm').addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveDevice();
        });

        // 模态框关闭
        document.getElementById('modalClose').addEventListener('click', () => {
            UI.hideModal('deviceModal');
        });
        document.getElementById('cancelBtn').addEventListener('click', () => {
            UI.hideModal('deviceModal');
        });

        // 详情模态框关闭
        document.getElementById('detailModalClose').addEventListener('click', () => {
            UI.hideModal('deviceDetailModal');
        });
        document.getElementById('detailCloseBtn').addEventListener('click', () => {
            UI.hideModal('deviceDetailModal');
        });

        // 过滤器变化
        document.getElementById('statusFilter').addEventListener('change', () => {
            this.currentFilters.status = document.getElementById('statusFilter').value;
            this.currentPage = 1;
            this.loadDevices();
        });

        document.getElementById('sensorTypeFilter').addEventListener('change', () => {
            this.currentFilters.sensor_type = document.getElementById('sensorTypeFilter').value;
            this.currentPage = 1;
            this.loadDevices();
        });

        // 搜索（使用防抖）
        document.getElementById('deviceSearch').addEventListener('input', 
            UI.debounce(() => {
                this.currentFilters.search = document.getElementById('deviceSearch').value;
                this.currentPage = 1;
                this.loadDevices();
            }, 500)
        );
    },

    /**
     * 加载设备列表
     */
    async loadDevices(retryCount = 0) {
        const maxRetries = 2;

        console.log('[DeviceManager] loadDevices() 开始');
        console.log('[DeviceManager] 当前页:', this.currentPage);
        console.log('[DeviceManager] 每页数量:', this.pageSize);
        console.log('[DeviceManager] 过滤条件:', this.currentFilters);
        if (retryCount > 0) {
            console.log(`[DeviceManager] 重试次数: ${retryCount}/${maxRetries}`);
        }

        try {
            console.log('[DeviceManager] 显示加载中...');
            UI.showLoading();

            const params = {
                page: this.currentPage,
                limit: this.pageSize,
                ...this.currentFilters
            };

            console.log('[DeviceManager] 调用 API.getDevices(), 参数:', params);
            const response = await API.getDevices(params);
            console.log('[DeviceManager] API 响应:', response);

            this.devices = response.devices || [];
            console.log('[DeviceManager] 获取到设备数量:', this.devices.length);

            console.log('[DeviceManager] 渲染设备表格...');
            this.renderDeviceTable();

            // 计算总页数
            const totalPages = Math.ceil((response.total || 0) / this.pageSize);
            console.log('[DeviceManager] 总页数:', totalPages);

            UI.renderPagination(this.currentPage, totalPages, (page) => {
                this.currentPage = page;
                this.loadDevices();
            });

            console.log('[DeviceManager] ✓ 设备列表加载完成');

        } catch (error) {
            console.error('[DeviceManager] ✗ 加载设备列表失败:', error);
            console.error('[DeviceManager] 错误堆栈:', error.stack);

            // 如果是限流错误且未超过重试次数,延迟后重试
            if (error.message && error.message.includes('请求过于频繁') && retryCount < maxRetries) {
                const retryDelay = 2000 * (retryCount + 1); // 递增延迟: 2秒, 4秒
                console.log(`[DeviceManager] 将在${retryDelay/1000}秒后重试 (${retryCount + 1}/${maxRetries})`);
                UI.showToast(`请求频繁,${retryDelay/1000}秒后自动重试...`, 'warning');

                await new Promise(resolve => setTimeout(resolve, retryDelay));
                return this.loadDevices(retryCount + 1);
            }

            UI.showToast('加载设备列表失败: ' + error.message, 'error');
        } finally {
            console.log('[DeviceManager] 隐藏加载中...');
            UI.hideLoading();
        }
    },

    /**
     * 渲染设备表格
     */
    async renderDeviceTable() {
        const tbody = document.getElementById('deviceTableBody');

        if (!this.devices || this.devices.length === 0) {
            UI.renderEmptyState(tbody, '暂无设备数据');
            return;
        }

        // 如果有搜索关键词，进行客户端过滤
        let filteredDevices = this.devices;
        if (this.currentFilters.search) {
            const searchTerm = this.currentFilters.search.toLowerCase();
            filteredDevices = this.devices.filter(device =>
                device.device_id.toLowerCase().includes(searchTerm) ||
                (device.model && device.model.toLowerCase().includes(searchTerm))
            );
        }

        if (filteredDevices.length === 0) {
            UI.renderEmptyState(tbody, '未找到匹配的设备');
            return;
        }

        // 先渲染基本设备信息，不等待最新数据
        // 这样可以避免API失败导致页面无法加载
        const devicesWithData = filteredDevices.map(device => ({
            ...device,
            latestData: null,
            sensorUnit: UI.getSensorUnit(device.sensor_type)
        }));

        tbody.innerHTML = devicesWithData.map(device => `
            <tr data-device-id="${device.device_id}">
                <td><strong>${device.device_id}</strong></td>
                <td>${UI.getSensorTypeName(device.sensor_type)}</td>
                <td>${device.cabinet_id}</td>
                <td>${UI.getStatusBadge(device.status)}</td>
                <td>${device.model || '-'}</td>
                <td class="device-latest-data-${device.device_id.replace(/[^a-zA-Z0-9]/g, '_')}">${this.renderLatestData(device.latestData, device.sensorUnit)}</td>
                <td>${UI.formatDateTime(device.last_seen_at)}</td>
                <td>
                    <button class="btn btn-small btn-ghost" onclick="DeviceManager.viewDevice('${device.device_id}')">
                        <i class="fas fa-eye"></i> 查看
                    </button>
                    <button class="btn btn-small btn-ghost" onclick="DeviceManager.editDevice('${device.device_id}')">
                        <i class="fas fa-edit"></i> 编辑
                    </button>
                    <button class="btn btn-small btn-ghost" onclick="DeviceManager.deleteDevice('${device.device_id}')">
                        <i class="fas fa-trash"></i> 删除
                    </button>
                </td>
            </tr>
        `).join('');

        // 异步加载最新数据，不阻塞渲染
        this.loadLatestDataForDevices(filteredDevices);
    },

    /**
     * 异步加载设备最新数据
     */
    async loadLatestDataForDevices(devices) {
        devices.forEach(async (device) => {
            try {
                const latestData = await API.getDeviceLatestData(device.device_id);
                const safeId = device.device_id.replace(/[^a-zA-Z0-9]/g, '_');
                const cell = document.querySelector(`.device-latest-data-${safeId}`);
                if (cell) {
                    cell.innerHTML = this.renderLatestData(latestData, UI.getSensorUnit(device.sensor_type));
                }
            } catch (error) {
                console.warn(`获取设备 ${device.device_id} 最新数据失败:`, error);
                // 保持显示"暂无数据"，不影响其他设备
            }
        });
    },

    /**
     * 渲染最新传感器数据
     */
    renderLatestData(latestData, unit) {
        if (!latestData || latestData.value === null || latestData.value === undefined) {
            return '<span style="color: var(--gray-400); font-style: italic;">暂无数据</span>';
        }

        // 根据数值大小决定小数位数
        let formattedValue;
        if (latestData.value >= 1000) {
            formattedValue = latestData.value.toFixed(0);
        } else if (latestData.value >= 100) {
            formattedValue = latestData.value.toFixed(1);
        } else {
            formattedValue = latestData.value.toFixed(2);
        }

        // 根据数据质量显示不同颜色
        let qualityColor = 'var(--primary-600)';
        if (latestData.quality < 80) {
            qualityColor = 'var(--warning-600)';
        }
        if (latestData.quality < 60) {
            qualityColor = 'var(--danger-600)';
        }

        return `
            <div style="display: flex; flex-direction: column; align-items: flex-start;">
                <span style="font-weight: 600; color: ${qualityColor}; font-size: 1.1em;">
                    ${formattedValue} <span style="color: var(--gray-500); font-size: 0.85em; font-weight: normal;">${unit}</span>
                </span>
                <span style="color: var(--gray-400); font-size: 0.75em;">
                    ${UI.formatDateTime(latestData.timestamp)}
                </span>
            </div>
        `;
    },

    /**
     * 显示设备模态框（新建或编辑）
     */
    showDeviceModal(device = null) {
        const modal = document.getElementById('deviceModal');
        const title = document.getElementById('modalTitle');
        
        if (device) {
            title.textContent = '编辑设备';
            // 设置表单数据
            UI.setFormData('deviceForm', {
                device_id: device.device_id,
                cabinet_id: device.cabinet_id,
                sensor_type: device.sensor_type,
                status: device.status,
                model: device.model,
                manufacturer: device.manufacturer,
                firmware_ver: device.firmware_ver,
                public_key: device.public_key,
                commitment: device.commitment
            });
            // 设备ID不可编辑
            document.getElementById('deviceId').readOnly = true;
        } else {
            title.textContent = '注册新设备';
            UI.clearForm('deviceForm');
            document.getElementById('deviceId').readOnly = false;
        }
        
        UI.showModal('deviceModal');
    },

    /**
     * 保存设备（新建或更新）
     */
    async saveDevice() {
        try {
            UI.showLoading();
            
            const formData = UI.getFormData('deviceForm');
            const deviceId = formData.device_id;
            
            // 检查是新建还是更新
            const existingDevice = this.devices.find(d => d.device_id === deviceId);
            
            if (existingDevice) {
                // 更新设备
                await API.updateDevice(deviceId, formData);
                UI.showToast('设备信息更新成功', 'success');
            } else {
                // 注册新设备
                await API.registerDevice(formData);
                UI.showToast('设备注册成功', 'success');
            }
            
            UI.hideModal('deviceModal');
            this.loadDevices();
            
        } catch (error) {
            console.error('保存设备失败:', error);
            UI.showToast('保存设备失败: ' + error.message, 'error');
        } finally {
            UI.hideLoading();
        }
    },

    /**
     * 查看设备详情
     */
    async viewDevice(deviceId) {
        try {
            UI.showLoading();
            const device = await API.getDevice(deviceId);
            this.renderDeviceDetail(device);
            UI.showModal('deviceDetailModal');
        } catch (error) {
            console.error('获取设备详情失败:', error);
            UI.showToast('获取设备详情失败: ' + error.message, 'error');
        } finally {
            UI.hideLoading();
        }
    },

    /**
     * 渲染设备详情
     */
    renderDeviceDetail(device) {
        const content = document.getElementById('deviceDetailContent');
        content.innerHTML = `
            <div style="display: grid; gap: 20px;">
                <div class="card">
                    <div class="card-header">
                        <h3>基本信息</h3>
                    </div>
                    <div class="card-body">
                        <table style="width: 100%; border-collapse: collapse;">
                            <tr>
                                <td style="padding: 8px; font-weight: 600; width: 30%;">设备ID</td>
                                <td style="padding: 8px;">${device.device_id}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">设备类型</td>
                                <td style="padding: 8px;">${device.device_type}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">传感器类型</td>
                                <td style="padding: 8px;">${UI.getSensorTypeName(device.sensor_type)}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">状态</td>
                                <td style="padding: 8px;">${UI.getStatusBadge(device.status)}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">型号</td>
                                <td style="padding: 8px;">${device.model || '-'}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">制造商</td>
                                <td style="padding: 8px;">${device.manufacturer || '-'}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">固件版本</td>
                                <td style="padding: 8px;">${device.firmware_ver || '-'}</td>
                            </tr>
                        </table>
                    </div>
                </div>

                <div class="card">
                    <div class="card-header">
                        <h3>零知识证明信息</h3>
                    </div>
                    <div class="card-body">
                        <div style="margin-bottom: 16px;">
                            <strong>公钥:</strong>
                            <pre style="background: var(--gray-50); padding: 12px; border-radius: 4px; margin-top: 8px; overflow-x: auto; font-size: 12px;">${device.public_key}</pre>
                        </div>
                        <div>
                            <strong>承诺值:</strong>
                            <pre style="background: var(--gray-50); padding: 12px; border-radius: 4px; margin-top: 8px; overflow-x: auto; font-size: 12px;">${device.commitment}</pre>
                        </div>
                    </div>
                </div>

                <div class="card">
                    <div class="card-header">
                        <h3>时间信息</h3>
                    </div>
                    <div class="card-body">
                        <table style="width: 100%; border-collapse: collapse;">
                            <tr>
                                <td style="padding: 8px; font-weight: 600; width: 30%;">创建时间</td>
                                <td style="padding: 8px;">${UI.formatDateTime(device.created_at)}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">更新时间</td>
                                <td style="padding: 8px;">${UI.formatDateTime(device.updated_at)}</td>
                            </tr>
                            <tr>
                                <td style="padding: 8px; font-weight: 600;">最后在线</td>
                                <td style="padding: 8px;">${UI.formatDateTime(device.last_seen_at)}</td>
                            </tr>
                        </table>
                    </div>
                </div>
            </div>
        `;
    },

    /**
     * 编辑设备
     */
    async editDevice(deviceId) {
        try {
            UI.showLoading();
            const device = await API.getDevice(deviceId);
            this.showDeviceModal(device);
        } catch (error) {
            console.error('获取设备信息失败:', error);
            UI.showToast('获取设备信息失败: ' + error.message, 'error');
        } finally {
            UI.hideLoading();
        }
    },

    /**
     * 删除设备
     */
    async deleteDevice(deviceId) {
        const confirmed = await UI.confirm(`确定要删除设备 ${deviceId} 吗？此操作不可恢复。`);
        if (!confirmed) return;

        try {
            UI.showLoading();
            await API.deleteDevice(deviceId);
            UI.showToast('设备删除成功', 'success');
            this.loadDevices();
        } catch (error) {
            console.error('删除设备失败:', error);
            UI.showToast('删除设备失败: ' + error.message, 'error');
        } finally {
            UI.hideLoading();
        }
    },

    /**
     * 加载设备统计信息
     */
    async loadStatistics() {
        console.log('[DeviceManager] loadStatistics() 开始');

        try {
            console.log('[DeviceManager] 调用 API.getDeviceStatistics()...');
            const stats = await API.getDeviceStatistics();
            console.log('[DeviceManager] 统计数据:', stats);

            this.statistics = stats;
            console.log('[DeviceManager] 渲染统计信息...');
            this.renderStatistics();
            console.log('[DeviceManager] ✓ 统计信息加载完成');

        } catch (error) {
            console.error('[DeviceManager] ✗ 加载统计信息失败:', error);

            // 如果API失败，使用默认值
            this.statistics = {
                total: 0,
                online: 0,
                offline: 0,
                fault: 0,
                sensor_types: {}
            };
            console.log('[DeviceManager] 使用默认统计数据');
            this.renderStatistics();
        }
    },

    /**
     * 渲染统计信息
     */
    renderStatistics() {
        console.log('[DeviceManager] renderStatistics() 开始');

        if (!this.statistics) {
            console.warn('[DeviceManager] ⚠ 统计数据为空，跳过渲染');
            return;
        }

        console.log('[DeviceManager] 更新统计卡片...');

        try {
            // 更新统计卡片
            const statElements = {
                'statTotal': this.statistics.total || 0,
                'statOnline': this.statistics.online || 0,
                'statOffline': this.statistics.offline || 0,
                'statFault': this.statistics.fault || 0
            };

            for (const [id, value] of Object.entries(statElements)) {
                const element = document.getElementById(id);
                if (element) {
                    // 检查值是否变化,添加动画效果
                    const oldValue = parseInt(element.textContent) || 0;
                    if (oldValue !== value) {
                        element.style.transition = 'transform 0.3s, color 0.3s';
                        element.style.transform = 'scale(1.2)';
                        element.style.color = 'var(--primary-color)';
                        setTimeout(() => {
                            element.style.transform = 'scale(1)';
                            element.style.color = '';
                        }, 300);
                    }
                    element.textContent = value;
                    console.log(`[DeviceManager] ${id} = ${value}`);
                } else {
                    console.warn(`[DeviceManager] ⚠ 找不到元素: #${id}`);
                }
            }

            // 渲染传感器类型分布（可选元素，不存在时跳过）
            console.log('[DeviceManager] 渲染传感器类型分布...');
            const sensorTypesGrid = document.getElementById('sensorTypesGrid');

            if (!sensorTypesGrid) {
                console.log('[DeviceManager] 传感器类型分布容器不存在，跳过渲染');
                return;
            }

            const sensorTypes = this.statistics.sensor_types || {};
            console.log('[DeviceManager] 传感器类型数据:', sensorTypes);

            if (Object.keys(sensorTypes).length === 0) {
                sensorTypesGrid.innerHTML = '<p style="color: var(--gray-400);">暂无传感器数据</p>';
                console.log('[DeviceManager] 无传感器数据');
                return;
            }

            sensorTypesGrid.innerHTML = Object.entries(sensorTypes).map(([type, count]) => `
                <div class="sensor-type-item">
                    <div class="sensor-type-name">${UI.getSensorTypeName(type)}</div>
                    <div class="sensor-type-count">${count}</div>
                </div>
            `).join('');

            console.log('[DeviceManager] ✓ 统计信息渲染完成');

        } catch (error) {
            console.error('[DeviceManager] ✗ 渲染统计信息失败:', error);
            console.error('[DeviceManager] 错误堆栈:', error.stack);
        }
    },

    /**
     * 清理资源
     */
    destroy() {
        console.log('[DeviceManager] 清理资源...');
        
        // 取消WebSocket订阅
        if (this.unsubscribeDeviceStatus) {
            this.unsubscribeDeviceStatus();
            this.unsubscribeDeviceStatus = null;
        }
        
        if (this.unsubscribeConnection) {
            this.unsubscribeConnection();
            this.unsubscribeConnection = null;
        }
        
        console.log('[DeviceManager] 资源清理完成');
    }
};

// 导出DeviceManager对象
window.DeviceManager = DeviceManager;

