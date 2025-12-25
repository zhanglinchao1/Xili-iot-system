/**
 * Cabinet.js - 储能柜管理模块
 * Version: 1.0.5
 * 功能：储能柜信息管理、一键注册到Cloud端
 * 支持内网穿透访问
 */

// API基础URL - 动态检测访问环境
const API_BASE_URL = (() => {
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
    if (!port || port === '80' || port === '443') {
        return `${protocol}//${hostname}`;
    }
    
    // 其他情况：带端口访问
    return `${protocol}//${hostname}:${port}`;
})();

const Cabinet = {
    // 初始化标志，防止重复初始化
    initialized: false,
    
    // 当前储能柜信息
    cabinetInfo: {
        cabinet_id: '',
        name: '',
        location: '',
        latitude: null,
        longitude: null,
        capacity_kwh: 0,
        device_model: '',
        ip_address: '',
        mac_address: '',
        status: 'unknown',
        registered_to_cloud: false
    },

    // Cloud端配置
    // 从后端API动态获取，不使用硬编码默认值
    cloudConfig: {
        enabled: false,  // 默认禁用，等待API加载
        endpoint: '',    // 空值，等待API加载
        api_key: '',
        admin_token: '',
        cabinet_id: ''   // 空值，等待API加载
    },

    /**
     * 初始化储能柜管理模块
     */
    init() {
        console.log('[Cabinet] 初始化储能柜管理模块');

        // 🔥 防止重复初始化 - 事件监听器只绑定一次
        if (!this.initialized) {
            console.log('[Cabinet] 首次初始化，绑定事件监听器');
            this.bindEventListeners();
            this.initialized = true;
        } else {
            console.log('[Cabinet] 已初始化，跳过事件绑定');
        }

        // 每次切换页面时都重新加载数据
        this.loadCabinetInfo();
        this.loadCloudConfig();
    },

    /**
     * 绑定事件监听器
     */
    bindEventListeners() {
        // 保存按钮
        const saveBtn = document.getElementById('saveCabinetBtn');
        if (saveBtn) {
            saveBtn.addEventListener('click', () => this.saveCabinetInfo());
        }

        // 位置搜索按钮
        const searchLocationBtn = document.getElementById('searchLocationBtn');
        if (searchLocationBtn) {
            searchLocationBtn.addEventListener('click', () => this.searchLocation());
        }

        // 位置搜索输入框 - 实时搜索建议
        const locationInput = document.getElementById('cabinetLocation');
        if (locationInput) {
            let searchTimeout = null;
            locationInput.addEventListener('input', (e) => {
                const keyword = e.target.value.trim();
                
                // 清除之前的定时器
                if (searchTimeout) {
                    clearTimeout(searchTimeout);
                }

                // 如果输入为空，隐藏建议
                if (!keyword) {
                    this.hideLocationSuggestions();
                    return;
                }

                // 延迟搜索，避免频繁请求
                searchTimeout = setTimeout(() => {
                    this.searchPlaceSuggestions(keyword);
                }, 300);
            });

            // 点击外部时隐藏建议
            document.addEventListener('click', (e) => {
                const suggestions = document.getElementById('locationSuggestions');
                if (suggestions && !suggestions.contains(e.target) && e.target !== locationInput) {
                    this.hideLocationSuggestions();
                }
            });
        }

        // 注册到Cloud按钮
        const registerBtn = document.getElementById('registerToCloudBtn');
        if (registerBtn) {
            registerBtn.addEventListener('click', () => this.registerToCloud());
        }

        // 测试连接按钮
        const testBtn = document.getElementById('testCloudConnectionBtn');
        if (testBtn) {
            testBtn.addEventListener('click', () => this.testCloudConnection());
        }

        // 编辑Cloud配置按钮
        const editBtn = document.getElementById('editCloudConfigBtn');
        if (editBtn) {
            editBtn.addEventListener('click', () => this.showCloudConfigEdit());
        }

        // 取消编辑按钮
        const cancelBtn = document.getElementById('cancelEditCloudConfigBtn');
        if (cancelBtn) {
            cancelBtn.addEventListener('click', () => this.hideCloudConfigEdit());
        }

        // Cloud配置表单提交
        const configForm = document.getElementById('cloudConfigForm');
        if (configForm) {
            configForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.saveCloudConfig();
            });
        }
    },

    /**
     * 加载储能柜信息
     */
    async loadCabinetInfo() {
        try {
            console.log('[Cabinet] 开始加载储能柜信息...');

            // 从配置文件读取Cabinet ID和Cloud配置
            try {
                const configResponse = await fetch(`${API_BASE_URL}/api/v1/config`);
                console.log('[Cabinet] 配置API响应状态:', configResponse.status);

                if (configResponse.ok) {
                    const config = await configResponse.json();
                    console.log('[Cabinet] 配置数据:', config);

                    if (config && config.cloud) {
                        this.cabinetInfo.cabinet_id = config.cloud.cabinet_id || 'CABINET-001';

                        // 根据API Key判断注册状态
                        this.cabinetInfo.registered_to_cloud = config.cloud.enabled &&
                                                                config.cloud.api_key &&
                                                                config.cloud.api_key.length > 0;
                        console.log('[Cabinet] 注册状态:', this.cabinetInfo.registered_to_cloud);
                        
                        // 保存Cloud配置用于后续获取信息
                        this.cloudConfig.enabled = config.cloud.enabled || false;
                        this.cloudConfig.endpoint = config.cloud.endpoint || '';
                        this.cloudConfig.api_key = config.cloud.api_key || '';
                        
                        // 从配置中读取储能柜详细信息（后端存储）
                        if (config.cloud.cabinet_name) {
                            this.cabinetInfo.name = config.cloud.cabinet_name;
                        }
                        if (config.cloud.location) {
                            this.cabinetInfo.location = config.cloud.location;
                        }
                        if (config.cloud.latitude !== undefined && config.cloud.latitude !== null) {
                            this.cabinetInfo.latitude = config.cloud.latitude;
                        }
                        if (config.cloud.longitude !== undefined && config.cloud.longitude !== null) {
                            this.cabinetInfo.longitude = config.cloud.longitude;
                        }
                        if (config.cloud.capacity_kwh !== undefined && config.cloud.capacity_kwh !== null) {
                            this.cabinetInfo.capacity_kwh = config.cloud.capacity_kwh;
                        }
                        if (config.cloud.device_model) {
                            this.cabinetInfo.device_model = config.cloud.device_model;
                        }
                        console.log('[Cabinet] 从配置中读取的详细信息:', {
                            name: this.cabinetInfo.name,
                            location: this.cabinetInfo.location,
                            capacity_kwh: this.cabinetInfo.capacity_kwh,
                            device_model: this.cabinetInfo.device_model
                        });
                    }
                } else {
                    console.warn('[Cabinet] 配置API返回错误:', configResponse.status);
                    // 使用默认值
                    this.cabinetInfo.cabinet_id = 'CABINET-001';
                    this.cabinetInfo.registered_to_cloud = false;
                }
            } catch (err) {
                console.error('[Cabinet] 获取配置失败:', err);
                this.cabinetInfo.cabinet_id = 'CABINET-001';
                this.cabinetInfo.registered_to_cloud = false;
            }

            // 保存从API获取的注册状态,避免被localStorage覆盖
            const registeredStatus = this.cabinetInfo.registered_to_cloud;

            // 尝试从本地存储加载其他信息
            const saved = localStorage.getItem('cabinet_info');
            if (saved) {
                try {
                    const savedInfo = JSON.parse(saved);
                    // 合并数据，但保留从API获取的cabinet_id和registered_to_cloud
                    const savedCabinetId = savedInfo.cabinet_id;
                    const savedRegisteredStatus = savedInfo.registered_to_cloud;
                    this.cabinetInfo = { ...this.cabinetInfo, ...savedInfo };
                    // 恢复从API获取的值（优先级更高）
                    if (this.cabinetInfo.cabinet_id) {
                        // 如果API返回了cabinet_id，使用API的值
                    } else if (savedCabinetId) {
                        this.cabinetInfo.cabinet_id = savedCabinetId;
                    }
                    this.cabinetInfo.registered_to_cloud = registeredStatus;
                    console.log('[Cabinet] 从本地存储加载信息:', savedInfo);
                } catch (err) {
                    console.error('[Cabinet] 解析本地存储失败:', err);
                }
            }

            // 恢复从API获取的注册状态(优先级高于localStorage)
            this.cabinetInfo.registered_to_cloud = registeredStatus;
            
            console.log('[Cabinet] 合并后的储能柜信息:', JSON.stringify(this.cabinetInfo));

            // 获取MAC地址
            this.cabinetInfo.mac_address = await this.getMACAddress();

            // 获取IP地址
            this.cabinetInfo.ip_address = await this.getIPAddress();

            // 更新UI
            this.updateCabinetInfoUI();

            console.log('[Cabinet] 储能柜信息加载完成:', this.cabinetInfo);
        } catch (error) {
            console.error('[Cabinet] 加载储能柜信息失败:', error);
            UI.showToast('加载储能柜信息失败: ' + error.message, 'error');
        }
    },

    /**
     * 获取MAC地址
     */
    async getMACAddress() {
        try {
            console.log('[Cabinet] 正在获取MAC地址...');
            // 调用Edge API获取系统MAC地址
            const response = await fetch(`${API_BASE_URL}/api/v1/system/mac`);
            console.log('[Cabinet] MAC API响应状态:', response.status);

            if (response.ok) {
                const data = await response.json();
                console.log('[Cabinet] MAC数据:', data);
                return data.mac_address || '00:00:00:00:00:00';
            } else {
                console.warn('[Cabinet] MAC API返回错误:', response.status);
            }
        } catch (error) {
            console.error('[Cabinet] 获取MAC地址失败:', error);
        }
        return '00:00:00:00:00:00';
    },

    /**
     * 获取IP地址
     */
    async getIPAddress() {
        try {
            console.log('[Cabinet] 正在获取IP地址...');
            // 调用Edge API获取系统IP地址
            const response = await fetch(`${API_BASE_URL}/api/v1/system/ip`);
            console.log('[Cabinet] IP API响应状态:', response.status);

            if (response.ok) {
                const data = await response.json();
                console.log('[Cabinet] IP数据:', data);
                return data.ip_address || '0.0.0.0';
            } else {
                console.warn('[Cabinet] IP API返回错误:', response.status);
            }
        } catch (error) {
            console.error('[Cabinet] 获取IP地址失败:', error);
        }
        return '0.0.0.0';
    },

    /**
     * 更新储能柜信息UI
     */
    updateCabinetInfoUI() {
        console.log('[Cabinet] 更新UI，当前数据:', JSON.stringify(this.cabinetInfo));
        
        // Cabinet ID
        const cabinetIdInput = document.getElementById('cabinetId');
        if (cabinetIdInput) {
            cabinetIdInput.value = this.cabinetInfo.cabinet_id || '';
            console.log('[Cabinet] 设置cabinet_id:', cabinetIdInput.value);
        }

        // Cabinet Name
        const cabinetNameInput = document.getElementById('cabinetName');
        if (cabinetNameInput) {
            cabinetNameInput.value = this.cabinetInfo.name || '';
            console.log('[Cabinet] 设置name:', cabinetNameInput.value);
        }

        // Location
        const locationInput = document.getElementById('cabinetLocation');
        if (locationInput) {
            locationInput.value = this.cabinetInfo.location || '';
            console.log('[Cabinet] 设置location:', locationInput.value);
        }
        
        // Location coordinates (hidden fields)
        const latitudeInput = document.getElementById('cabinetLatitude');
        if (latitudeInput && this.cabinetInfo.latitude !== null && this.cabinetInfo.latitude !== undefined) {
            latitudeInput.value = this.cabinetInfo.latitude;
        }
        const longitudeInput = document.getElementById('cabinetLongitude');
        if (longitudeInput && this.cabinetInfo.longitude !== null && this.cabinetInfo.longitude !== undefined) {
            longitudeInput.value = this.cabinetInfo.longitude;
        }

        // Capacity - 处理数字类型
        const capacityInput = document.getElementById('cabinetCapacity');
        if (capacityInput) {
            if (this.cabinetInfo.capacity_kwh !== null && this.cabinetInfo.capacity_kwh !== undefined && this.cabinetInfo.capacity_kwh !== '') {
                capacityInput.value = String(this.cabinetInfo.capacity_kwh);
            } else {
                capacityInput.value = '';
            }
            console.log('[Cabinet] 设置capacity_kwh:', capacityInput.value);
        }

        // Device Model
        const deviceModelInput = document.getElementById('cabinetDeviceModel');
        if (deviceModelInput) {
            deviceModelInput.value = this.cabinetInfo.device_model || '';
            console.log('[Cabinet] 设置device_model:', deviceModelInput.value);
        }

        // IP Address
        const ipAddressSpan = document.getElementById('cabinetIPAddress');
        if (ipAddressSpan) {
            ipAddressSpan.textContent = this.cabinetInfo.ip_address || '0.0.0.0';
        }

        // MAC Address
        const macAddressSpan = document.getElementById('cabinetMacAddress');
        if (macAddressSpan) {
            macAddressSpan.textContent = this.cabinetInfo.mac_address;
        }

        // Status Badge
        this.updateStatusBadge();
    },

    /**
     * 更新状态徽章
     */
    updateStatusBadge() {
        const statusBadge = document.getElementById('cabinetStatusBadge');
        if (!statusBadge) return;

        if (this.cabinetInfo.registered_to_cloud) {
            statusBadge.className = 'badge badge-success';
            statusBadge.innerHTML = '<i class="fas fa-check-circle"></i> 已注册到Cloud';
        } else {
            statusBadge.className = 'badge badge-warning';
            statusBadge.innerHTML = '<i class="fas fa-exclamation-circle"></i> 未注册到Cloud';
        }
    },

    /**
     * 搜索地点建议
     */
    async searchPlaceSuggestions(keyword) {
        if (!keyword || keyword.length < 2) {
            this.hideLocationSuggestions();
            return;
        }

        try {
            console.log('[Cabinet] 搜索地点建议:', keyword);

            // 调用后端地图搜索代理接口（避免浏览器 CORS 限制）
            const response = await fetch('/api/v1/map/search', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    keyword: keyword,
                    region: '全国'
                })
            });

            if (!response.ok) {
                console.warn('[Cabinet] 地点搜索失败:', response.status);
                this.hideLocationSuggestions();
                return;
            }

            const result = await response.json();
            console.log('[Cabinet] 地点搜索结果:', result);

            // 后端返回格式: {status: 0, message: "query ok", count: N, data: [...]}
            if (result.status === 0 && result.data && result.data.length > 0) {
                this.showLocationSuggestions(result.data);
            } else {
                this.hideLocationSuggestions();
            }
        } catch (error) {
            console.error('[Cabinet] 地点搜索失败:', error);
            this.hideLocationSuggestions();
        }
    },

    /**
     * 显示地点建议列表
     */
    showLocationSuggestions(suggestions) {
        const suggestionsDiv = document.getElementById('locationSuggestions');
        if (!suggestionsDiv) return;

        // 清空之前的内容
        suggestionsDiv.innerHTML = '';

        // 创建建议项
        suggestions.forEach((item, index) => {
            const itemDiv = document.createElement('div');
            itemDiv.className = 'location-suggestion-item';
            itemDiv.innerHTML = `
                <div class="suggestion-title">${item.title}</div>
                <div class="suggestion-address">${item.address || ''}</div>
            `;
            
            itemDiv.addEventListener('click', () => {
                this.selectLocationSuggestion(item);
            });

            suggestionsDiv.appendChild(itemDiv);
        });

        suggestionsDiv.style.display = 'block';
    },

    /**
     * 隐藏地点建议列表
     */
    hideLocationSuggestions() {
        const suggestionsDiv = document.getElementById('locationSuggestions');
        if (suggestionsDiv) {
            suggestionsDiv.style.display = 'none';
            suggestionsDiv.innerHTML = '';
        }
    },

    /**
     * 选择地点建议
     */
    selectLocationSuggestion(suggestion) {
        const locationInput = document.getElementById('cabinetLocation');
        const latitudeInput = document.getElementById('cabinetLatitude');
        const longitudeInput = document.getElementById('cabinetLongitude');
        const coordsHint = document.getElementById('locationCoords');

        // 腾讯地图 API 返回的坐标在 location 对象中
        const lat = suggestion.location ? suggestion.location.lat : null;
        const lng = suggestion.location ? suggestion.location.lng : null;

        // 填充输入框
        if (locationInput) {
            locationInput.value = suggestion.title;
        }

        // 填充坐标
        if (latitudeInput && lat !== null) {
            latitudeInput.value = lat;
        }
        if (longitudeInput && lng !== null) {
            longitudeInput.value = lng;
        }

        // 更新提示信息
        if (coordsHint && lat !== null && lng !== null) {
            coordsHint.textContent = `坐标: ${lat.toFixed(6)}, ${lng.toFixed(6)}`;
            coordsHint.style.color = '#10b981';
        }

        // 更新cabinetInfo
        this.cabinetInfo.location = suggestion.title;
        this.cabinetInfo.latitude = lat;
        this.cabinetInfo.longitude = lng;

        // 隐藏建议列表
        this.hideLocationSuggestions();

        UI.showToast('位置已选择', 'success');
    },

    /**
     * 搜索位置并获取坐标（保留原有功能，用于按钮点击）
     */
    async searchLocation() {
        const locationInput = document.getElementById('cabinetLocation');
        const latitudeInput = document.getElementById('cabinetLatitude');
        const longitudeInput = document.getElementById('cabinetLongitude');
        const coordsHint = document.getElementById('locationCoords');
        
        if (!locationInput) {
            console.error('[Cabinet] 位置输入框不存在');
            return;
        }
        
        const address = locationInput.value.trim();
        if (!address) {
            UI.showToast('请输入位置信息', 'warning');
            return;
        }

        // 显示加载状态
        const searchBtn = document.getElementById('searchLocationBtn');
        if (searchBtn) {
            searchBtn.disabled = true;
            searchBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i>';
        }

        try {
            console.log('[Cabinet] 开始地理编码:', address);
            
            // 调用Cloud端的地理编码API
            const response = await fetch(`${this.cloudConfig.endpoint}/map/geocode`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${this.cloudConfig.admin_token || ''}`
                },
                body: JSON.stringify({
                    address: address
                })
            });

            if (!response.ok) {
                const error = await response.json().catch(() => ({ message: '地理编码请求失败' }));
                throw new Error(error.message || '地理编码失败');
            }

            const result = await response.json();
            console.log('[Cabinet] 地理编码结果:', result);

            if (result.success && result.data) {
                // 更新隐藏字段
                if (latitudeInput) {
                    latitudeInput.value = result.data.latitude;
                }
                if (longitudeInput) {
                    longitudeInput.value = result.data.longitude;
                }

                // 更新提示信息
                if (coordsHint) {
                    coordsHint.textContent = `坐标: ${result.data.latitude.toFixed(6)}, ${result.data.longitude.toFixed(6)}`;
                    coordsHint.style.color = '#10b981';
                }

                // 更新cabinetInfo
                this.cabinetInfo.latitude = result.data.latitude;
                this.cabinetInfo.longitude = result.data.longitude;

                UI.showToast('位置坐标获取成功', 'success');
            } else {
                throw new Error('地理编码返回数据格式错误');
            }
        } catch (error) {
            console.error('[Cabinet] 地理编码失败:', error);
            UI.showToast('位置搜索失败: ' + error.message, 'error');
            
            // 重置坐标提示
            if (coordsHint) {
                coordsHint.textContent = '储能柜物理位置';
                coordsHint.style.color = '';
            }
        } finally {
            // 恢复按钮状态
            if (searchBtn) {
                searchBtn.disabled = false;
                searchBtn.innerHTML = '<i class="fas fa-search"></i>';
            }
        }
    },

    /**
     * 保存储能柜信息
     * @param {boolean} showErrors - 是否显示错误提示（默认true）
     */
    async saveCabinetInfo(showErrors = true) {
        // 获取表单数据
        const cabinetId = document.getElementById('cabinetId')?.value.trim();
        const cabinetName = document.getElementById('cabinetName')?.value.trim();
        const location = document.getElementById('cabinetLocation')?.value.trim();
        const latitude = document.getElementById('cabinetLatitude')?.value;
        const longitude = document.getElementById('cabinetLongitude')?.value;
        const capacity = parseFloat(document.getElementById('cabinetCapacity')?.value);
        const deviceModel = document.getElementById('cabinetDeviceModel')?.value.trim();

        // 验证必填字段
        if (!cabinetId) {
            if (showErrors) {
                UI.showToast('请输入储能柜ID', 'error');
            }
            return false;
        }

        if (!cabinetName) {
            if (showErrors) {
                UI.showToast('请输入储能柜名称', 'error');
            }
            return false;
        }

        // ⚠️ 检查是否修改了储能柜ID
        const originalCabinetId = this.cabinetInfo?.cabinet_id;
        if (originalCabinetId && cabinetId !== originalCabinetId) {
            if (showErrors) {
                const confirmed = confirm(
                    `⚠️ 注意：修改储能柜ID后需要重启Edge服务才能生效！\n\n` +
                    `当前ID: ${originalCabinetId}\n` +
                    `新ID: ${cabinetId}\n\n` +
                    `修改后请手动执行以下操作：\n` +
                    `1. 保存配置\n` +
                    `2. 在Cloud端查看新ID是否已注册\n` +
                    `3. 重启Edge服务使新配置生效\n\n` +
                    `是否继续保存？`
                );
                if (!confirmed) {
                    console.log('[Cabinet] 用户取消了储能柜ID修改');
                    return false;
                }
            }
        }

        // 更新信息
        this.cabinetInfo.cabinet_id = cabinetId;
        this.cabinetInfo.name = cabinetName;
        this.cabinetInfo.location = location;
        this.cabinetInfo.latitude = latitude ? parseFloat(latitude) : null;
        this.cabinetInfo.longitude = longitude ? parseFloat(longitude) : null;
        this.cabinetInfo.capacity_kwh = capacity || 0;
        this.cabinetInfo.device_model = deviceModel || '';

        // 保存到本地存储
        localStorage.setItem('cabinet_info', JSON.stringify(this.cabinetInfo));

        let apiSuccess = false;
        let cloudSyncSuccess = false;
        let errorMessage = '';

        // 通过Edge后端API保存并同步到Cloud端（统一入口）
        // 前端只调用Edge API，由Edge后端负责：
        // 1. 更新配置文件中的cabinet_id
        // 2. 同步储能柜信息到Cloud端
        try {
            console.log('[Cabinet] 正在通过Edge后端API保存并同步储能柜信息...');
            
            const syncData = {
                cabinet_id: cabinetId,
                name: cabinetName,
                location: location || '',
                latitude: latitude ? parseFloat(latitude) : null,
                longitude: longitude ? parseFloat(longitude) : null,
                capacity_kwh: capacity || null,
                device_model: deviceModel || ''
            };

            console.log('[Cabinet] 调用Edge后端API:', syncData);

            const syncResponse = await fetch(`${API_BASE_URL}/api/v1/cabinets/info`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(syncData)
            });

            if (syncResponse.ok) {
                const result = await syncResponse.json();
                console.log('[Cabinet] Edge后端API响应:', result);
                
                apiSuccess = result.success || result.config_update_success;
                cloudSyncSuccess = result.cloud_sync_success || false;
                
                if (cloudSyncSuccess) {
                    console.log('[Cabinet] ✅ 储能柜信息已成功同步到Cloud端');
                } else if (result.cloud_sync_error) {
                    console.warn('[Cabinet] ⚠ Cloud端同步失败:', result.cloud_sync_error);
                    errorMessage = result.cloud_sync_error;
                } else {
                    console.log('[Cabinet] ℹ Cloud同步未执行（可能未配置）');
                }
            } else {
                const error = await syncResponse.json().catch(() => ({
                    message: `HTTP ${syncResponse.status} ${syncResponse.statusText}`
                }));
                console.warn('[Cabinet] Edge后端API调用失败:', error);
                errorMessage = error.message || '保存失败';
            }
        } catch (error) {
            console.error('[Cabinet] 调用Edge后端API时发生错误:', error);
            errorMessage = error.message;
        }

        // 显示结果提示
        if (showErrors) {
            if (apiSuccess && cloudSyncSuccess) {
                UI.showToast('✓ 储能柜信息已保存并成功同步到Cloud端', 'success');
            } else if (apiSuccess && !cloudSyncSuccess && this.cloudConfig.enabled && this.cloudConfig.endpoint) {
                UI.showToast('✓ 储能柜信息已保存到Edge端\n⚠ 但同步到Cloud端失败：' + (errorMessage || '请检查Cloud服务是否启动'), 'warning');
            } else if (apiSuccess) {
                UI.showToast('✓ 储能柜信息已保存到Edge端', 'success');
            } else {
                UI.showToast('⚠ 保存失败：' + errorMessage, 'error');
            }
        }

        return apiSuccess;
    },

    /**
     * 加载Cloud配置
     */
    async loadCloudConfig() {
        try {
            console.log('[Cabinet] 正在加载Cloud配置...');
            const response = await fetch(`${API_BASE_URL}/api/v1/config`);
            console.log('[Cabinet] Cloud配置API响应状态:', response.status);

            if (response.ok) {
                // 检查响应是否为JSON
                const contentType = response.headers.get('Content-Type');
                if (!contentType || !contentType.includes('application/json')) {
                    const text = await response.text();
                    console.error('[Cabinet] 配置API返回非JSON响应:', text.substring(0, 200));
                    throw new Error(`配置API响应格式错误（HTTP ${response.status}）`);
                }

                const config = await response.json();
                console.log('[Cabinet] Cloud配置数据:', config);

                if (config && config.cloud) {
                    this.cloudConfig = {
                        enabled: config.cloud.enabled !== undefined ? config.cloud.enabled : false,
                        endpoint: config.cloud.endpoint || '',
                        api_key: config.cloud.api_key || '',
                        admin_token: config.cloud.admin_token || '',
                        cabinet_id: config.cloud.cabinet_id || ''
                    };

                    console.log('[Cabinet] 解析后的Cloud配置:', this.cloudConfig);
                } else {
                    console.warn('[Cabinet] 配置响应中没有cloud字段');
                }

                // 更新Cloud配置UI
                this.updateCloudConfigUI();
            } else {
                const errorText = await response.text();
                console.error('[Cabinet] Cloud配置API返回错误:', response.status, errorText);
                UI.showToast('获取Cloud配置失败: ' + response.status, 'error');
            }
        } catch (error) {
            console.error('[Cabinet] 加载Cloud配置失败:', error);
            UI.showToast('加载Cloud配置失败: ' + error.message, 'error');
        }
    },

    /**
     * 更新Cloud配置UI
     */
    updateCloudConfigUI() {
        console.log('[Cabinet] 更新Cloud配置UI, 当前配置:', JSON.stringify(this.cloudConfig));
        
        const cloudEndpoint = document.getElementById('cloudEndpoint');
        if (cloudEndpoint) {
            const endpoint = this.cloudConfig.endpoint || '未配置';
            cloudEndpoint.textContent = endpoint;
            // 如果已配置，添加可编辑提示
            if (endpoint !== '未配置') {
                cloudEndpoint.title = '点击编辑';
                cloudEndpoint.style.cursor = 'pointer';
            }
            console.log('[Cabinet] 更新Cloud端地址:', endpoint);
        } else {
            console.warn('[Cabinet] 未找到cloudEndpoint元素');
        }

        // 更新API Key显示
        const cloudApiKey = document.getElementById('cloudApiKey');
        if (cloudApiKey) {
            const apiKey = this.cloudConfig.api_key || '';
            if (apiKey) {
                // 脱敏显示：显示前10个字符 + *** + 后3个字符
                if (apiKey.length > 13) {
                    const masked = apiKey.substring(0, 10) + '***' + apiKey.substring(apiKey.length - 3);
                    cloudApiKey.textContent = masked;
                } else {
                    cloudApiKey.textContent = '***';
                }
                cloudApiKey.style.color = '#28a745'; // 绿色表示已配置
            } else {
                cloudApiKey.textContent = '未配置';
                cloudApiKey.style.color = '#999';
            }
            console.log('[Cabinet] 更新API Key显示:', apiKey ? '已配置' : '未配置');
        } else {
            console.warn('[Cabinet] 未找到cloudApiKey元素');
        }

        const cloudEnabled = document.getElementById('cloudEnabled');
        if (cloudEnabled) {
            // 确保将enabled转换为布尔值
            const enabled = Boolean(this.cloudConfig.enabled);
            cloudEnabled.textContent = enabled ? '已启用' : '未启用';
            cloudEnabled.className = enabled ? 'badge badge-success' : 'badge badge-secondary';
            console.log('[Cabinet] 更新启用状态:', enabled, '(原始值:', this.cloudConfig.enabled, ')');
        } else {
            console.warn('[Cabinet] 未找到cloudEnabled元素');
        }

        // 根据配置状态启用/禁用注册按钮（直接注册不需要admin_token）
        const registerBtn = document.getElementById('registerToCloudBtn');
        if (registerBtn) {
            const shouldDisable = !this.cloudConfig.enabled || !this.cloudConfig.endpoint;
            registerBtn.disabled = shouldDisable;
            registerBtn.title = shouldDisable ? 'Cloud端未配置或未启用' : '点击注册到Cloud端';
            console.log('[Cabinet] 注册按钮状态:', shouldDisable ? '禁用' : '启用');
        }
        
        // 根据配置状态启用/禁用测试连接按钮
        const testBtn = document.getElementById('testCloudConnectionBtn');
        if (testBtn) {
            const shouldDisable = !this.cloudConfig.enabled || !this.cloudConfig.endpoint;
            testBtn.disabled = shouldDisable;
            testBtn.title = shouldDisable ? 'Cloud端未配置或未启用' : '点击测试连接';
        }
    },

    /**
     * 测试Cloud连接
     */
    /**
     * 测试Cloud连接（通过Edge后端代理，避免浏览器CORS限制）
     */
    async testCloudConnection() {
        if (!this.cloudConfig.enabled || !this.cloudConfig.endpoint) {
            UI.showToast('Cloud端未配置或未启用', 'warning');
            return;
        }

        const testBtn = document.getElementById('testCloudConnectionBtn');
        if (testBtn) {
            testBtn.disabled = true;
            testBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 测试中...';
        }

        try {
            console.log('[Cabinet] 测试Cloud连接（通过Edge代理）:', this.cloudConfig.endpoint);

            // 调用Edge后端的代理接口（使用API.request确保正确的baseURL）
            const startTime = performance.now();

            const result = await API.request('/api/v1/config/test-cloud', {
                method: 'GET'
            });

            const endTime = performance.now();
            const totalLatency = Math.round(endTime - startTime);

            console.log('[Cabinet] 代理响应:', result);

            if (result.success) {
                // 连接成功
                let message = `✓ 连接成功！\n`;
                message += `• 总延迟: ${totalLatency}ms\n`;
                message += `• Cloud延迟: ${result.details.latency}ms\n`;
                message += `• HTTP状态: ${result.details.status_code}\n`;
                if (result.data && result.data.status) {
                    message += `• 服务状态: ${result.data.status}\n`;
                }
                if (result.data && result.data.service) {
                    message += `• 服务名称: ${result.data.service}`;
                }
                
                UI.showToast(message, 'success');
            } else {
                // 连接失败
                let errorMsg = `✗ 连接失败\n`;
                errorMsg += `• 原因: ${result.message}\n`;
                errorMsg += `• 目标地址: ${result.details.endpoint}`;
                
                UI.showToast(errorMsg, 'error');
            }
        } catch (error) {
            console.error('[Cabinet] 测试连接失败:', error);
            
            let errorMsg = '✗ 连接失败\n';
            errorMsg += `• 原因: ${error.message}\n`;
            errorMsg += '• 建议: 检查Edge端服务是否正常运行';
            
            UI.showToast(errorMsg, 'error');
        } finally {
            if (testBtn) {
                testBtn.disabled = false;
                testBtn.innerHTML = '<i class="fas fa-link"></i> 测试连接';
            }
        }
    },

    /**
     * 注册储能柜到Cloud端（通过Edge后端代理）
     */
    async registerToCloud() {
        // 先保存信息（不显示错误提示，避免重复）
        const saveSuccess = await this.saveCabinetInfo(false);
        if (!saveSuccess) {
            // 如果保存失败，检查是否是必填字段问题
            if (!this.cabinetInfo.cabinet_id || !this.cabinetInfo.name) {
                UI.showToast('请先完善储能柜信息（ID和名称必填）', 'error');
                return;
            }
        }

        // 验证必填字段
        if (!this.cabinetInfo.cabinet_id || !this.cabinetInfo.name) {
            UI.showToast('请先完善储能柜信息（ID和名称必填）', 'error');
            return;
        }

        if (!this.cloudConfig.enabled || !this.cloudConfig.endpoint) {
            UI.showToast('Cloud端未配置或未启用，请检查配置文件', 'warning');
            return;
        }

        // 先检查Cloud服务是否可达（使用代理）
        try {
            console.log('[Cabinet] 检查Cloud服务连接状态（通过Edge代理）...');
            const healthResult = await API.request('/api/v1/config/test-cloud', {
                method: 'GET'
            });

            if (!healthResult.success) {
                throw new Error(healthResult.message || 'Cloud服务不可达');
            }

            console.log('[Cabinet] Cloud服务连接正常');
        } catch (healthError) {
            console.error('[Cabinet] Cloud服务不可达:', healthError);
            let errorMsg = '✗ 连接失败\n';
            if (healthError.name === 'AbortError' || healthError.message.includes('timeout')) {
                errorMsg += '• 原因: 连接超时（10秒）\n';
                errorMsg += '• 建议: 检查Cloud端地址和网络连接';
            } else {
                errorMsg += `• 原因: ${healthError.message}\n`;
                errorMsg += '• 建议: 检查Cloud端是否正常运行';
            }
            UI.showToast(errorMsg, 'error');
            return;
        }

        const registerBtn = document.getElementById('registerToCloudBtn');
        if (registerBtn) {
            registerBtn.disabled = true;
            registerBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 注册中...';
        }

        try {
            // 构建注册请求
            const payload = {
                cabinet_id: this.cabinetInfo.cabinet_id,
                name: this.cabinetInfo.name,
                location: this.cabinetInfo.location || null,
                latitude: this.cabinetInfo.latitude || null,
                longitude: this.cabinetInfo.longitude || null,
                capacity_kwh: this.cabinetInfo.capacity_kwh || null,
                device_model: this.cabinetInfo.device_model || null,
                ip_address: this.cabinetInfo.ip_address || null,
                mac_address: this.cabinetInfo.mac_address
            };

            console.log('[Cabinet] 注册储能柜到Cloud端（通过Edge代理）:', payload);

            // 调用Edge端代理接口（使用API.request确保正确的baseURL）
            const startTime = Date.now();
            const result = await API.request('/api/v1/cloud/register', {
                method: 'POST',
                body: JSON.stringify(payload)
            });
            const latency = Date.now() - startTime;

            console.log('[Cabinet] 注册请求完成，延迟:', latency + 'ms');
            console.log('[Cabinet] 注册响应数据:', result);

            if (result.success) {
                // 注册成功，保存API凭证
                this.cabinetInfo.registered_to_cloud = true;
                
                // 保存返回的API Key和Secret
                if (result.data && result.data.api_key) {
                    console.log('[Cabinet] 获得API凭证:', {
                        api_key: result.data.api_key,
                        api_secret: result.data.api_secret ? '***' : 'null'
                    });
                    
                    // 自动保存API凭证到config.yaml
                    try {
                        const saveResult = await API.request('/api/v1/config/credentials', {
                            method: 'PUT',
                            body: JSON.stringify({
                                api_key: result.data.api_key,
                                api_secret: result.data.api_secret
                            })
                        });

                        console.log('[Cabinet] API凭证已保存到配置文件:', saveResult);
                        UI.showToast(
                            `✓ 注册成功！\n` +
                            `• API Key: ${result.data.api_key.substring(0, 20)}...\n` +
                            `• 凭证已自动保存到 configs/config.yaml\n` +
                            `• api_secret请妥善保管：${result.data.api_secret}`,
                            'success'
                        );
                    } catch (saveError) {
                        console.error('[Cabinet] 保存凭证异常:', saveError);
                        UI.showToast(
                            `✓ 注册成功但保存凭证失败\n` +
                            `请手动保存到 configs/config.yaml:\n` +
                            `api_key: "${result.data.api_key}"\n` +
                            `api_secret: "${result.data.api_secret}"`,
                            'warning'
                        );
                    }
                    
                    // 保存到localStorage（临时）
                    this.cabinetInfo.api_key = result.data.api_key;
                    this.cabinetInfo.api_secret = result.data.api_secret;
                }
                
                localStorage.setItem('cabinet_info', JSON.stringify(this.cabinetInfo));
                this.updateStatusBadge();

                console.log('[Cabinet] 注册成功:', result);
            } else {
                // 注册失败
                const errorMsg = result.message || result.error?.message || result.error || '注册失败';
                const errorCode = result.error?.code || result.error || 'UNKNOWN_ERROR';
                const statusCode = result.details?.status_code || result.status_code || 'N/A';
                console.error('[Cabinet] 注册失败:', {
                    statusCode: statusCode,
                    errorCode: errorCode,
                    errorMsg: errorMsg,
                    fullResponse: result
                });
                
                let displayMsg = `✗ 注册失败\n`;
                displayMsg += `• 原因: ${errorMsg}`;
                if (statusCode !== 'N/A') {
                    displayMsg += `\n• HTTP状态码: ${statusCode}`;
                }
                if (errorCode !== 'UNKNOWN_ERROR') {
                    displayMsg += `\n• 错误代码: ${errorCode}`;
                }
                UI.showToast(displayMsg, 'error');
            }
        } catch (error) {
            console.error('[Cabinet] 注册到Cloud失败:', error);
            let errorMsg = '✗ 注册失败\n';

            // 特殊处理409冲突错误（储能柜已注册）
            if (error.message.includes('409') || error.message.includes('Conflict')) {
                errorMsg = '⚠️ 储能柜已注册\n';
                errorMsg += '• 该储能柜ID已在Cloud端注册过\n';
                errorMsg += '• 如需重新注册，请先在Cloud端删除该储能柜\n';
                errorMsg += '• 或者修改本地的cabinet_id后再注册';
                UI.showToast(errorMsg, 'warning');
            } else if (error.name === 'AbortError' || error.message.includes('timeout')) {
                errorMsg += '• 原因: 请求超时（30秒）\n';
                errorMsg += '• 建议: 检查网络连接和Cloud端状态';
                UI.showToast(errorMsg, 'error');
            } else if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
                errorMsg += '• 原因: 无法连接到服务器\n';
                errorMsg += '• 建议: 检查Cloud端地址和网络连接\n';
                errorMsg += `• 目标地址: ${this.cloudConfig.endpoint}/cabinets/register`;
                UI.showToast(errorMsg, 'error');
            } else if (error.message.includes('CORS')) {
                errorMsg += '• 原因: CORS跨域错误\n';
                errorMsg += '• 建议: 检查Cloud端CORS配置';
                UI.showToast(errorMsg, 'error');
            } else {
                errorMsg += `• 原因: ${error.message}`;
                UI.showToast(errorMsg, 'error');
            }
        } finally {
            if (registerBtn) {
                registerBtn.disabled = false;
                registerBtn.innerHTML = '<i class="fas fa-cloud-upload-alt"></i> 注册到Cloud端';
            }
        }
    },

    /**
     * 显示Cloud配置编辑表单
     */
    showCloudConfigEdit() {
        console.log('[Cabinet] 显示配置编辑表单');
        
        // 隐藏显示区域
        const displayDiv = document.getElementById('cloudConfigDisplay');
        if (displayDiv) {
            displayDiv.style.display = 'none';
        }

        // 显示编辑区域
        const editDiv = document.getElementById('cloudConfigEdit');
        if (editDiv) {
            editDiv.style.display = 'block';
        }

        // 填充当前配置到表单（使用当前配置值）
        const endpointInput = document.getElementById('cloudEndpointInput');
        if (endpointInput) {
            // 使用当前配置的值，如果为空则显示空值让用户输入
            endpointInput.value = this.cloudConfig.endpoint || '';
            console.log('[Cabinet] 填充endpoint:', endpointInput.value);
        }

        const enabledInput = document.getElementById('cloudEnabledInput');
        if (enabledInput) {
            enabledInput.checked = Boolean(this.cloudConfig.enabled);
            console.log('[Cabinet] 填充enabled:', enabledInput.checked);
        }

        const apiKeyInput = document.getElementById('cloudApiKeyInput');
        if (apiKeyInput) {
            apiKeyInput.value = this.cloudConfig.api_key || '';
            console.log('[Cabinet] 填充api_key:', this.cloudConfig.api_key ? '***' : '');
        }
    },

    /**
     * 隐藏Cloud配置编辑表单
     */
    hideCloudConfigEdit() {
        console.log('[Cabinet] 隐藏配置编辑表单');
        
        // 显示显示区域
        const displayDiv = document.getElementById('cloudConfigDisplay');
        if (displayDiv) {
            displayDiv.style.display = 'block';
        }

        // 隐藏编辑区域
        const editDiv = document.getElementById('cloudConfigEdit');
        if (editDiv) {
            editDiv.style.display = 'none';
        }
    },

    /**
     * 保存Cloud配置
     */
    async saveCloudConfig() {
        try {
            const endpointInput = document.getElementById('cloudEndpointInput');
            const enabledInput = document.getElementById('cloudEnabledInput');
            const apiKeyInput = document.getElementById('cloudApiKeyInput');

            const endpoint = endpointInput.value.trim();
            const enabled = enabledInput.checked;
            const apiKey = apiKeyInput.value.trim();

            console.log('[Cabinet] 保存Cloud配置:', { endpoint, enabled, apiKey: apiKey ? '***' : '' });

            // 验证endpoint格式
            if (endpoint && !endpoint.startsWith('http://') && !endpoint.startsWith('https://')) {
                UI.showToast('Cloud端地址必须以http://或https://开头', 'error');
                return;
            }

            // 验证API Key格式
            if (enabled && !apiKey) {
                UI.showToast('启用Cloud连接时必须配置API Key', 'error');
                return;
            }

            if (apiKey && !apiKey.startsWith('ck_')) {
                UI.showToast('API Key格式无效，应以ck_开头', 'warning');
            }

            // 调用API更新配置
            const response = await fetch(`${API_BASE_URL}/api/v1/config`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    cloud: {
                        enabled: enabled,
                        endpoint: endpoint,
                        api_key: apiKey
                    }
                })
            });

            if (response.ok) {
                const result = await response.json();
                console.log('[Cabinet] 配置更新成功:', result);

                // 更新本地配置
                this.cloudConfig.enabled = enabled;
                this.cloudConfig.endpoint = endpoint;
                this.cloudConfig.api_key = apiKey;

                // 更新UI显示
                this.updateCloudConfigUI();

                // 隐藏编辑表单
                this.hideCloudConfigEdit();

                UI.showToast('Cloud配置已更新，建议重启Edge服务以完全生效', 'success');
            } else {
                // 尝试解析错误响应，如果不是JSON则显示状态码
                let errorMessage = `HTTP ${response.status}`;
                try {
                    const contentType = response.headers.get('Content-Type');
                    if (contentType && contentType.includes('application/json')) {
                        const error = await response.json();
                        errorMessage = error.message || errorMessage;
                    } else {
                        const text = await response.text();
                        console.error('[Cabinet] 非JSON响应:', text.substring(0, 200));
                    }
                } catch (parseError) {
                    console.error('[Cabinet] 解析错误响应失败:', parseError);
                }
                console.error('[Cabinet] 配置更新失败:', errorMessage);
                UI.showToast('配置更新失败: ' + errorMessage, 'error');
            }
        } catch (error) {
            console.error('[Cabinet] 保存配置失败:', error);
            UI.showToast('保存配置失败: ' + error.message, 'error');
        }
    }
};

// 页面加载后自动初始化
if (typeof App !== 'undefined') {
    // 等待App初始化完成后再初始化Cabinet
    console.log('[Cabinet] 模块已加载，等待App初始化');
} else {
    console.warn('[Cabinet] App对象未找到，延迟初始化');
}
