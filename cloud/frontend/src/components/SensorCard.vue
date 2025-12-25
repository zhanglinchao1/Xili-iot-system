<!--
  传感器数据卡片组件
  按照设计图重新设计，显示传感器实时数据、阈值和告警信息
-->
<template>
  <el-card class="sensor-card" shadow="hover" @click="handleCardClick">
    <!-- 顶部：图标和传感器名称 -->
    <div class="sensor-header">
      <div class="sensor-icon-wrapper" :class="iconColorClass">
        <el-icon :size="20">
          <component :is="sensorIcon" />
        </el-icon>
      </div>
      <div class="sensor-title-group">
        <div class="sensor-name">{{ sensorNameCN }}</div>
        <div class="sensor-status-text" :class="statusTextClass">{{ statusTextCN }}</div>
      </div>
    </div>

    <!-- 当前读数区域（白色凹陷区域） -->
    <div class="sensor-reading-container">
      <div class="sensor-reading">
        <span class="reading-value">{{ displayValue }}</span>
        <span class="reading-unit">{{ unit }}</span>
      </div>
    </div>

    <!-- 阈值和告警时间信息 -->
    <div class="sensor-info-section">
      <div class="info-item">
        <el-icon :size="14" class="info-icon"><Bell /></el-icon>
        <span class="info-label">阈值:</span>
        <span class="info-value">{{ thresholdText }}</span>
      </div>
      <div class="info-item">
        <el-icon :size="14" class="info-icon"><Clock /></el-icon>
        <span class="info-label">告警时间:</span>
        <span class="info-value">{{ alarmTimeText }}</span>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { 
  Clock, 
  Bell,
  Warning,
  WindPower,
  Finished,
  DataLine,
  HotWater,
  Sunny,
  ColdDrink
} from '@element-plus/icons-vue'

/**
 * 组件属性定义
 */
interface Props {
  sensorType: string       // 传感器类型
  deviceName?: string      // 设备名称
  deviceId: string         // 设备ID
  value?: number           // 传感器值（可选，如果没有数据则为undefined）
  unit: string             // 单位
  quality?: number         // 数据质量 (0-100)
  status?: string          // 状态 (normal/warning/error/offline)
  timestamp?: string       // 时间戳
  threshold?: {            // 阈值信息（可选）
    min?: number
    max?: number
  }
  alarmTime?: string       // 告警时间（可选）
}

const props = withDefaults(defineProps<Props>(), {
  deviceName: '',
  quality: 100,
  status: 'offline'
})

/**
 * 事件定义
 */
const emit = defineEmits<{
  viewDetail: [deviceId: string, sensorType: string]
  click: [deviceId: string]
}>()

/**
 * 处理卡片点击
 */
const handleCardClick = () => {
  emit('click', props.deviceId)
}

/**
 * 传感器类型映射表（中文名称和图标）
 */
const sensorTypeMap: Record<string, { 
  nameCN: string, 
  nameEN: string, 
  icon: any,
  iconColor: string,
  defaultThreshold: { min?: number, max?: number }
}> = {
  'co2': { 
    nameCN: 'CO2传感器', 
    nameEN: 'CO2 Sensor', 
    icon: ColdDrink,
    iconColor: 'purple',
    defaultThreshold: { max: 5000 }
  },
  'co': { 
    nameCN: 'CO传感器', 
    nameEN: 'CO Sensor', 
    icon: Warning,
    iconColor: 'pink',
    defaultThreshold: { max: 50 }
  },
  'smoke': { 
    nameCN: '烟雾传感器', 
    nameEN: 'Smoke Sensor', 
    icon: WindPower,
    iconColor: 'orange',
    defaultThreshold: { max: 1000 }
  },
  'liquid_level': { 
    nameCN: '液位传感器', 
    nameEN: 'Liquid Level Sensor', 
    icon: Finished,
    iconColor: 'teal',
    defaultThreshold: { min: 0, max: 900 }
  },
  'conductivity': { 
    nameCN: '电导率传感器', 
    nameEN: 'Conductivity Sensor', 
    icon: DataLine,
    iconColor: 'yellow',
    defaultThreshold: { max: 10 }
  },
  'temperature': { 
    nameCN: '温度传感器', 
    nameEN: 'Temperature Sensor', 
    icon: Sunny,
    iconColor: 'red',
    defaultThreshold: { min: -10, max: 60 }
  },
  'flow': { 
    nameCN: '流速传感器', 
    nameEN: 'Flow Sensor', 
    icon: HotWater,
    iconColor: 'green',
    defaultThreshold: { max: 100 }
  }
}

/**
 * 传感器中文名称
 */
const sensorNameCN = computed(() => {
  if (props.deviceName) {
    return props.deviceName
  }
  return sensorTypeMap[props.sensorType]?.nameCN || props.sensorType
})

/**
 * 传感器图标
 */
