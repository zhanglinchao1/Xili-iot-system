/**
 * 告警阈值管理模块
 * 负责从API获取和格式化阈值信息
 */

const ThresholdManager = {
    thresholds: null,
    loaded: false,

    /**
     * 从API加载阈值配置
     */
    async loadThresholds() {
        try {
            const config = await API.getAlertConfig();
            if (config.enabled && config.thresholds) {
                this.thresholds = config.thresholds;
                this.loaded = true;
                console.log('[Threshold] 阈值配置已加载:', this.thresholds);
                return true;
            } else {
                console.warn('[Threshold] 告警未启用或阈值配置为空');
                this.thresholds = null;
                this.loaded = false;
                return false;
            }
        } catch (error) {
            console.error('[Threshold] 加载阈值配置失败:', error);
            this.thresholds = null;
            this.loaded = false;
            return false;
        }
    },

    /**
     * 获取传感器阈值文本
     * @param {string} sensorType - 传感器类型
     * @returns {string} 格式化的阈值文本
     */
    getThresholdText(sensorType) {
        if (!this.loaded || !this.thresholds || !this.thresholds[sensorType]) {
            return this.getDefaultThresholdText(sensorType);
        }

        const threshold = this.thresholds[sensorType];
        const unit = this.getSensorUnit(sensorType);

        // 处理单限制(最大值或最小值)
        if (threshold.min === 0) {
            return `< ${threshold.max} ${unit}`;
        } else if (threshold.max === 0 || !threshold.max) {
            return `> ${threshold.min} ${unit}`;
        }

        // 处理范围限制
        return `${threshold.min}-${threshold.max} ${unit}`;
    },

    /**
     * 获取默认阈值文本(用于兜底)
     */
    getDefaultThresholdText(sensorType) {
        const defaults = {
            'co2': '< 5000 ppm',
            'co': '< 50 ppm',
            'smoke': '< 1000 AD值',
            'liquid_level': '100-900 mm',
            'conductivity': '0.5-10 mS/cm',
            'temperature': '-10 ~ 60 °C',
            'flow': '0.5-100 L/min'
        };
        return defaults[sensorType] || '-';
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
     * 获取传感器名称
     */
    getSensorName(sensorType) {
        const names = {
            'co2': 'CO2',
            'co': 'CO',
            'smoke': '烟雾',
            'liquid_level': '液位',
            'conductivity': '电导率',
            'temperature': '温度',
            'flow': '流速'
        };
        return names[sensorType] || sensorType;
    },

    /**
     * 检查是否超出阈值
     * @param {string} sensorType - 传感器类型
     * @param {number} value - 传感器值
     * @returns {boolean} 是否超出阈值
     */
    isExceeded(sensorType, value) {
        if (!this.loaded || !this.thresholds || !this.thresholds[sensorType]) {
            return false;
        }

        const threshold = this.thresholds[sensorType];

        // 检查最大值
        if (threshold.max > 0 && value > threshold.max) {
            return true;
        }

        // 检查最小值
        if (threshold.min > 0 && value < threshold.min) {
            return true;
        }

        return false;
    }
};

// 导出全局对象
window.ThresholdManager = ThresholdManager;
