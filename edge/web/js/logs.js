/**
 * 日志记录模块
 * 处理告警日志和认证日志的展示与筛选
 * @version 1.0.0
 */

const Logs = {
    // 当前激活的Tab
    activeTab: 'alert-logs',

    // 告警日志筛选条件
    alertLogsFilters: {
        startDate: '',
        endDate: '',
        severity: '',
        resolved: '',
        deviceID: '',
        page: 1,
        limit: 20
    },

    // 认证日志筛选条件
    authLogsFilters: {
        startDate: '',
        endDate: '',
        status: '',
        deviceID: '',
        page: 1,
        limit: 20
    },

    // 分页信息
    alertLogsTotalPages: 1,
    authLogsTotalPages: 1,

    // 标志位：防止重复绑定事件
    eventsInitialized: false,

    /**
     * 初始化日志模块
     */
    async init() {
        console.log('[Logs] 初始化日志模块');

        // 只在第一次时绑定事件
        if (!this.eventsInitialized) {
            this.bindEvents();
            this.eventsInitialized = true;
        }

        // 设置默认日期（最近7天）
        this.setDefaultDates();

        // 加载当前Tab的日志
        await this.loadCurrentTabLogs();
    },

    /**
     * 绑定事件监听器
     */
    bindEvents() {
        // Tab切换
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tab = e.currentTarget.dataset.tab;
                this.switchTab(tab);
            });
        });

        // 告警日志筛选表单
        const alertForm = document.getElementById('alertLogsFilterForm');
        if (alertForm) {
            alertForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.alertLogsFilters.page = 1;
                this.loadAlertLogs();
            });
        }

        // 认证日志筛选表单
        const authForm = document.getElementById('authLogsFilterForm');
        if (authForm) {
            authForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.authLogsFilters.page = 1;
                this.loadAuthLogs();
            });
        }

        // 重置按钮
        document.getElementById('resetAlertLogsBtn')?.addEventListener('click', () => {
            this.resetAlertLogsFilters();
        });

        document.getElementById('resetAuthLogsBtn')?.addEventListener('click', () => {
            this.resetAuthLogsFilters();
        });

        // 分页按钮 - 告警日志
        document.getElementById('alertLogsPrevBtn')?.addEventListener('click', () => {
            if (this.alertLogsFilters.page > 1) {
                this.alertLogsFilters.page--;
                this.loadAlertLogs();
            }
        });

        document.getElementById('alertLogsNextBtn')?.addEventListener('click', () => {
            if (this.alertLogsFilters.page < this.alertLogsTotalPages) {
                this.alertLogsFilters.page++;
                this.loadAlertLogs();
            }
        });

        // 分页按钮 - 认证日志
        document.getElementById('authLogsPrevBtn')?.addEventListener('click', () => {
            if (this.authLogsFilters.page > 1) {
                this.authLogsFilters.page--;
                this.loadAuthLogs();
            }
        });

        document.getElementById('authLogsNextBtn')?.addEventListener('click', () => {
            if (this.authLogsFilters.page < this.authLogsTotalPages) {
                this.authLogsFilters.page++;
                this.loadAuthLogs();
            }
        });

        // 全选/取消全选
        document.getElementById('selectAllAlertLogs')?.addEventListener('change', (e) => {
            const checkboxes = document.querySelectorAll('.alert-checkbox');
            checkboxes.forEach(cb => cb.checked = e.target.checked);
            this.updateBatchDeleteButton();
        });

        // 单个复选框变化
        document.addEventListener('change', (e) => {
            if (e.target.classList.contains('alert-checkbox')) {
                this.updateBatchDeleteButton();
            }
            if (e.target.classList.contains('auth-checkbox')) {
                this.updateBatchDeleteAuthButton();
            }
        });

        // 批量删除按钮 - 告警日志
        document.getElementById('batchDeleteAlertLogsBtn')?.addEventListener('click', () => {
            this.batchDeleteAlertLogs();
        });

        // 全选/取消全选 - 认证日志
        document.getElementById('selectAllAuthLogs')?.addEventListener('change', (e) => {
            const checkboxes = document.querySelectorAll('.auth-checkbox');
            checkboxes.forEach(cb => cb.checked = e.target.checked);
            this.updateBatchDeleteAuthButton();
        });

        // 批量删除按钮 - 认证日志
        document.getElementById('batchDeleteAuthLogsBtn')?.addEventListener('click', () => {
            this.batchDeleteAuthLogs();
        });

        // 一键清空按钮 - 认证日志
        document.getElementById('clearAllAuthLogsBtn')?.addEventListener('click', () => {
            this.clearAllAuthLogs();
        });
    },

    /**
     * 设置默认日期（最近7天）
     */
    setDefaultDates() {
        const today = new Date();
        const sevenDaysAgo = new Date(today);
        sevenDaysAgo.setDate(today.getDate() - 7);

        const formatDate = (date) => {
            return date.toISOString().split('T')[0];
        };

        // 告警日志日期
        document.getElementById('alertLogStartDate').value = formatDate(sevenDaysAgo);
        document.getElementById('alertLogEndDate').value = formatDate(today);

        // 认证日志日期
        document.getElementById('authLogStartDate').value = formatDate(sevenDaysAgo);
        document.getElementById('authLogEndDate').value = formatDate(today);
    },

    /**
     * 切换Tab
     */
    switchTab(tabName) {
        console.log('[Logs] 切换Tab:', tabName);

        // 更新激活状态
        this.activeTab = tabName;

        // 更新Tab按钮样式
        document.querySelectorAll('.tab-btn').forEach(btn => {
            if (btn.dataset.tab === tabName) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });

        // 更新Tab内容显示
        document.querySelectorAll('.tab-content').forEach(content => {
            if (content.id === `${tabName}-tab`) {
                content.classList.add('active');
            } else {
                content.classList.remove('active');
            }
        });

        // 加载对应的日志
        this.loadCurrentTabLogs();
    },

    /**
     * 加载当前Tab的日志
     */
    async loadCurrentTabLogs() {
        if (this.activeTab === 'alert-logs') {
            await this.loadAlertLogs();
        } else if (this.activeTab === 'auth-logs') {
            await this.loadAuthLogs();
        }
    },

    /**
     * 加载告警日志
     */
    async loadAlertLogs() {
        try {
            UI.showLoading();

            // 获取筛选条件
            this.alertLogsFilters.startDate = document.getElementById('alertLogStartDate').value;
            this.alertLogsFilters.endDate = document.getElementById('alertLogEndDate').value;
            this.alertLogsFilters.severity = document.getElementById('alertLogSeverity').value;
            this.alertLogsFilters.resolved = document.getElementById('alertLogResolved').value;
            this.alertLogsFilters.deviceID = document.getElementById('alertLogDeviceID').value;

            console.log('[Logs] 加载告警日志:', this.alertLogsFilters);

            // 调用API
            const result = await API.getAlertLogs(this.alertLogsFilters);

            console.log('[Logs] 告警日志结果:', result);

            // 更新表格
            this.renderAlertLogsTable(result.logs || []);

            // 更新分页
            this.alertLogsTotalPages = Math.ceil(result.total / this.alertLogsFilters.limit);
            this.updateAlertLogsPagination(result.total);

            UI.hideLoading();
        } catch (error) {
            console.error('[Logs] 加载告警日志失败:', error);
            UI.showToast('加载告警日志失败: ' + error.message, 'error');
            UI.hideLoading();
        }
    },

    /**
     * 渲染告警日志表格
     */
    renderAlertLogsTable(logs) {
        const tbody = document.getElementById('alertLogsTableBody');

        if (!logs || logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="10" class="text-center">暂无数据</td></tr>';
            return;
        }

        tbody.innerHTML = logs.map(log => {
            const severityClass = this.getSeverityClass(log.severity);
            const statusBadge = log.resolved
                ? '<span class="badge badge-success">已解决</span>'
                : '<span class="badge badge-warning">未解决</span>';

            const resolvedTime = log.resolved_at
                ? this.formatDateTime(log.resolved_at)
                : '-';

            // 处理空的device_id和alert_type
            const deviceId = log.device_id && log.device_id.trim() !== '' 
                ? log.device_id 
                : this.inferDeviceIdFromMessage(log.message);
            
            const alertType = log.alert_type && log.alert_type.trim() !== '' 
                ? log.alert_type 
                : this.inferAlertTypeFromMessage(log.message);

            // 安全处理value和threshold（可能为null）
            const valueDisplay = (log.value != null && log.value !== undefined)
                ? log.value.toFixed(2)
                : '-';
            const thresholdDisplay = (log.threshold != null && log.threshold !== undefined)
                ? log.threshold.toFixed(2)
                : '-';

            return `
                <tr data-alert-id="${log.id}">
                    <td><input type="checkbox" class="alert-checkbox" value="${log.id}"></td>
                    <td>${log.id}</td>
                    <td><code>${deviceId}</code></td>
                    <td>${this.getAlertTypeLabel(alertType)}</td>
                    <td><span class="badge ${severityClass}">${this.getSeverityLabel(log.severity)}</span></td>
                    <td>${valueDisplay}</td>
                    <td>${thresholdDisplay}</td>
                    <td>${this.formatDateTime(log.timestamp)}</td>
                    <td>${statusBadge}</td>
                    <td>${resolvedTime}</td>
                </tr>
            `;
        }).join('');
    },

    /**
     * 加载认证日志
     */
    async loadAuthLogs() {
        try {
            UI.showLoading();

            // 获取筛选条件
            this.authLogsFilters.startDate = document.getElementById('authLogStartDate').value;
            this.authLogsFilters.endDate = document.getElementById('authLogEndDate').value;
            this.authLogsFilters.status = document.getElementById('authLogStatus').value;
            this.authLogsFilters.deviceID = document.getElementById('authLogDeviceID').value;

            console.log('[Logs] 加载认证日志:', this.authLogsFilters);

            // 调用API
            const result = await API.getAuthLogs(this.authLogsFilters);

            console.log('[Logs] 认证日志结果:', result);

            // 更新表格
            this.renderAuthLogsTable(result.logs || []);

            // 更新分页
            this.authLogsTotalPages = Math.ceil(result.total / this.authLogsFilters.limit);
            this.updateAuthLogsPagination(result.total);

            UI.hideLoading();
        } catch (error) {
            console.error('[Logs] 加载认证日志失败:', error);
            UI.showToast('加载认证日志失败: ' + error.message, 'error');
            UI.hideLoading();
        }
    },

    /**
     * 渲染认证日志表格
     */
    renderAuthLogsTable(logs) {
        const tbody = document.getElementById('authLogsTableBody');

        if (!logs || logs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="8" class="text-center">暂无数据</td></tr>';
            return;
        }

        tbody.innerHTML = logs.map(log => {
            const statusBadge = log.status === 'success'
                ? '<span class="badge badge-success">成功</span>'
                : '<span class="badge badge-secondary">待验证</span>';

            const actionLabel = this.getActionLabel(log.action);
            
            // 会话ID显示优化：有ID则显示前8位，无ID则显示更友好的提示
            let sessionIDDisplay;
            if (log.session_id && log.session_id.trim() !== '') {
                sessionIDDisplay = `<code title="${log.session_id}">${log.session_id.substring(0, 8)}...</code>`;
            } else {
                sessionIDDisplay = '<span style="color: #999;">无会话</span>';
            }

            return `
                <tr>
                    <td><input type="checkbox" class="auth-checkbox" value="${log.id}" /></td>
                    <td><code>${log.id.substring(0, 8)}...</code></td>
                    <td><code>${log.device_id}</code></td>
                    <td>${actionLabel}</td>
                    <td>${statusBadge}</td>
                    <td>${this.formatDateTime(log.timestamp)}</td>
                    <td>${sessionIDDisplay}</td>
                    <td>${log.details}</td>
                </tr>
            `;
        }).join('');
    },

    /**
     * 更新告警日志分页
     */
    updateAlertLogsPagination(total) {
        document.getElementById('alertLogsPaginationInfo').textContent = `共 ${total} 条记录`;
        document.getElementById('alertLogsCurrentPage').textContent = `第 ${this.alertLogsFilters.page} 页`;

        document.getElementById('alertLogsPrevBtn').disabled = this.alertLogsFilters.page === 1;
        document.getElementById('alertLogsNextBtn').disabled = this.alertLogsFilters.page >= this.alertLogsTotalPages;
    },

    /**
     * 更新认证日志分页
     */
    updateAuthLogsPagination(total) {
        document.getElementById('authLogsPaginationInfo').textContent = `共 ${total} 条记录`;
        document.getElementById('authLogsCurrentPage').textContent = `第 ${this.authLogsFilters.page} 页`;

        document.getElementById('authLogsPrevBtn').disabled = this.authLogsFilters.page === 1;
        document.getElementById('authLogsNextBtn').disabled = this.authLogsFilters.page >= this.authLogsTotalPages;
    },

    /**
     * 重置告警日志筛选
     */
    resetAlertLogsFilters() {
        this.setDefaultDates();
        document.getElementById('alertLogSeverity').value = '';
        document.getElementById('alertLogResolved').value = '';
        document.getElementById('alertLogDeviceID').value = '';
        this.alertLogsFilters.page = 1;
        this.loadAlertLogs();
    },

    /**
     * 重置认证日志筛选
     */
    resetAuthLogsFilters() {
        this.setDefaultDates();
        document.getElementById('authLogStatus').value = '';
        document.getElementById('authLogDeviceID').value = '';
        this.authLogsFilters.page = 1;
        this.loadAuthLogs();
    },

    /**
     * 格式化日期时间
     */
    formatDateTime(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
    },

    /**
     * 获取严重级别样式类
     */
    getSeverityClass(severity) {
        const classes = {
            'critical': 'badge-danger',
            'high': 'badge-warning',
            'medium': 'badge-info',
            'low': 'badge-secondary'
        };
        return classes[severity] || 'badge-secondary';
    },

    /**
     * 获取严重级别标签
     */
    getSeverityLabel(severity) {
        const labels = {
            'critical': '严重',
            'high': '高',
            'medium': '中',
            'low': '低'
        };
        return labels[severity] || severity;
    },

    /**
     * 获取告警类型标签
     */
    getAlertTypeLabel(type) {
        const labels = {
            'co2_high': 'CO2超标',
            'co_high': 'CO超标',
            'smoke_detected': '烟雾检测',
            'liquid_level_low': '液位过低',
            'liquid_level_high': '液位过高',
            'conductivity_low': '电导率过低',
            'conductivity_high': '电导率过高',
            'temperature_low': '温度过低',
            'temperature_high': '温度过高',
            'flow_low': '流速过低',
            'flow_high': '流速过高'
        };
        return labels[type] || type;
    },

    /**
     * 获取操作类型标签
     */
    getActionLabel(action) {
        const labels = {
            'challenge_requested': '请求认证挑战',
            'challenge_used': '挑战已使用',
            'session_created': '会话创建成功'
        };
        return labels[action] || action;
    },

    /**
     * 更新批量删除按钮状态
     */
    updateBatchDeleteButton() {
        const checkboxes = document.querySelectorAll('.alert-checkbox:checked');
        const actionsDiv = document.getElementById('batchActionsAlert');
        const countSpan = document.getElementById('selectedAlertLogsCount');
        
        if (checkboxes.length > 0) {
            actionsDiv.style.display = 'block';
            countSpan.textContent = `已选择 ${checkboxes.length} 条记录`;
        } else {
            actionsDiv.style.display = 'none';
            countSpan.textContent = '';
        }
    },

    /**
     * 批量删除告警日志
     */
    async batchDeleteAlertLogs() {
        const checkboxes = document.querySelectorAll('.alert-checkbox:checked');
        const ids = Array.from(checkboxes).map(cb => parseInt(cb.value));
        
        if (ids.length === 0) {
            UI.showToast('请选择要删除的日志', 'warning');
            return;
        }

        if (!confirm(`确定要删除选中的 ${ids.length} 条告警日志吗？此操作不可恢复！`)) {
            return;
        }

        try {
            UI.showLoading();
            
            await API.batchDeleteAlertLogs(ids);
            
            UI.showToast(`成功删除 ${ids.length} 条告警日志`, 'success');
            
            // 取消全选
            document.getElementById('selectAllAlertLogs').checked = false;
            
            // 重新加载当前页
            await this.loadAlertLogs();
            
            UI.hideLoading();
        } catch (error) {
            console.error('[Logs] 批量删除告警日志失败:', error);
            UI.showToast('批量删除失败: ' + error.message, 'error');
            UI.hideLoading();
        }
    },

    /**
     * 更新批量删除认证日志按钮状态
     */
    updateBatchDeleteAuthButton() {
        const checkboxes = document.querySelectorAll('.auth-checkbox:checked');
        const countSpan = document.getElementById('selectedAuthLogsCount');
        const actionsDiv = document.getElementById('batchActionsAuth');
        
        if (checkboxes.length > 0) {
            actionsDiv.style.display = 'block';
            countSpan.textContent = `已选择 ${checkboxes.length} 条记录`;
        } else {
            actionsDiv.style.display = 'none';
            countSpan.textContent = '';
        }
    },

    /**
     * 批量删除认证日志
     */
    async batchDeleteAuthLogs() {
        const checkboxes = document.querySelectorAll('.auth-checkbox:checked');
        const ids = Array.from(checkboxes).map(cb => cb.value);
        
        if (ids.length === 0) {
            UI.showToast('请选择要删除的日志', 'warning');
            return;
        }

        if (!confirm(`确定要删除选中的 ${ids.length} 条认证日志吗？此操作不可恢复！`)) {
            return;
        }

        try {
            UI.showLoading();
            
            await API.batchDeleteAuthLogs(ids);
            
            UI.showToast(`成功删除 ${ids.length} 条认证日志`, 'success');
            
            // 取消全选
            document.getElementById('selectAllAuthLogs').checked = false;
            
            // 重新加载当前页
            await this.loadAuthLogs();
            
            UI.hideLoading();
        } catch (error) {
            console.error('[Logs] 批量删除认证日志失败:', error);
            UI.showToast('批量删除失败: ' + error.message, 'error');
            UI.hideLoading();
        }
    },

    /**
     * 从message推断device_id
     */
    inferDeviceIdFromMessage(message) {
        if (!message) return '未知设备';
        // 尝试从消息中提取设备ID，如果失败则返回"未知设备"
        // 消息格式可能是："co2数值超过安全阈值"
        return '未知设备';
    },

    /**
     * 从message推断alert_type
     */
    inferAlertTypeFromMessage(message) {
        if (!message) return 'unknown';
        
        // 根据消息内容推断告警类型
        if (message.includes('co2') || message.includes('CO2')) {
            return 'co2_high';
        } else if (message.includes('co') || message.includes('CO') && !message.includes('co2')) {
            return 'co_high';
        } else if (message.includes('烟雾')) {
            return 'smoke_detected';
        } else if (message.includes('液位') && message.includes('低')) {
            return 'liquid_level_low';
        } else if (message.includes('液位') && message.includes('高')) {
            return 'liquid_level_high';
        } else if (message.includes('电导率') && message.includes('低')) {
            return 'conductivity_low';
        } else if (message.includes('电导率') && message.includes('高')) {
            return 'conductivity_high';
        } else if (message.includes('温度') && message.includes('低')) {
            return 'temperature_low';
        } else if (message.includes('温度') && message.includes('高')) {
            return 'temperature_high';
        } else if (message.includes('流速') && message.includes('低')) {
            return 'flow_low';
        } else if (message.includes('流速') && message.includes('高')) {
            return 'flow_high';
        }
        
        return 'unknown';
    },

    /**
     * 一键清空所有认证日志
     */
    async clearAllAuthLogs() {
        // 先确认操作
        const confirmed = confirm('⚠️ 警告：此操作将清空所有认证日志！\n\n这是一个不可恢复的操作，确定要继续吗？');
        if (!confirmed) {
            return;
        }

        // 再次确认
        const doubleConfirmed = confirm('⚠️ 最后确认：\n\n您确定要清空所有认证日志吗？\n此操作无法撤销！');
        if (!doubleConfirmed) {
            return;
        }

        try {
            UI.showLoading();
            
            await API.clearAllAuthLogs();
            
            UI.showToast('成功清空所有认证日志', 'success');
            
            // 取消全选
            document.getElementById('selectAllAuthLogs').checked = false;
            
            // 重新加载当前页
            await this.loadAuthLogs();
            
            UI.hideLoading();
        } catch (error) {
            console.error('[Logs] 清空认证日志失败:', error);
            UI.showToast('清空日志失败: ' + error.message, 'error');
            UI.hideLoading();
        }
    }
};
