/**
 * 统计分析模块 - 7种传感器历史数据曲线图
 * 支持7天/15天/30天的时间范围选择
 * 使用WebSocket实现图表增量更新
 * @version 4.0.0
 */

const Statistics = {
    // Chart实例存储
    charts: {},

    // 当前选择的天数（默认为0表示实时状态，显示今日数据）
    currentDays: 0,

    // 初始化标志
    isInitialized: false,

    // 正在加载的传感器记录(防止重复请求)
    loadingRequests: new Map(),

    // WebSocket订阅取消函数
    unsubscribeSensorData: null,
    unsubscribeConnection: null,

    // 各传感器的原始数据缓存(用于增量更新)
    rawDataCache: {},

    // 传感器配置
    sensorConfigs: {
        co2: {
            name: 'CO2',
            unit: 'ppm',
            color: '#8b5cf6',
            thresholds: { max: 5000 }
        },
        co: {
            name: 'CO',
            unit: 'ppm',
            color: '#ec4899',
            thresholds: { max: 50 }
        },
        smoke: {
            name: '烟雾',
            unit: 'AD值',
            color: '#f59e0b',
            thresholds: { max: 1000 }
        },
        liquid_level: {
            name: '液位',
            unit: 'mm',
            color: '#06b6d4',
            thresholds: { min: 100, max: 900 }
        },
        conductivity: {
            name: '电导率',
            unit: 'mS/cm',
            color: '#eab308',
            thresholds: { min: 0.5, max: 10 }
        },
        temperature: {
            name: '温度',
            unit: '°C',
            color: '#ef4444',
            thresholds: { min: -10, max: 60 }
        },
        flow: {
            name: '流速',
            unit: 'L/min',
            color: '#10b981',
            thresholds: { min: 0.5, max: 100 }
        }
    },

    /**
     * 初始化统计分析模块
     */
    async init() {
        console.log('[Statistics] 初始化统计分析模块...');
        console.log('[Statistics] Chart.js 是否加载:', typeof Chart !== 'undefined');

        // 重新绑定时间选择器事件（每次初始化都重新绑定，确保事件正确）
        console.log('[Statistics] 绑定时间选择器事件监听器');
        this.bindTimeSelector();

        // 订阅WebSocket消息
        console.log('[Statistics] 订阅WebSocket消息...');
        this.subscribeWebSocket();

        // 加载所有图表
        await this.loadAllCharts();
        console.log('[Statistics] 初始化完成');
        
        this.isInitialized = true;
    },

    /**
     * 订阅WebSocket消息
     */
    subscribeWebSocket() {
        if (typeof WebSocketManager === 'undefined') {
            console.warn('[Statistics] WebSocketManager未定义，跳过WebSocket订阅');
            return;
        }

        // 订阅传感器数据 - 用于实时更新图表
        this.unsubscribeSensorData = WebSocketManager.subscribe('sensor_data', (data) => {
            this.handleRealtimeSensorData(data);
        });

        // 订阅连接状态变化
        this.unsubscribeConnection = WebSocketManager.subscribe('connection', (data) => {
            if (data.status === 'connected') {
                console.log('[Statistics] WebSocket重连，刷新图表数据');
                // 重连后只在实时模式下刷新
                if (this.currentDays === 0) {
                    this.loadAllCharts();
                }
            }
        });
    },

    /**
     * 处理实时传感器数据 - 增量更新图表
     * @param {Object} data - 传感器数据
     */
    handleRealtimeSensorData(data) {
        // 只在实时模式(今日数据)下增量更新
        if (this.currentDays !== 0) {
            return;
        }

        if (!data || !data.sensor_type) {
            return;
        }

        const sensorType = data.sensor_type;
        const chart = this.charts[sensorType];
        
        if (!chart) {
            console.log(`[Statistics] ${sensorType} 图表未初始化，跳过增量更新`);
            return;
        }

        console.log(`[Statistics] 实时更新 ${sensorType} 图表`);

        // 添加新数据点到缓存
        if (!this.rawDataCache[sensorType]) {
            this.rawDataCache[sensorType] = [];
        }
        this.rawDataCache[sensorType].push(data);

        // 更新图表
        const config = this.sensorConfigs[sensorType];
        const chartData = this.processChartData(this.rawDataCache[sensorType]);
        
        // 更新图表数据
        chart.data.labels = chartData.labels;
        chart.data.datasets[0].data = chartData.values;
        chart.update('none'); // 'none'模式避免动画

        // 更新统计信息
        this.updateStatistics(sensorType, this.rawDataCache[sensorType]);
    },

    /**
     * 绑定时间选择器事件
     */
    bindTimeSelector() {
        const timeButtons = document.querySelectorAll('.time-btn');
        console.log(`[Statistics] 找到 ${timeButtons.length} 个时间按钮`);

        timeButtons.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                const days = parseInt(e.target.dataset.days);
                console.log('[Statistics] 点击时间按钮:', days === 0 ? '实时状态(当天)' : `${days}天`);

                // 更新按钮状态
                timeButtons.forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');

                // 更新选择的天数
                const oldDays = this.currentDays;
                this.currentDays = days;
                console.log(`[Statistics] 时间范围切换: ${oldDays === 0 ? '实时' : oldDays + '天'} → ${days === 0 ? '实时(当天)' : days + '天'}`);

                // 重新加载所有图表
                console.log('[Statistics] 重新加载所有图表...');
                this.loadAllCharts();
            });
        });
    },

    /**
     * 加载所有传感器的图表
     */
    async loadAllCharts() {
        const timeDesc = this.currentDays === 0 ? '今日实时数据' : `${this.currentDays}天历史数据`;
        console.log(`加载所有图表 (${timeDesc})...`);

        const sensorTypes = Object.keys(this.sensorConfigs);

        // 顺序加载,避免触发后端限流 (改为串行加载,添加延迟)
        for (let i = 0; i < sensorTypes.length; i++) {
            const type = sensorTypes[i];
            await this.loadSensorChart(type);

            // 每个请求之间延迟100ms,避免触发限流
            if (i < sensorTypes.length - 1) {
                await new Promise(resolve => setTimeout(resolve, 100));
            }
        }

        console.log('所有图表加载完成');
    },

    /**
     * 加载单个传感器的图表
     */
    async loadSensorChart(sensorType) {
        // 生成请求key(传感器类型+天数)
        const requestKey = `${sensorType}_${this.currentDays}`;

        // 检查是否已有相同的请求正在进行
        if (this.loadingRequests.has(requestKey)) {
            console.log(`[Statistics] ⚠️ ${sensorType} 正在加载中,跳过重复请求 (key: ${requestKey})`);
            return this.loadingRequests.get(requestKey);
        }

        try {
            const timeDesc = this.currentDays === 0 ? '今日实时' : `${this.currentDays}天`;
            console.log(`[Statistics] 开始加载 ${sensorType} 图表, 时间范围: ${timeDesc}`);
            const config = this.sensorConfigs[sensorType];

            // 获取服务器时间(修复时间同步问题)
            const serverTime = await API.getServerTime();

            // 计算时间范围
            const endTime = new Date(serverTime.getTime());
            const startTime = new Date(serverTime.getTime());

            if (this.currentDays === 0) {
                // 实时状态：显示今天从00:00:00开始的数据
                startTime.setHours(0, 0, 0, 0);
                console.log(`[Statistics] 实时状态模式: 今日 ${startTime.toLocaleString()} 至 ${endTime.toLocaleString()}`);
            } else {
                // 历史数据：显示最近N天的数据
                startTime.setDate(startTime.getDate() - this.currentDays);
                console.log(`[Statistics] 历史数据模式: ${this.currentDays}天`);
            }

            console.log(`[Statistics] 时间范围: ${startTime.toISOString()} 到 ${endTime.toISOString()}`);

            // 查询数据
            const queryParams = {
                sensor_type: sensorType,
                start_time: startTime.toISOString(),
                end_time: endTime.toISOString(),
                limit: 1000000, // 取消限制：设置为100000，足够获取所有历史数据
                page: 1
            };
            console.log(`[Statistics] API查询参数:`, queryParams);

            // 创建Promise并存储,用于去重
            const loadPromise = (async () => {
                const response = await API.queryData(queryParams);
                console.log(`[Statistics] API返回数据:`, response);
                return response;
            })();

            this.loadingRequests.set(requestKey, loadPromise);

            const response = await loadPromise;

            if (response && response.data && response.data.length > 0) {
                console.log(`[Statistics] ${sensorType} 获取到 ${response.data.length} 条数据`);

                // 缓存原始数据(用于增量更新)
                this.rawDataCache[sensorType] = response.data;

                // 处理数据
                const chartData = this.processChartData(response.data);
                console.log(`[Statistics] 处理后的图表数据:`, chartData);

                // 更新统计信息
                this.updateStatistics(sensorType, response.data);

                // 渲染图表
                this.renderChart(sensorType, chartData, config);
            } else {
                console.warn(`[Statistics] ${sensorType} 无数据, response:`, response);
                // 清空缓存
                this.rawDataCache[sensorType] = [];
                // 无数据
                this.showNoData(sensorType);
            }
        } catch (error) {
            console.error(`[Statistics] 加载${sensorType}图表失败:`, error);
            this.showNoData(sensorType);
        } finally {
            // 清理请求记录,允许后续重新加载
            this.loadingRequests.delete(requestKey);
            console.log(`[Statistics] ${sensorType} 加载请求已完成,清理请求记录 (key: ${requestKey})`);
        }
    },

    /**
     * 处理图表数据
     */
    processChartData(data) {
        // 按时间排序
        const sortedData = data.sort((a, b) =>
            new Date(a.timestamp) - new Date(b.timestamp)
        );

        // 数据采样 - 如果数据点过多,进行采样
        const maxPoints = 100; // 最多显示100个点
        let sampledData = sortedData;

        if (sortedData.length > maxPoints) {
            const step = Math.ceil(sortedData.length / maxPoints);
            sampledData = sortedData.filter((_, index) => index % step === 0);
        }

        // 提取时间和数值
        const labels = sampledData.map(d => {
            const date = new Date(d.timestamp);
            return date.toLocaleString('zh-CN', {
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit'
            });
        });

        const values = sampledData.map(d => parseFloat(d.value));

        return { labels, values, rawData: sortedData };
    },

    /**
     * 更新统计信息
     */
    updateStatistics(sensorType, data) {
        const values = data.map(d => parseFloat(d.value));

        const count = data.length;
        const min = Math.min(...values).toFixed(1);
        const max = Math.max(...values).toFixed(1);
        const avg = (values.reduce((a, b) => a + b, 0) / count).toFixed(1);

        // 转换为驼峰命名: liquid_level -> liquidLevel
        const camelCase = sensorType.replace(/_([a-z])/g, (m, p1) => p1.toUpperCase());

        // 安全地更新DOM元素
        const countEl = document.getElementById(`${camelCase}Count`);
        const minEl = document.getElementById(`${camelCase}Min`);
        const maxEl = document.getElementById(`${camelCase}Max`);
        const avgEl = document.getElementById(`${camelCase}Avg`);

        if (countEl) countEl.textContent = count;
        if (minEl) minEl.textContent = min;
        if (maxEl) maxEl.textContent = max;
        if (avgEl) avgEl.textContent = avg;

        console.log(`[Statistics] ${sensorType} 统计信息已更新:`, { count, min, max, avg });
    },

    /**
     * 渲染图表
     */
    renderChart(sensorType, chartData, config) {
        console.log(`[Statistics] 开始渲染 ${sensorType} 图表`);

        // 转换sensor_type为驼峰命名: liquid_level -> liquidLevel
        const camelType = sensorType.replace(/_([a-z])/g, (m, p1) => p1.toUpperCase());
        const canvasId = `${camelType}Chart`;
        console.log(`[Statistics] Canvas ID: ${canvasId}`);

        const canvas = document.getElementById(canvasId);

        if (!canvas) {
            console.error(`[Statistics] Canvas not found: ${canvasId} (sensor_type: ${sensorType})`);
            return;
        }

        console.log(`[Statistics] Canvas元素找到:`, canvas);

        const ctx = canvas.getContext('2d');

        // 销毁旧图表
        if (this.charts[sensorType]) {
            console.log(`[Statistics] 销毁旧图表: ${sensorType}`);
            this.charts[sensorType].destroy();
        }

        // 创建新图表
        console.log(`[Statistics] 创建新图表, 数据点数量: ${chartData.values.length}`);
        this.charts[sensorType] = new Chart(ctx, {
            type: 'line',
            data: {
                labels: chartData.labels,
                datasets: [{
                    label: `${config.name} (${config.unit})`,
                    data: chartData.values,
                    borderColor: config.color,
                    backgroundColor: this.hexToRgba(config.color, 0.1),
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4, // 平滑曲线
                    pointRadius: 3,
                    pointHoverRadius: 5,
                    pointBackgroundColor: config.color,
                    pointBorderColor: '#fff',
                    pointBorderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: true,
                        position: 'top',
                        labels: {
                            usePointStyle: true,
                            padding: 15,
                            font: {
                                size: 12,
                                family: '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif'
                            }
                        }
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                        backgroundColor: 'rgba(0, 0, 0, 0.8)',
                        titleColor: '#fff',
                        bodyColor: '#fff',
                        borderColor: config.color,
                        borderWidth: 1,
                        padding: 12,
                        displayColors: true,
                        callbacks: {
                            label: (context) => {
                                return ` ${context.dataset.label}: ${context.parsed.y}`;
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            display: false
                        },
                        ticks: {
                            maxRotation: 45,
                            minRotation: 45,
                            font: {
                                size: 10
                            }
                        }
                    },
                    y: {
                        display: true,
                        beginAtZero: false,
                        grid: {
                            color: '#e5e7eb',
                            borderDash: [5, 5]
                        },
                        ticks: {
                            font: {
                                size: 11
                            },
                            callback: (value) => {
                                return value.toFixed(1);
                            }
                        }
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                }
            }
        });
    },

    /**
     * 显示无数据状态
     */
    showNoData(sensorType) {
        console.log(`[Statistics] ${sensorType} 显示无数据状态`);

        // 转换为驼峰命名: liquid_level -> liquidLevel
        const camelCase = sensorType.replace(/_([a-z])/g, (m, p1) => p1.toUpperCase());

        // 安全地更新DOM元素
        const countEl = document.getElementById(`${camelCase}Count`);
        const minEl = document.getElementById(`${camelCase}Min`);
        const maxEl = document.getElementById(`${camelCase}Max`);
        const avgEl = document.getElementById(`${camelCase}Avg`);

        if (countEl) countEl.textContent = '0';
        if (minEl) minEl.textContent = '--';
        if (maxEl) maxEl.textContent = '--';
        if (avgEl) avgEl.textContent = '--';

        // 如果DOM元素缺失,记录警告
        if (!countEl || !minEl || !maxEl || !avgEl) {
            console.warn(`[Statistics] ${sensorType} DOM元素缺失:`, {
                count: !!countEl,
                min: !!minEl,
                max: !!maxEl,
                avg: !!avgEl
            });
        }

        // 清空图表
        const camelType = sensorType.replace(/_([a-z])/g, (m, p1) => p1.toUpperCase());
        const canvasId = `${camelType}Chart`;
        const canvas = document.getElementById(canvasId);

        if (canvas && this.charts[sensorType]) {
            this.charts[sensorType].destroy();
            delete this.charts[sensorType];
        }

        // 显示无数据提示
        if (canvas) {
            const ctx = canvas.getContext('2d');
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = '#9ca3af';
            ctx.font = '14px sans-serif';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText('暂无数据', canvas.width / 2, canvas.height / 2);
        }
    },

    /**
     * 十六进制颜色转RGBA
     */
    hexToRgba(hex, alpha) {
        const r = parseInt(hex.slice(1, 3), 16);
        const g = parseInt(hex.slice(3, 5), 16);
        const b = parseInt(hex.slice(5, 7), 16);
        return `rgba(${r}, ${g}, ${b}, ${alpha})`;
    },

    /**
     * 清理资源
     */
    destroy() {
        console.log('[Statistics] 清理资源...');

        // 取消WebSocket订阅
        if (this.unsubscribeSensorData) {
            this.unsubscribeSensorData();
            this.unsubscribeSensorData = null;
        }
        
        if (this.unsubscribeConnection) {
            this.unsubscribeConnection();
            this.unsubscribeConnection = null;
        }

        // 销毁所有图表
        Object.values(this.charts).forEach(chart => {
            if (chart) chart.destroy();
        });
        this.charts = {};

        // 清理数据缓存
        this.rawDataCache = {};

        // 清理所有正在进行的请求记录
        this.loadingRequests.clear();

        // 重置初始化标志,允许下次重新初始化
        this.isInitialized = false;

        console.log('[Statistics] 资源清理完成');
    }
};

// 导出模块
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Statistics;
}
