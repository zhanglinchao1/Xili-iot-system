/**
 * 主应用入口
 * 初始化应用和全局事件处理
 * 集成统一WebSocket管理器
 * @version 3.0.0
 */

const App = {
    currentPage: 'dashboard',
    version: '3.0.0',

    /**
     * 初始化应用
     */
    async init() {
        console.log('%c=== Edge设备管理系统初始化 ===', 'color: #2563eb; font-size: 16px; font-weight: bold');
        console.log('版本:', this.version);
        console.log('时间:', new Date().toLocaleString());
        console.log('-------------------------------');

        try {
            // 1. 绑定全局事件
            console.log('步骤 1/7: 绑定全局事件...');
            this.bindGlobalEvents();
            console.log('✓ 全局事件绑定完成');

            // 2. 初始化WebSocket管理器
            console.log('步骤 2/7: 初始化WebSocket管理器...');
            if (typeof WebSocketManager !== 'undefined') {
                WebSocketManager.init();
                console.log('✓ WebSocket管理器初始化完成');
            } else {
                console.warn('⚠ WebSocketManager未定义，将使用HTTP轮询模式');
            }

            // 3. 检查系统健康状态
            console.log('步骤 3/7: 检查系统健康状态...');
            await this.checkSystemHealth();
            console.log('✓ 系统健康检查完成');

            // 4. 初始化设备管理模块
            console.log('步骤 4/7: 初始化设备管理模块...');
            if (typeof DeviceManager === 'undefined') {
                throw new Error('DeviceManager 未定义！请检查 devices.js 是否正确加载');
            }
            DeviceManager.init();
            console.log('✓ 设备管理模块初始化完成');

            // 5. 恢复上次访问的页面（通过URL hash或localStorage）
            console.log('步骤 5/7: 恢复上次访问的页面...');
            const savedPage = this.getSavedPage();
            if (savedPage && savedPage !== 'dashboard') {
                console.log(`恢复到页面: ${savedPage}`);
                this.switchPage(savedPage);
            } else {
                console.log('加载默认页面: dashboard');
                await this.loadDashboard();
            }
            console.log('✓ 页面恢复完成');

            // 6. 更新告警通知徽章
            console.log('步骤 6/7: 更新告警通知...');
            if (typeof Alerts !== 'undefined') {
                await Alerts.updateUnresolvedCount();
                console.log('✓ 告警通知更新完成');
            }

            // 7. 加载许可证信息
            console.log('步骤 7/7: 加载许可证信息...');
            await this.loadLicenseInfo();
            console.log('✓ 许可证信息加载完成');

            console.log('%c✓ 应用初始化完成!', 'color: #10b981; font-size: 14px; font-weight: bold');
        } catch (error) {
            console.error('%c✗ 应用初始化失败!', 'color: #ef4444; font-size: 14px; font-weight: bold');
            console.error('错误详情:', error);
            console.error('错误堆栈:', error.stack);

            // 显示错误提示
            if (typeof UI !== 'undefined' && UI.showToast) {
                UI.showToast('应用初始化失败: ' + error.message, 'error', 10000);
            } else {
                alert('应用初始化失败: ' + error.message);
            }
        }
    },

    /**
     * 绑定全局事件
     */
    bindGlobalEvents() {
        // 响应式菜单切换
        const menuToggle = document.getElementById('menuToggle');
        const sidebar = document.getElementById('sidebar');

        menuToggle.addEventListener('click', () => {
            const width = window.innerWidth;

            if (width <= 1024) {
                // 小屏幕：切换显示/隐藏
                sidebar.classList.toggle('active');
            } else {
                // 中大屏幕：切换收起/展开
                sidebar.classList.toggle('collapsed');
            }
        });

        // 侧边栏导航切换
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                const page = item.dataset.page;
                this.switchPage(page);
                
                // 移动端：点击导航项后自动关闭侧边栏
                if (window.innerWidth <= 1024 && sidebar.classList.contains('active')) {
                    sidebar.classList.remove('active');
                }
            });
        });

        // 点击遮罩关闭侧边栏（小屏幕）
        document.addEventListener('click', (e) => {
            if (window.innerWidth <= 1024 &&
                !sidebar.contains(e.target) &&
                !menuToggle.contains(e.target) &&
                sidebar.classList.contains('active')) {
                sidebar.classList.remove('active');
            }
        });

        // 刷新按钮
        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.refreshCurrentPage();
        });

        // 模态框点击遮罩关闭
        document.querySelectorAll('.modal-overlay').forEach(overlay => {
            overlay.addEventListener('click', () => {
                overlay.parentElement.classList.remove('active');
            });
        });

        // 键盘快捷键
        document.addEventListener('keydown', (e) => {
            // ESC关闭模态框
            if (e.key === 'Escape') {
                document.querySelectorAll('.modal.active').forEach(modal => {
                    modal.classList.remove('active');
                });
            }

            // Ctrl/Cmd + R 刷新
            if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
                e.preventDefault();
                this.refreshCurrentPage();
            }
        });

        // 页面卸载时清理资源
        window.addEventListener('beforeunload', () => {
            console.log('🧹 页面卸载,清理资源...');
            
            // 销毁WebSocket管理器
            if (typeof WebSocketManager !== 'undefined' && WebSocketManager.destroy) {
                WebSocketManager.destroy();
            }
            
            // 销毁各模块
            if (typeof RealtimeMonitor !== 'undefined' && RealtimeMonitor.destroy) {
                RealtimeMonitor.destroy();
            }
            if (typeof DeviceManager !== 'undefined' && DeviceManager.destroy) {
                DeviceManager.destroy();
            }
            if (typeof Statistics !== 'undefined' && Statistics.destroy) {
                Statistics.destroy();
            }
            if (typeof Alerts !== 'undefined' && Alerts.destroy) {
                Alerts.destroy();
            }
        });
    },

    /**
     * 切换页面
     */
    switchPage(pageName) {
        console.log(`%c[App] 切换页面: ${pageName}`, 'color: #2563eb; font-weight: bold');

        try {
            // 更新导航状态
            console.log('[App] 更新导航状态...');
            document.querySelectorAll('.nav-item').forEach(item => {
                item.classList.remove('active');
            });

            const navItem = document.querySelector(`.nav-item[data-page="${pageName}"]`);
            if (!navItem) {
                console.error(`[App] ✗ 找不到导航项: data-page="${pageName}"`);
                UI.showToast(`导航项不存在: ${pageName}`, 'error');
                return;
            }
            navItem.classList.add('active');
            console.log('[App] ✓ 导航状态已更新');

            // 销毁旧页面模块
            if (this.currentPage && this.currentPage !== pageName) {
                console.log(`[App] 销毁旧页面模块: ${this.currentPage}`);
                this.destroyPageModule(this.currentPage);
            }

            // 更新页面显示
            console.log('[App] 更新页面显示...');
            document.querySelectorAll('.page').forEach(page => {
                page.classList.remove('active');
            });

            const pageElement = document.getElementById(`${pageName}Page`);
            if (!pageElement) {
                console.error(`[App] ✗ 找不到页面元素: id="${pageName}Page"`);
                UI.showToast(`页面不存在: ${pageName}`, 'error');
                return;
            }
            pageElement.classList.add('active');
            console.log('[App] ✓ 页面显示已更新');

            this.currentPage = pageName;

            // 保存当前页面到localStorage
            this.savePage(pageName);

            // 加载对应页面数据
            console.log('[App] 加载页面数据...');
            this.loadPageData(pageName);

            console.log(`[App] ✓ 页面切换完成: ${pageName}`);
        } catch (error) {
            console.error('[App] ✗ 切换页面失败:', error);
            console.error('[App] 错误堆栈:', error.stack);
            UI.showToast('切换页面失败: ' + error.message, 'error');
        }
    },

    /**
     * 销毁页面模块
     */
    destroyPageModule(pageName) {
        switch (pageName) {
            case 'statistics':
                if (typeof Statistics !== 'undefined' && typeof Statistics.destroy === 'function') {
                    Statistics.destroy();
                }
                break;
            case 'license':
                if (typeof License !== 'undefined' && typeof License.destroy === 'function') {
                    License.destroy();
                }
                break;
            case 'alerts':
                if (typeof Alerts !== 'undefined' && typeof Alerts.destroy === 'function') {
                    Alerts.destroy();
                }
                break;
            case 'logs':
                if (typeof Logs !== 'undefined' && typeof Logs.destroy === 'function') {
                    Logs.destroy();
                }
                break;
            case 'cabinet':
                if (typeof Cabinet !== 'undefined' && typeof Cabinet.destroy === 'function') {
                    Cabinet.destroy();
                }
                break;
            case 'vulnerability':
                if (typeof Vulnerability !== 'undefined' && typeof Vulnerability.destroy === 'function') {
                    Vulnerability.destroy();
                }
                break;
        }
    },

    /**
     * 加载页面数据
     */
    async loadPageData(pageName) {
        switch (pageName) {
            case 'dashboard':
                await this.loadDashboard();
                break;
            case 'devices':
                await DeviceManager.loadDevices();
                break;
            case 'statistics':
                // 初始化统计分析模块
                if (typeof Statistics !== 'undefined') {
                    await Statistics.init();
                }
                break;
            case 'alerts':
                // 初始化告警管理模块
                if (typeof Alerts !== 'undefined') {
                    await Alerts.init();
                }
                break;
            case 'logs':
                // 初始化日志记录模块
                if (typeof Logs !== 'undefined') {
                    await Logs.init();
                }
                break;
            case 'license':
                // 初始化许可证管理模块
                if (typeof License !== 'undefined') {
                    await License.init();
                }
                break;
            case 'cabinet':
                // 初始化储能柜管理模块
                if (typeof Cabinet !== 'undefined') {
                    await Cabinet.init();
                }
                break;
            case 'vulnerability':
                // 初始化脆弱性分析模块
                if (typeof Vulnerability !== 'undefined') {
                    await Vulnerability.init();
                }
                break;
            case 'settings':
                // 系统设置页面（待开发）
                break;
        }
    },

    /**
     * 刷新当前页面
     */
    async refreshCurrentPage() {
        const btn = document.getElementById('refreshBtn');
        btn.querySelector('i').classList.add('fa-spin');
        
        try {
            await this.loadPageData(this.currentPage);
            UI.showToast('数据已刷新', 'success', 2000);
        } catch (error) {
            UI.showToast('刷新失败', 'error');
        } finally {
            setTimeout(() => {
                btn.querySelector('i').classList.remove('fa-spin');
            }, 500);
        }
    },

    /**
     * 检查系统健康状态
     */
    async checkSystemHealth() {
        try {
            const health = await API.healthCheck();
            console.log('系统健康检查:', health);
            
            // 更新系统状态显示
            const statusEl = document.querySelector('.system-status');
            if (health.status === 'ok') {
                statusEl.innerHTML = '<i class="fas fa-circle status-online"></i><span>系统在线</span>';
            } else {
                statusEl.innerHTML = '<i class="fas fa-circle" style="color: var(--danger-color);"></i><span>系统异常</span>';
                UI.showToast('系统健康检查异常', 'warning');
            }
        } catch (error) {
            console.error('健康检查失败:', error);
            const statusEl = document.querySelector('.system-status');
            statusEl.innerHTML = '<i class="fas fa-circle" style="color: var(--danger-color);"></i><span>连接失败</span>';
            UI.showToast('无法连接到后端服务，请检查服务是否启动', 'error', 5000);
        }
    },

    /**
     * 加载仪表板数据
     */
    async loadDashboard() {
        try {
            // 加载统计信息
            await DeviceManager.loadStatistics();

            // 初始化实时传感器监控
            if (typeof RealtimeMonitor !== 'undefined') {
                console.log('初始化实时传感器监控...');
                RealtimeMonitor.init();
            } else {
                console.warn('RealtimeMonitor 未定义');
            }

        } catch (error) {
            console.error('加载仪表板数据失败:', error);
            UI.showToast('加载仪表板数据失败', 'error');
        }
    },

    /**
     * 加载许可证信息并更新显示
     */
    async loadLicenseInfo() {
        try {
            const info = await API.getLicenseInfo();
            const licenseStatus = document.getElementById('licenseStatus');
            const licenseStatusText = document.getElementById('licenseStatusText');
            const licenseDetail = document.getElementById('licenseDetail');

            if (!info.enabled) {
                // 许可证未启用，不显示
                licenseStatus.style.display = 'none';
                return;
            }

            // 许可证已启用，显示状态
            licenseStatus.style.display = 'flex';

            // 检查许可证状态
            const now = new Date();
            const expiresAt = new Date(info.expires_at);
            const isExpired = info.is_expired;
            const inGracePeriod = info.in_grace_period;

            // 移除所有状态类
            licenseStatus.classList.remove('warning', 'expired');

            if (isExpired && !inGracePeriod) {
                // 已过期且超过宽限期
                licenseStatus.classList.add('expired');
                licenseStatusText.textContent = '许可证已过期';
                licenseDetail.textContent = `过期时间: ${expiresAt.toLocaleDateString()}`;
            } else if (inGracePeriod) {
                // 宽限期内
                licenseStatus.classList.add('warning');
                licenseStatusText.textContent = '许可证宽限期';
                licenseDetail.textContent = '请尽快续期';
            } else {
                // 正常状态
                const daysRemaining = Math.ceil((expiresAt - now) / (1000 * 60 * 60 * 24));
                licenseStatusText.textContent = '许可证有效';
                licenseDetail.textContent = `剩余 ${daysRemaining} 天`;

                // 如果少于30天，显示警告
                if (daysRemaining < 30) {
                    licenseStatus.classList.add('warning');
                }
            }

            // 添加设备数量信息
            if (info.max_devices) {
                licenseDetail.textContent += ` | 最多${info.max_devices}台设备`;
            }

        } catch (error) {
            console.error('加载许可证信息失败:', error);
            // 加载失败不显示许可证状态
            const licenseStatus = document.getElementById('licenseStatus');
            licenseStatus.style.display = 'none';
        }
    },

    /**
     * 定期刷新数据(备用方案 - WebSocket连接失败时使用)
     * 由于已使用WebSocket实时推送，此方法主要用于:
     * 1. 定期同步告警徽章数量
     * 2. 作为WebSocket失败时的备用机制
     */
    startAutoRefresh(interval = 120000) { // 120秒刷新一次 (降低频率,WebSocket提供实时数据)
        setInterval(() => {
            // 检查WebSocket连接状态
            const wsConnected = typeof WebSocketManager !== 'undefined' && WebSocketManager.isConnected();
            
            if (!wsConnected) {
                // WebSocket未连接时,使用HTTP轮询作为备用
                console.log('[App] WebSocket未连接,使用HTTP轮询刷新数据');
                
                if (this.currentPage === 'dashboard') {
                    this.loadDashboard();
                } else if (this.currentPage === 'devices') {
                    DeviceManager.loadDevices();
                }
            }

            // 无论WebSocket状态如何,都定期更新告警徽章(确保数量准确)
            if (typeof Alerts !== 'undefined') {
                Alerts.updateUnresolvedCount();
            }
        }, interval);
    },

    /**
     * 保存当前页面到localStorage
     */
    savePage(pageName) {
        try {
            localStorage.setItem('edge_current_page', pageName);
            console.log(`[App] 页面已保存: ${pageName}`);
        } catch (error) {
            console.error('[App] 保存页面失败:', error);
        }
    },

    /**
     * 从localStorage恢复上次访问的页面
     */
    getSavedPage() {
        try {
            const savedPage = localStorage.getItem('edge_current_page');
            if (savedPage) {
                console.log(`[App] 恢复保存的页面: ${savedPage}`);
                return savedPage;
            }
        } catch (error) {
            console.error('[App] 读取保存的页面失败:', error);
        }
        return 'dashboard'; // 默认返回仪表盘
    },

    /**
     * 生成测试数据（开发用）
     */
    async generateTestDevice() {
        const testDevice = {
            device_id: `TEST_${Date.now()}`,
            cabinet_id: 'CABINET_TEST_01',
            sensor_type: 'co2',
            public_key: '04' + Array(64).fill(0).map(() => 
                Math.floor(Math.random() * 16).toString(16)).join(''),
            commitment: '0x' + Array(64).fill(0).map(() => 
                Math.floor(Math.random() * 16).toString(16)).join(''),
            model: 'TEST-MODEL-V1',
            manufacturer: 'TestMfg',
            firmware_ver: '1.0.0'
        };

        try {
            await API.registerDevice(testDevice);
            UI.showToast('测试设备创建成功', 'success');
            DeviceManager.loadDevices();
        } catch (error) {
            UI.showToast('创建测试设备失败: ' + error.message, 'error');
        }
    }
};

// 页面加载完成后初始化应用
// 使用标准的DOMContentLoaded模式,避免重复初始化
(function() {
    function initializeApp() {
        console.log('[App] 开始初始化应用...');
        App.init();
        // 启动备用自动刷新（每120秒,主要依赖WebSocket实时推送）
        App.startAutoRefresh(120000);
        console.log('[App] 应用初始化完成');
    }

    // 如果文档已加载完成,立即初始化
    if (document.readyState === 'loading') {
        // 文档正在加载,等待DOMContentLoaded事件
        document.addEventListener('DOMContentLoaded', initializeApp, { once: true });
    } else {
        // 文档已加载完成,立即初始化
        initializeApp();
    }
})();

// 将App暴露到全局，方便调试
window.App = App;

// 开发环境辅助函数
if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
    console.log('%c开发模式已启用', 'color: #2563eb; font-size: 14px; font-weight: bold;');
    console.log('%c可用的调试命令:', 'color: #666; font-size: 12px;');
    console.log('  App.generateTestDevice() - 创建测试设备');
    console.log('  DeviceManager.loadDevices() - 重新加载设备列表');
    console.log('  DeviceManager.loadStatistics() - 重新加载统计信息');
    console.log('  API.healthCheck() - 健康检查');
}

