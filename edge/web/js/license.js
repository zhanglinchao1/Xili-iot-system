/**
 * 许可证管理模块
 * 版本: 1.0.0
 * 功能: 显示许可证状态和详细信息
 */

const License = {
    version: '1.0.0',
    licenseData: null,
    isInitialized: false,
    isLoading: false,

    /**
     * 初始化许可证管理模块
     */
    init() {
        console.log(`[License] 许可证管理模块初始化 v${this.version}`);

        // 防止重复初始化
        if (this.isInitialized) {
            console.log('[License] ⚠️ 已初始化,跳过重复初始化');
            return;
        }

        this.isInitialized = true;
        this.loadLicenseInfo();
    },

    /**
     * 销毁模块(页面切换时调用)
     */
    destroy() {
        console.log('[License] 模块销毁');
        this.isInitialized = false;
        this.isLoading = false;
    },

    /**
     * 加载许可证信息
     */
    async loadLicenseInfo() {
        // 防止重复加载
        if (this.isLoading) {
            console.log('[License] ⚠️ 正在加载中,跳过重复请求');
            return;
        }

        try {
            this.isLoading = true;
            console.log('[License] 开始加载许可证信息...');

            const licenseInfoCard = document.getElementById('licenseInfoCard');
            const licenseLoading = document.getElementById('licenseLoading');

            // 显示加载动画
            if (licenseLoading) {
                licenseLoading.style.display = 'flex';
            }

            // 调用API获取许可证信息
            const data = await API.getLicenseInfo();
            this.licenseData = data;
            console.log('[License] 许可证数据:', data);

            // 渲染许可证信息
            this.renderLicenseInfo(data);
            console.log('[License] ✓ 许可证信息渲染完成');

            // 隐藏加载动画
            if (licenseLoading) {
                licenseLoading.style.display = 'none';
            }

        } catch (error) {
            console.error('[License] 加载许可证信息失败:', error);
            this.renderError('加载许可证信息失败: ' + error.message);
        } finally {
            this.isLoading = false;
            console.log('[License] 加载流程结束');
        }
    },

    /**
     * 渲染许可证信息
     */
    renderLicenseInfo(data) {
        const licenseInfoCard = document.getElementById('licenseInfoCard');

        if (!data.enabled) {
            // 检查是否有last_event来判断是否是吊销状态
            const isRevoked = data.last_event && (data.last_event.includes('吊销') || data.last_event === 'license_revoked');

            if (isRevoked) {
                // 许可证已被吊销或无效
                licenseInfoCard.innerHTML = `
                    <div class="license-disabled-notice license-revoked">
                        <i class="fas fa-ban"></i>
                        <h2>无许可证</h2>
                        <p>当前系统<strong>没有有效的许可证</strong>或许可证已被吊销。</p>
                        <div class="notice-details">
                            <p>没有有效许可证时，系统将受到以下限制：</p>
                            <ul>
                                <li>设备数量受到限制</li>
                                <li>部分高级功能不可用</li>
                                <li>无法进行正常的生产运营</li>
                            </ul>
                            <p class="mt-3"><strong>获取有效许可证：</strong></p>
                            <ol>
                                <li>联系管理员在Cloud端签发新的许可证</li>
                                <li>下载许可证文件到 <code>configs/license.lic</code></li>
                                <li>确保配置文件中 <code>license.enabled: true</code></li>
                                <li>重启Edge服务</li>
                            </ol>
                            ${data.last_event ? `<p class="mt-3"><strong>最后事件：</strong>${data.last_event} ${data.last_event_at ? `(${new Date(data.last_event_at).toLocaleString('zh-CN')})` : ''}</p>` : ''}
                        </div>
                    </div>
                `;
            } else {
                // 开发模式 - 许可证验证未启用
                licenseInfoCard.innerHTML = `
                    <div class="license-disabled-notice">
                        <i class="fas fa-info-circle"></i>
                        <h2>许可证验证未启用</h2>
                        <p>当前系统运行在<strong>开发模式</strong>，许可证验证功能已禁用。</p>
                        <div class="notice-details">
                            <p>开发模式下，系统不会进行以下验证：</p>
                            <ul>
                                <li>MAC地址绑定检查</li>
                                <li>许可证过期时间检查</li>
                                <li>设备数量限制检查</li>
                            </ul>
                            <p class="mt-3"><strong>如需启用许可证验证：</strong></p>
                            <ol>
                                <li>修改配置文件 <code>configs/config.yaml</code></li>
                                <li>将 <code>license.enabled</code> 设置为 <code>true</code></li>
                                <li>确保许可证文件 <code>configs/license.lic</code> 存在</li>
                                <li>重启Edge服务</li>
                            </ol>
                        </div>
                    </div>
                `;
            }
            return;
        }

        // 许可证已启用，显示详细信息
        const now = new Date();
        const expiresAt = new Date(data.expires_at);
        const isExpired = data.is_expired;
        const inGracePeriod = data.in_grace_period;

        // 计算剩余天数
        const daysRemaining = Math.ceil((expiresAt - now) / (1000 * 60 * 60 * 24));

        // 确定状态
        let statusClass = 'license-status-valid';
        let statusIcon = 'fa-check-circle';
        let statusText = '许可证有效';
        let statusColor = '#10b981';

        if (isExpired && !inGracePeriod) {
            statusClass = 'license-status-expired';
            statusIcon = 'fa-times-circle';
            statusText = '许可证已过期';
            statusColor = '#ef4444';
        } else if (inGracePeriod) {
            statusClass = 'license-status-warning';
            statusIcon = 'fa-exclamation-circle';
            statusText = '宽限期内';
            statusColor = '#f59e0b';
        } else if (daysRemaining < 30) {
            statusClass = 'license-status-warning';
            statusIcon = 'fa-exclamation-triangle';
            statusText = '即将过期';
            statusColor = '#f59e0b';
        }

        licenseInfoCard.innerHTML = `
            <div class="license-info-header ${statusClass}">
                <div class="license-status-badge">
                    <i class="fas ${statusIcon}" style="color: ${statusColor}"></i>
                    <span>${statusText}</span>
                </div>
                <div class="license-id-badge">
                    <i class="fas fa-fingerprint"></i>
                    <span>${data.license_id}</span>
                </div>
            </div>

            <div class="license-info-body">
                <!-- 基本信息 -->
                <div class="info-section">
                    <h3><i class="fas fa-info-circle"></i> 基本信息</h3>
                    <div class="info-grid">
                        <div class="info-item">
                            <label>许可证ID</label>
                            <div class="value">${data.license_id}</div>
                        </div>
                        <div class="info-item">
                            <label>绑定MAC地址</label>
                            <div class="value"><code>${data.mac_address}</code></div>
                        </div>
                        <div class="info-item">
                            <label>最大设备数</label>
                            <div class="value"><strong>${data.max_devices === -1 ? '无限制' : data.max_devices + ' 台'}</strong></div>
                        </div>
                        <div class="info-item">
                            <label>状态</label>
                            <div class="value">
                                <span class="status-badge ${statusClass}">
                                    <i class="fas ${statusIcon}"></i>
                                    ${statusText}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 时间信息 -->
                <div class="info-section">
                    <h3><i class="fas fa-clock"></i> 有效期信息</h3>
                    <div class="info-grid">
                        <div class="info-item">
                            <label>过期时间</label>
                            <div class="value">${expiresAt.toLocaleString('zh-CN')}</div>
                        </div>
                        <div class="info-item">
                            <label>剩余天数</label>
                            <div class="value ${daysRemaining < 30 ? 'text-warning' : ''}">
                                <strong>${daysRemaining >= 0 ? daysRemaining : 0}</strong> 天
                                ${daysRemaining < 0 ? '（已过期）' : ''}
                            </div>
                        </div>
                        <div class="info-item">
                            <label>宽限期状态</label>
                            <div class="value">
                                ${inGracePeriod ?
                                    '<span class="text-warning"><i class="fas fa-clock"></i> 宽限期内（72小时）</span>' :
                                    '<span class="text-success"><i class="fas fa-check"></i> 未进入宽限期</span>'}
                            </div>
                        </div>
                        <div class="info-item">
                            <label>是否过期</label>
                            <div class="value">
                                ${isExpired ?
                                    '<span class="text-danger"><i class="fas fa-times"></i> 已过期</span>' :
                                    '<span class="text-success"><i class="fas fa-check"></i> 未过期</span>'}
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 警告提示 -->
                ${this.generateAlertSection(isExpired, inGracePeriod, daysRemaining)}
            </div>

            <div class="license-info-footer">
                <div class="footer-actions">
                    <button class="btn btn-secondary" onclick="License.loadLicenseInfo()">
                        <i class="fas fa-sync-alt"></i> 刷新
                    </button>
                    <button class="btn btn-secondary" onclick="License.showLicenseGuide()">
                        <i class="fas fa-question-circle"></i> 续期指南
                    </button>
                </div>
                <div class="footer-note">
                    <i class="fas fa-shield-alt"></i>
                    <span>许可证通过RSA-2048签名验证，MAC地址绑定，确保系统安全</span>
                </div>
            </div>
        `;
    },

    /**
     * 生成告警提示区域
     */
    generateAlertSection(isExpired, inGracePeriod, daysRemaining) {
        if (isExpired && !inGracePeriod) {
            return `
                <div class="alert alert-danger">
                    <i class="fas fa-exclamation-circle"></i>
                    <div class="alert-content">
                        <strong>许可证已过期且超过宽限期！</strong>
                        <p>系统将拒绝新设备的认证请求。请立即联系厂商续期。</p>
                    </div>
                </div>
            `;
        } else if (inGracePeriod) {
            return `
                <div class="alert alert-warning">
                    <i class="fas fa-exclamation-triangle"></i>
                    <div class="alert-content">
                        <strong>许可证已过期，当前在宽限期内</strong>
                        <p>系统仍可正常使用，但请尽快联系厂商续期。宽限期为72小时。</p>
                    </div>
                </div>
            `;
        } else if (daysRemaining < 30) {
            return `
                <div class="alert alert-warning">
                    <i class="fas fa-exclamation-triangle"></i>
                    <div class="alert-content">
                        <strong>许可证即将过期</strong>
                        <p>剩余${daysRemaining}天，建议提前联系厂商续期，避免影响系统正常使用。</p>
                    </div>
                </div>
            `;
        }
        return '';
    },

    /**
     * 渲染错误信息
     */
    renderError(message) {
        const licenseInfoCard = document.getElementById('licenseInfoCard');
        const licenseLoading = document.getElementById('licenseLoading');

        if (licenseLoading) {
            licenseLoading.style.display = 'none';
        }

        licenseInfoCard.innerHTML = `
            <div class="error-message">
                <i class="fas fa-exclamation-triangle"></i>
                <h3>加载失败</h3>
                <p>${message}</p>
                <button class="btn btn-primary" onclick="License.loadLicenseInfo()">
                    <i class="fas fa-redo"></i> 重新加载
                </button>
            </div>
        `;
    },

    /**
     * 显示续期指南
     */
    showLicenseGuide() {
        UI.showModal('许可证续期指南', `
            <div class="license-guide">
                <h4>如何续期许可证？</h4>
                <ol>
                    <li>
                        <strong>联系厂商</strong>
                        <p>联系Edge系统供应商，提供以下信息：</p>
                        <ul>
                            <li>许可证ID: <code>${this.licenseData?.license_id || 'N/A'}</code></li>
                            <li>MAC地址: <code>${this.licenseData?.mac_address || 'N/A'}</code></li>
                            <li>需要的设备数量</li>
                        </ul>
                    </li>
                    <li>
                        <strong>获取新许可证</strong>
                        <p>厂商会提供新的许可证文件 <code>license.lic</code></p>
                    </li>
                    <li>
                        <strong>替换许可证文件</strong>
                        <p>将新的许可证文件替换到:</p>
                        <code>configs/license.lic</code>
                    </li>
                    <li>
                        <strong>重启服务</strong>
                        <p>运行命令重启Edge服务:</p>
                        <code>./stop_all.sh && ./start_all.sh</code>
                    </li>
                </ol>

                <div class="alert alert-info">
                    <i class="fas fa-info-circle"></i>
                    <p><strong>注意：</strong>宽限期为72小时，超过宽限期后新设备将无法认证，已认证设备不受影响。</p>
                </div>
            </div>
        `);
    }
};

// 导出模块
window.License = License;
