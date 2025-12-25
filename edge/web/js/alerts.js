/**
 * 告警管理模块
 * 处理传感器告警监测与处理功能
 * 使用WebSocket接收实时告警推送
 * @version 2.0.0
 */

const Alerts = {
    initialized: false, // 防止重复初始化
    currentFilters: {
        severity: '',
        resolved: 'false',
        page: 1,
        limit: 20
    },
    totalPages: 1,
    totalAlerts: 0,
    selectedAlerts: new Set(), // 存储选中的告警ID
    
    // WebSocket订阅取消函数
    unsubscribeAlert: null,
    unsubscribeConnection: null,
    
    // 自动刷新定时器(已弃用,改用WebSocket)
    autoRefreshTimer: null,

    /**
     * 初始化告警管理模块
     */
    async init() {
        console.log('[Alerts] 初始化告警管理模块');

        // 🔥 防止重复初始化 - 事件监听器只绑定一次
        if (!this.initialized) {
            console.log('[Alerts] 首次初始化，绑定事件监听器');
            this.bindEvents();
            
            // 订阅WebSocket消息
            console.log('[Alerts] 订阅WebSocket消息...');
            this.subscribeWebSocket();
            
            this.initialized = true;
        } else {
            console.log('[Alerts] 已初始化，跳过事件绑定');
        }

        // 每次切换页面时都重新加载数据
        await this.loadAlerts();
        await this.updateUnresolvedCount();

        // 注意: 不再启动自动刷新,改用WebSocket实时推送
        // this.startAutoRefresh();
    },

    /**
     * 订阅WebSocket消息
     */
    subscribeWebSocket() {
        if (typeof WebSocketManager === 'undefined') {
            console.warn('[Alerts] WebSocketManager未定义，回退到轮询模式');
            this.startAutoRefresh();
            return;
        }

        // 订阅告警消息
        this.unsubscribeAlert = WebSocketManager.subscribe('alert', (data) => {
            this.handleRealtimeAlert(data);
        });

        // 订阅连接状态变化
        this.unsubscribeConnection = WebSocketManager.subscribe('connection', (data) => {
            if (data.status === 'connected') {
                console.log('[Alerts] WebSocket重连，刷新告警数据');
                this.loadAlerts();
                this.updateUnresolvedCount();
            }
        });

        console.log('[Alerts] WebSocket订阅完成');
    },

    /**
     * 处理实时告警推送
     * @param {Object} data - 告警数据
     */
    handleRealtimeAlert(data) {
        console.log('[Alerts] 收到实时告警:', data);
        
        // 显示Toast通知
        if (typeof UI !== 'undefined' && UI.showToast) {
            const severityText = {
                'critical': '🚨 严重',
                'high': '⚠️ 高危',
                'medium': '⚡ 中等',
                'low': 'ℹ️ 低'
            };
            const prefix = severityText[data.severity] || '📢';
            UI.showToast(`${prefix} ${data.message}`, 'warning', 8000);
        }

        // 更新未解决告警数量
        this.updateUnresolvedCount();

        // 如果当前页面是告警管理页面,刷新列表
        const alertsPage = document.getElementById('alertsPage');
        if (alertsPage && alertsPage.classList.contains('active')) {
            // 如果在第一页且筛选条件为未解决告警,则插入新告警到列表顶部
            if (this.currentFilters.page === 1 && this.currentFilters.resolved === 'false') {
                this.insertNewAlert(data);
            } else {
                // 否则重新加载列表
                this.loadAlerts();
            }
        }
    },

    /**
     * 插入新告警到列表顶部
     * @param {Object} alert - 告警数据
     */
    insertNewAlert(alert) {
        const tbody = document.getElementById('alertsTableBody');
        if (!tbody) return;

        // 如果当前显示的是空状态,重新加载
        if (tbody.querySelector('.empty-state-small')) {
            this.loadAlerts();
            return;
        }

        // 创建新行
        const newRow = document.createElement('tr');
        newRow.className = 'alert-row alert-unresolved';
        newRow.setAttribute('data-severity', alert.severity);
        newRow.setAttribute('data-alert-id', alert.id);
        newRow.innerHTML = `
            <td>
                <input type="checkbox" class="alert-checkbox" data-alert-id="${alert.id}">
            </td>
            <td>${this.getSeverityBadge(alert.severity)}</td>
            <td>
                <div class="alert-device">
                    ${this.getAlertTypeIcon(alert.alert_type)}
                    <span class="device-id" title="${alert.device_id}">${alert.device_id}</span>
                </div>
            </td>
            <td>
                <div class="alert-message">
                    <strong>${alert.message}</strong>
                </div>
            </td>
            <td class="text-right">
                <span class="alert-value">${alert.value ? alert.value.toFixed(2) : '--'}</span>
            </td>
            <td class="text-right">
                <span class="alert-threshold">${alert.threshold ? alert.threshold.toFixed(2) : '--'}</span>
            </td>
            <td>
                <div class="alert-time">
                    <div>${this.formatDateTime(alert.timestamp)}</div>
                    <small class="text-muted">${new Date(alert.timestamp).toLocaleString('zh-CN')}</small>
                </div>
            </td>
            <td>${this.getResolvedBadge(false)}</td>
            <td>
                <button class="btn btn-sm btn-success" onclick="Alerts.resolveAlert(${alert.id})" title="标记为已解决">
                    <i class="fas fa-check"></i> 解决
                </button>
            </td>
        `;

        // 添加入场动画
        newRow.style.animation = 'slideIn 0.3s ease';
        newRow.style.backgroundColor = 'rgba(245, 158, 11, 0.2)';
        
        // 插入到列表顶部
        tbody.insertBefore(newRow, tbody.firstChild);

        // 绑定复选框事件
        const checkbox = newRow.querySelector('.alert-checkbox');
        if (checkbox) {
            checkbox.addEventListener('change', (e) => {
                const alertId = parseInt(e.target.dataset.alertId);
                if (e.target.checked) {
                    this.selectedAlerts.add(alertId);
                } else {
                    this.selectedAlerts.delete(alertId);
                }
                this.updateBatchResolveButton();
                this.updateSelectAllCheckbox();
            });
        }

        // 背景色渐变消失
        setTimeout(() => {
            newRow.style.backgroundColor = '';
        }, 2000);

        // 更新总数
        this.totalAlerts++;
        console.log('[Alerts] 新告警已插入列表');
    },

    /**
     * 绑定事件监听器
     */
    bindEvents() {
        // 筛选表单提交
        const filterForm = document.getElementById('alertsFilterForm');
        if (filterForm) {
            filterForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.currentFilters.page = 1;
                this.loadAlerts();
            });
        }

        // 刷新按钮
        const refreshBtn = document.getElementById('refreshAlertsBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.loadAlerts();
                this.updateUnresolvedCount();
            });
        }

        // 全选复选框
        const selectAllCheckbox = document.getElementById('selectAllAlerts');
        if (selectAllCheckbox) {
            selectAllCheckbox.addEventListener('change', (e) => {
                this.toggleSelectAll(e.target.checked);
            });
        }

        // 批量解决按钮
        const batchResolveBtn = document.getElementById('batchResolveAlertsBtn');
        if (batchResolveBtn) {
            batchResolveBtn.addEventListener('click', () => {
                this.batchResolveAlerts();
            });
        }

        // 分页按钮
        const prevBtn = document.getElementById('alertsPrevBtn');
        const nextBtn = document.getElementById('alertsNextBtn');

        if (prevBtn) {
            prevBtn.addEventListener('click', () => {
                if (this.currentFilters.page > 1) {
                    this.currentFilters.page--;
                    this.loadAlerts();
                }
            });
        }

        if (nextBtn) {
            nextBtn.addEventListener('click', () => {
                if (this.currentFilters.page < this.totalPages) {
                    this.currentFilters.page++;
                    this.loadAlerts();
                }
            });
        }
    },

    /**
     * 加载告警列表
     */
    async loadAlerts() {
        try {
            UI.showLoading();

            // 获取筛选条件
            this.currentFilters.severity = document.getElementById('alertsSeverity').value;
            this.currentFilters.resolved = document.getElementById('alertsResolved').value;

            console.log('[Alerts] 加载告警列表:', this.currentFilters);

            // 调用API获取告警
            const result = await API.getAlerts(
                this.currentFilters.page,
                this.currentFilters.limit,
                this.currentFilters.severity,
                this.currentFilters.resolved
            );

            console.log('[Alerts] 告警数据:', result);

            // 更新总数和总页数
            this.totalAlerts = result.total;
            this.totalPages = Math.ceil(result.total / this.currentFilters.limit);

            // 渲染告警列表
            this.renderAlerts(result.alerts);

            // 更新分页显示
            this.updatePagination();

        } catch (error) {
            console.error('[Alerts] 加载告警失败:', error);
            UI.showToast('加载告警失败: ' + error.message, 'error');

            // 显示错误状态
            const tbody = document.getElementById('alertsTableBody');
            tbody.innerHTML = `
                <tr>
                    <td colspan="8" class="text-center">
                        <div class="empty-state-small">
                            <i class="fas fa-exclamation-circle"></i>
                            <p>加载失败：${error.message}</p>
                        </div>
                    </td>
                </tr>
            `;
        } finally {
            UI.hideLoading();
        }
    },

    /**
     * 渲染告警列表
     */
    renderAlerts(alerts) {
        const tbody = document.getElementById('alertsTableBody');

        if (!alerts || alerts.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="9" class="text-center">
                        <div class="empty-state-small">
                            <i class="fas fa-inbox"></i>
                            <p>没有符合条件的告警</p>
                        </div>
                    </td>
                </tr>
            `;
            this.updateBatchResolveButton();
            return;
        }

        tbody.innerHTML = alerts.map(alert => `
            <tr class="alert-row ${!alert.resolved ? 'alert-unresolved' : ''}" data-severity="${alert.severity}" data-alert-id="${alert.id}">
                <td>
                    ${!alert.resolved ? `<input type="checkbox" class="alert-checkbox" data-alert-id="${alert.id}" ${this.selectedAlerts.has(alert.id) ? 'checked' : ''}>` : ''}
                </td>
                <td>${this.getSeverityBadge(alert.severity)}</td>
                <td>
                    <div class="alert-device">
                        ${this.getAlertTypeIcon(alert.alert_type)}
                        <span class="device-id" title="${alert.device_id}">${alert.device_id}</span>
                    </div>
                </td>
                <td>
                    <div class="alert-message">
                        <strong>${alert.message}</strong>
                    </div>
                </td>
                <td class="text-right">
                    <span class="alert-value">${alert.value.toFixed(2)}</span>
                </td>
                <td class="text-right">
                    <span class="alert-threshold">${alert.threshold.toFixed(2)}</span>
                </td>
                <td>
                    <div class="alert-time">
                        <div>${this.formatDateTime(alert.timestamp)}</div>
                        <small class="text-muted">${new Date(alert.timestamp).toLocaleString('zh-CN')}</small>
                    </div>
                </td>
                <td>${this.getResolvedBadge(alert.resolved, alert.resolved_at)}</td>
                <td>
                    ${!alert.resolved ? `
                        <button class="btn btn-sm btn-success" onclick="Alerts.resolveAlert(${alert.id})" title="标记为已解决">
                            <i class="fas fa-check"></i> 解决
                        </button>
                    ` : '<span class="text-muted">-</span>'}
                </td>
            </tr>
        `).join('');

        // 绑定复选框事件
        document.querySelectorAll('.alert-checkbox').forEach(checkbox => {
            checkbox.addEventListener('change', (e) => {
                const alertId = parseInt(e.target.dataset.alertId);
                if (e.target.checked) {
                    this.selectedAlerts.add(alertId);
                } else {
                    this.selectedAlerts.delete(alertId);
                }
                this.updateBatchResolveButton();
                this.updateSelectAllCheckbox();
            });
        });

        this.updateBatchResolveButton();
        this.updateSelectAllCheckbox();
    },

    /**
     * 更新分页显示
     */
    updatePagination() {
        const pagination = document.getElementById('alertsPagination');
        const currentPageSpan = document.getElementById('alertsCurrentPage');
        const totalPagesSpan = document.getElementById('alertsTotalPages');
        const prevBtn = document.getElementById('alertsPrevBtn');
        const nextBtn = document.getElementById('alertsNextBtn');

        if (this.totalPages > 0) {
            pagination.style.display = 'flex';
            currentPageSpan.textContent = this.currentFilters.page;
            totalPagesSpan.textContent = this.totalPages;

            // 更新按钮状态
            prevBtn.disabled = this.currentFilters.page <= 1;
            nextBtn.disabled = this.currentFilters.page >= this.totalPages;
        } else {
            pagination.style.display = 'none';
        }
    },

    /**
     * 解决告警
     */
    async resolveAlert(alertId) {
        if (!confirm('确认将此告警标记为已解决？')) {
            return;
        }

        try {
            UI.showLoading();

            console.log('[Alerts] 解决告警:', alertId);

            await API.resolveAlert(alertId);

            UI.showToast('告警已标记为已解决', 'success');

            // 重新加载告警列表
            await this.loadAlerts();

            // 更新未解决告警数量
            await this.updateUnresolvedCount();

        } catch (error) {
            console.error('[Alerts] 解决告警失败:', error);
            UI.showToast('解决告警失败: ' + error.message, 'error');
        } finally {
            UI.hideLoading();
        }
    },

    /**
     * 更新未解决告警数量
     */
    async updateUnresolvedCount() {
        try {
            const result = await API.getAlerts(1, 1, '', 'false');
            const count = result.total;

            // 更新告警页面的徽章
            const badge = document.getElementById('unresolvedAlertsCount');
            if (badge) {
                badge.textContent = count;
                badge.className = count > 0 ? 'badge badge-danger' : 'badge badge-success';
            }

            // 更新顶部导航栏的通知徽章
            const topBarBadge = document.getElementById('topBarAlertCount');
            if (topBarBadge) {
                if (count > 0) {
                    topBarBadge.textContent = count > 99 ? '99+' : count;
                    topBarBadge.classList.add('has-alerts');
                } else {
                    topBarBadge.textContent = '';
                    topBarBadge.classList.remove('has-alerts');
                }
            }

            console.log('[Alerts] 未解决告警数量:', count);
        } catch (error) {
            console.error('[Alerts] 更新未解决告警数量失败:', error);
        }
    },

    /**
     * 启动自动刷新(备用方案,WebSocket不可用时使用)
     */
    startAutoRefresh() {
        // 清除已有的定时器
        if (this.autoRefreshTimer) {
            clearInterval(this.autoRefreshTimer);
        }
        
        this.autoRefreshTimer = setInterval(() => {
            // 如果当前页面是告警管理页面，则自动刷新
            const alertsPage = document.getElementById('alertsPage');
            if (alertsPage && alertsPage.classList.contains('active')) {
                console.log('[Alerts] 自动刷新告警列表(轮询模式)');
                this.loadAlerts();
                this.updateUnresolvedCount();
            }
        }, 60000); // 60秒刷新一次
        
        console.log('[Alerts] 启动自动刷新(轮询模式)');
    },

    /**
     * 停止自动刷新
     */
    stopAutoRefresh() {
        if (this.autoRefreshTimer) {
            clearInterval(this.autoRefreshTimer);
            this.autoRefreshTimer = null;
        }
    },

    /**
     * 清理资源
     */
    destroy() {
        console.log('[Alerts] 清理资源...');
        
        // 停止自动刷新
        this.stopAutoRefresh();
        
        // 取消WebSocket订阅
        if (this.unsubscribeAlert) {
            this.unsubscribeAlert();
            this.unsubscribeAlert = null;
        }
        
        if (this.unsubscribeConnection) {
            this.unsubscribeConnection();
            this.unsubscribeConnection = null;
        }
        
        // 清空选中状态
        this.selectedAlerts.clear();
        
        console.log('[Alerts] 资源清理完成');
    },

    /**
     * 获取严重程度徽章
     */
    getSeverityBadge(severity) {
        const badges = {
            'critical': '<span class="badge badge-critical"><i class="fas fa-skull-crossbones"></i> 严重</span>',
            'high': '<span class="badge badge-danger"><i class="fas fa-exclamation-triangle"></i> 高危</span>',
            'medium': '<span class="badge badge-warning"><i class="fas fa-exclamation-circle"></i> 中等</span>',
            'low': '<span class="badge badge-info"><i class="fas fa-info-circle"></i> 低</span>'
        };
        return badges[severity] || severity;
    },

    /**
     * 获取告警类型图标
     */
    getAlertTypeIcon(alertType) {
        const icons = {
            'co2_high': '<i class="fas fa-wind" style="color: #f59e0b;"></i>',
            'co_high': '<i class="fas fa-smog" style="color: #ef4444;"></i>',
            'smoke_detected': '<i class="fas fa-fire" style="color: #dc2626;"></i>',
            'liquid_level_low': '<i class="fas fa-tint-slash" style="color: #3b82f6;"></i>',
            'liquid_level_high': '<i class="fas fa-tint" style="color: #3b82f6;"></i>',
            'conductivity_abnormal': '<i class="fas fa-bolt" style="color: #8b5cf6;"></i>',
            'temperature_low': '<i class="fas fa-thermometer-empty" style="color: #06b6d4;"></i>',
            'temperature_high': '<i class="fas fa-thermometer-full" style="color: #f97316;"></i>',
            'flow_abnormal': '<i class="fas fa-water" style="color: #0ea5e9;"></i>'
        };
        return icons[alertType] || '<i class="fas fa-bell"></i>';
    },

    /**
     * 获取解决状态徽章
     */
    getResolvedBadge(resolved, resolvedAt) {
        if (resolved) {
            return `<span class="badge badge-success" title="已解决${resolvedAt ? ': ' + this.formatDateTime(resolvedAt) : ''}"><i class="fas fa-check"></i> 已解决</span>`;
        } else {
            return `<span class="badge badge-secondary"><i class="fas fa-clock"></i> 未解决</span>`;
        }
    },

    /**
     * 格式化日期时间
     */
    formatDateTime(dateTime) {
        if (!dateTime) return '-';

        const date = new Date(dateTime);
        const now = new Date();
        const diff = now - date;

        // 如果在1小时内，显示相对时间
        if (diff < 3600000) {
            const minutes = Math.floor(diff / 60000);
            return `${minutes}分钟前`;
        }

        // 如果在24小时内，显示小时
        if (diff < 86400000) {
            const hours = Math.floor(diff / 3600000);
            return `${hours}小时前`;
        }

        // 否则显示完整时间
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    },

    /**
     * 全选/取消全选
     */
    toggleSelectAll(checked) {
        const checkboxes = document.querySelectorAll('.alert-checkbox');
        checkboxes.forEach(checkbox => {
            checkbox.checked = checked;
            const alertId = parseInt(checkbox.dataset.alertId);
            if (checked) {
                this.selectedAlerts.add(alertId);
            } else {
                this.selectedAlerts.delete(alertId);
            }
        });
        this.updateBatchResolveButton();
    },

    /**
     * 更新全选复选框状态
     */
    updateSelectAllCheckbox() {
        const selectAllCheckbox = document.getElementById('selectAllAlerts');
        const checkboxes = document.querySelectorAll('.alert-checkbox');

        if (checkboxes.length === 0) {
            selectAllCheckbox.checked = false;
            selectAllCheckbox.indeterminate = false;
            return;
        }

        const checkedCount = Array.from(checkboxes).filter(cb => cb.checked).length;

        if (checkedCount === 0) {
            selectAllCheckbox.checked = false;
            selectAllCheckbox.indeterminate = false;
        } else if (checkedCount === checkboxes.length) {
            selectAllCheckbox.checked = true;
            selectAllCheckbox.indeterminate = false;
        } else {
            selectAllCheckbox.checked = false;
            selectAllCheckbox.indeterminate = true;
        }
    },

    /**
     * 更新批量解决按钮显示状态
     */
    updateBatchResolveButton() {
        const btn = document.getElementById('batchResolveAlertsBtn');
        if (btn) {
            if (this.selectedAlerts.size > 0) {
                btn.style.display = 'block';
                btn.innerHTML = `<i class="fas fa-check-double"></i> 批量解决 (${this.selectedAlerts.size})`;
            } else {
                btn.style.display = 'none';
            }
        }
    },

    /**
     * 批量解决告警
     */
    async batchResolveAlerts() {
        if (this.selectedAlerts.size === 0) {
            UI.showToast('请先选择要解决的告警', 'warning');
            return;
        }

        const count = this.selectedAlerts.size;
        if (!confirm(`确认批量解决 ${count} 条告警吗?`)) {
            return;
        }

        try {
            UI.showLoading();

            const alertIds = Array.from(this.selectedAlerts);
            console.log('[Alerts] 批量解决告警:', alertIds);

            // 逐个调用解决API (因为后端目前只有单个解决API)
            let successCount = 0;
            let failCount = 0;

            for (const alertId of alertIds) {
                try {
                    await API.resolveAlert(alertId);
                    successCount++;
                } catch (error) {
                    console.error(`[Alerts] 解决告警 ${alertId} 失败:`, error);
                    failCount++;
                }
            }

            // 清空选中状态
            this.selectedAlerts.clear();

            // 显示结果
            if (failCount === 0) {
                UI.showToast(`成功解决 ${successCount} 条告警`, 'success');
            } else {
                UI.showToast(`成功解决 ${successCount} 条告警, ${failCount} 条失败`, 'warning');
            }

            // 重新加载告警列表
            await this.loadAlerts();

            // 更新未解决告警数量
            await this.updateUnresolvedCount();

        } catch (error) {
            console.error('[Alerts] 批量解决告警失败:', error);
            UI.showToast('批量解决告警失败: ' + error.message, 'error');
        } finally {
            UI.hideLoading();
        }
    }
};
