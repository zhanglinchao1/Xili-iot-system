/**
 * UI工具函数模块
 * 提供通用的UI交互功能
 * @version 2.0.1 - 添加详细日志
 */

console.log('%c[UI] 模块加载', 'color: #8b5cf6; font-weight: bold');

const UI = {
    version: '2.0.1',
    /**
     * 显示Toast通知
     */
    showToast(message, type = 'info', duration = 3000) {
        console.log(`[UI] showToast: "${message}" (类型: ${type}, 持续: ${duration}ms)`);

        const container = document.getElementById('toastContainer');
        if (!container) {
            console.error('[UI] ✗ 找不到 toastContainer 元素');
            return;
        }

        const toast = document.createElement('div');
        
        const icons = {
            success: 'fa-check-circle',
            error: 'fa-times-circle',
            warning: 'fa-exclamation-triangle',
            info: 'fa-info-circle'
        };
        
        toast.className = `toast toast-${type}`;
        toast.innerHTML = `
            <i class="fas ${icons[type]}"></i>
            <div class="toast-message">${message}</div>
        `;
        
        container.appendChild(toast);
        
        setTimeout(() => {
            toast.style.opacity = '0';
            setTimeout(() => toast.remove(), 300);
        }, duration);
    },

    /**
     * 显示加载状态
     */
    showLoading() {
        console.log('[UI] showLoading()');
        const overlay = document.getElementById('loadingOverlay');
        if (overlay) {
            overlay.classList.add('active');
        } else {
            console.error('[UI] ✗ 找不到 loadingOverlay 元素');
        }
    },

    /**
     * 隐藏加载状态
     */
    hideLoading() {
        console.log('[UI] hideLoading()');
        const overlay = document.getElementById('loadingOverlay');
        if (overlay) {
            overlay.classList.remove('active');
        } else {
            console.error('[UI] ✗ 找不到 loadingOverlay 元素');
        }
    },

    /**
     * 显示模态框
     */
    showModal(modalId) {
        console.log(`[UI] showModal: ${modalId}`);
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.add('active');
        } else {
            console.error(`[UI] ✗ 找不到模态框: #${modalId}`);
        }
    },

    /**
     * 隐藏模态框
     */
    hideModal(modalId) {
        console.log(`[UI] hideModal: ${modalId}`);
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('active');
        } else {
            console.error(`[UI] ✗ 找不到模态框: #${modalId}`);
        }
    },

    /**
     * 格式化日期时间
     */
    formatDateTime(dateString) {
        if (!dateString) return '-';
        
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;
        
        // 如果是最近的时间，显示相对时间
        if (diff < 60000) return '刚刚';
        if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
        if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
        
        // 否则显示完整日期
        return date.toLocaleString('zh-CN', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    },

    /**
     * 获取状态徽章HTML
     */
    getStatusBadge(status) {
        const statusMap = {
            online: { icon: 'fa-circle', text: '在线', class: 'status-online' },
            offline: { icon: 'fa-circle', text: '离线', class: 'status-offline' },
            fault: { icon: 'fa-exclamation-circle', text: '故障', class: 'status-fault' }
        };

        const config = statusMap[status] || statusMap.offline;
        return `
            <span class="status-badge ${config.class}">
                <i class="fas ${config.icon}"></i>
                ${config.text}
            </span>
        `;
    },

    /**
     * 获取传感器类型显示名称
     */
    getSensorTypeName(sensorType) {
        const typeMap = {
            co2: 'CO2传感器',
            co: 'CO传感器',
            smoke: '烟雾传感器',
            liquid_level: '液位传感器',
            conductivity: '电导率传感器',
            temperature: '温度传感器',
            flow: '流速传感器'
        };
        return typeMap[sensorType] || sensorType;
    },

    /**
     * 确认对话框
     */
    async confirm(message) {
        return window.confirm(message);
    },

    /**
     * 防抖函数
     */
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    /**
     * 生成分页HTML
     */
    renderPagination(currentPage, totalPages, onPageChange) {
        console.log(`[UI] renderPagination: 第${currentPage}页 / 共${totalPages}页`);

        const container = document.getElementById('pagination');
        if (!container) {
            console.error('[UI] ✗ 找不到 pagination 元素');
            return;
        }

        if (totalPages <= 1) {
            container.innerHTML = '';
            console.log('[UI] 总页数 <= 1, 不显示分页');
            return;
        }

        let html = `
            <button class="pagination-btn" ${currentPage === 1 ? 'disabled' : ''} data-page="${currentPage - 1}">
                <i class="fas fa-chevron-left"></i>
            </button>
        `;

        // 显示页码
        for (let i = 1; i <= totalPages; i++) {
            if (i === 1 || i === totalPages || (i >= currentPage - 2 && i <= currentPage + 2)) {
                html += `
                    <button class="pagination-btn ${i === currentPage ? 'active' : ''}" data-page="${i}">
                        ${i}
                    </button>
                `;
            } else if (i === currentPage - 3 || i === currentPage + 3) {
                html += `<span class="pagination-info">...</span>`;
            }
        }

        html += `
            <button class="pagination-btn" ${currentPage === totalPages ? 'disabled' : ''} data-page="${currentPage + 1}">
                <i class="fas fa-chevron-right"></i>
            </button>
            <span class="pagination-info">共 ${totalPages} 页</span>
        `;

        container.innerHTML = html;

        // 绑定点击事件
        container.querySelectorAll('.pagination-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const page = parseInt(btn.dataset.page);
                if (page && !btn.disabled) {
                    onPageChange(page);
                }
            });
        });
    },

    /**
     * 清空表单
     */
    clearForm(formId) {
        document.getElementById(formId).reset();
    },

    /**
     * 设置表单数据
     */
    setFormData(formId, data) {
        const form = document.getElementById(formId);
        Object.keys(data).forEach(key => {
            const input = form.elements[key];
            if (input) {
                input.value = data[key] || '';
            }
        });
    },

    /**
     * 获取表单数据
     */
    getFormData(formId) {
        const form = document.getElementById(formId);
        const formData = new FormData(form);
        const data = {};
        
        for (const [key, value] of formData.entries()) {
            if (value) {
                data[key] = value;
            }
        }
        
        return data;
    },

    /**
     * 空状态显示
     */
    renderEmptyState(container, message = '暂无数据') {
        console.log(`[UI] renderEmptyState: "${message}"`);

        if (!container) {
            console.error('[UI] ✗ renderEmptyState: container 为空');
            return;
        }

        container.innerHTML = `
            <tr>
                <td colspan="100%" style="text-align: center; padding: 40px; color: var(--gray-400);">
                    <i class="fas fa-inbox" style="font-size: 48px; margin-bottom: 12px; display: block;"></i>
                    ${message}
                </td>
            </tr>
        `;
    },

    /**
     * 获取传感器单位
     */
    getSensorUnit(sensorType) {
        const units = {
            'co2': 'ppm',
            'co': 'ppm',
            'smoke': 'AD值',
            'liquid_level': 'mm',
            'conductivity': 'mS/cm',
            'temperature': '°C',
            'flow': 'L/min'
        };
        return units[sensorType] || '';
    },

    /**
     * 格式化传感器数值显示
     */
    formatSensorValue(value, unit) {
        if (value === null || value === undefined) {
            return '<span style="color: var(--gray-400);">暂无数据</span>';
        }
        
        // 根据数值大小决定小数位数
        let formattedValue;
        if (value >= 1000) {
            formattedValue = value.toFixed(0);
        } else if (value >= 100) {
            formattedValue = value.toFixed(1);
        } else {
            formattedValue = value.toFixed(2);
        }
        
        return `<span style="font-weight: 600; color: var(--primary-600);">${formattedValue}</span> <span style="color: var(--gray-500); font-size: 0.85em;">${unit}</span>`;
    }
};

// 导出UI对象
window.UI = UI;

console.log('[UI] ✓ UI模块导出完成, version:', UI.version);