const sensorIcon = computed(() => {
  return sensorTypeMap[props.sensorType]?.icon || DataLine
})

/**
 * 图标颜色类
 */
const iconColorClass = computed(() => {
  const color = sensorTypeMap[props.sensorType]?.iconColor || 'default'
  return `icon-${color}`
})

/**
 * 状态文本（中文）
 */
const statusTextCN = computed(() => {
  if (props.status === 'offline' || props.value === undefined || props.value === null) {
    return '离线'
  }
  const statusMap: Record<string, string> = {
    'normal': '正常',
    'warning': '警告',
    'error': '异常'
  }
  return statusMap[props.status || 'normal'] || '未知'
})

/**
 * 状态文本样式类
 */
const statusTextClass = computed(() => {
  if (props.status === 'offline' || props.value === undefined || props.value === null) {
    return 'status-offline'
  }
  return `status-${props.status || 'normal'}`
})

/**
 * 显示值（如果没有数据则显示"--"）
 */
const displayValue = computed(() => {
  // 如果状态是offline或者value是undefined/null/0且quality为0，显示"--"
  if (props.status === 'offline' || (props.value === undefined || props.value === null || (props.value === 0 && props.quality === 0))) {
    return '--'
  }
  // 根据单位格式化数值
  if (props.unit === '°C' || props.unit === 'mm' || props.unit === 'mS/cm' || props.unit === 'L/min') {
    return props.value.toFixed(1)
  }
  // ppm等整数单位
  return Math.round(props.value).toString()
})

/**
 * 阈值文本
 */
const thresholdText = computed(() => {
  const threshold = props.threshold || sensorTypeMap[props.sensorType]?.defaultThreshold || { max: 0 }
  const unit = props.unit || ''
  
  if (threshold.min !== undefined && threshold.max !== undefined) {
    return `< ${threshold.min}-${threshold.max} ${unit}`
  } else if (threshold.max !== undefined) {
    return `< ${threshold.max} ${unit}`
  } else if (threshold.min !== undefined) {
    return `> ${threshold.min} ${unit}`
  }
  return '-'
})

/**
 * 告警时间文本
 */
const alarmTimeText = computed(() => {
  if (props.alarmTime) {
    const date = new Date(props.alarmTime)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    
    // 小于1分钟显示"刚刚"
    if (diff < 60000) {
      return '刚刚'
    }
    
    // 小于1小时显示分钟数
    if (diff < 3600000) {
      const minutes = Math.floor(diff / 60000)
      return `${minutes}分钟前`
    }
    
    // 小于24小时显示小时数
    if (diff < 86400000) {
      const hours = Math.floor(diff / 3600000)
      return `${hours}小时前`
    }
    
    // 显示日期和时间
    return date.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  return '-'
})
</script>

<style scoped>
.sensor-card {
  border-radius: 8px;
  transition: all 0.3s;
  border: 1px solid #ebeef5;
  cursor: pointer;
  padding: 16px;
  background: #ffffff;
}

.sensor-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* 顶部：图标和名称 */
.sensor-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.sensor-icon-wrapper {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

/* 图标颜色 */
.icon-purple {
  background-color: #9c27b0;
}

.icon-pink {
  background-color: #e91e63;
}

.icon-orange {
  background-color: #ff9800;
}

.icon-teal {
  background-color: #009688;
}

.icon-yellow {
  background-color: #ffc107;
}

.icon-red {
  background-color: #f44336;
}

.icon-green {
  background-color: #4caf50;
}

.sensor-title-group {
  flex: 1;
  min-width: 0;
}

.sensor-name {
  font-size: 16px;
  font-weight: bold;
  color: #303133;
  margin-bottom: 4px;
  line-height: 1.4;
}

.sensor-status-text {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.status-offline {
  color: #909399;
}

.status-normal {
  color: #67c23a;
}

.status-warning {
  color: #e6a23c;
}

.status-error {
  color: #f56c6c;
}

/* 当前读数区域（白色凹陷区域） */
.sensor-reading-container {
  background-color: #f5f7fa;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 16px;
  border: 1px solid #e4e7ed;
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.05);
}

.sensor-reading {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 4px;
}

.reading-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
  line-height: 1;
}

.reading-unit {
  font-size: 14px;
  color: #606266;
  font-weight: normal;
}

/* 阈值和告警时间信息 */
.sensor-info-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #606266;
}

.info-icon {
  color: #909399;
  flex-shrink: 0;
}

.info-label {
  color: #909399;
  min-width: 60px;
}

.info-value {
  color: #303133;
  flex: 1;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .sensor-card {
    padding: 12px;
  }

  .sensor-icon-wrapper {
    width: 36px;
    height: 36px;
  }

  .sensor-name {
    font-size: 14px;
  }

  .reading-value {
    font-size: 20px;
  }
}
</style>
