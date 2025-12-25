<template>
  <div class="traffic-detail-page">
    <el-breadcrumb separator="/" class="breadcrumb">
      <el-breadcrumb-item :to="{ path: '/traffic' }">流量检测</el-breadcrumb-item>
      <el-breadcrumb-item>{{ cabinetId }}</el-breadcrumb-item>
    </el-breadcrumb>

    <div class="page-header">
      <div class="header-content">
        <h1 class="title">{{ cabinetId }} - 流量详情</h1>
        <p class="subtitle">{{ cabinetInfo.location || '未设置位置' }}</p>
      </div>
      <div class="header-actions">
        <el-select v-model="timeRange" placeholder="时间范围" style="width: 150px; margin-right: 12px">
          <el-option label="最近1小时" value="1h" />
          <el-option label="最近6小时" value="6h" />
          <el-option label="最近24小时" value="24h" />
          <el-option label="最近7天" value="7d" />
        </el-select>
        <el-button :icon="Refresh" @click="refreshData">刷新</el-button>
      </div>
    </div>

    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">当前流量</span>
            <el-icon class="stat-icon"><Odometer /></el-icon>
          </div>
          <h3 class="stat-value">{{ stats.currentFlow }}</h3>
          <p class="stat-desc">最新上报 {{ stats.lastUpdate }}</p>
        </div>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">平均延迟</span>
            <el-icon class="stat-icon success"><Connection /></el-icon>
          </div>
          <h3 class="stat-value">{{ stats.latency }}</h3>
          <p class="stat-desc">MQTT网络延迟</p>
        </div>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">丢包率</span>
            <el-icon class="stat-icon warning"><Warning /></el-icon>
          </div>
          <h3 class="stat-value">{{ stats.packetLoss }}</h3>
          <p class="stat-desc">最近一次上报</p>
        </div>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <span class="stat-label">MQTT成功率</span>
            <el-icon class="stat-icon info"><Download /></el-icon>
          </div>
          <h3 class="stat-value">{{ stats.mqttSuccess }}</h3>
          <p class="stat-desc">
            <el-tag :type="stats.anomaly ? 'danger' : 'success'" size="small">
              {{ stats.anomaly ? '存在异常' : '运行正常' }}
            </el-tag>
          </p>
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="charts-row">
      <el-col :xs="24" :lg="16">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">
                <el-icon><TrendCharts /></el-icon>
                流量趋势（{{ timeRangeLabel }}）
              </span>
            </div>
          </template>
          <div ref="trendChartRef" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="8">
        <el-card class="chart-card">
          <template #header>
            <span class="card-title">
              <el-icon><PieChart /></el-icon>
              协议分布
            </span>
          </template>
          <div ref="protocolChartRef" class="chart-container-small"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="alerts-card" v-if="alerts.length > 0">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            <el-icon><Bell /></el-icon>
            最近告警
          </span>
          <el-tag type="danger">{{ alerts.length }} 条告警</el-tag>
        </div>
      </template>

      <el-table :data="alerts" style="width: 100%">
        <el-table-column prop="created_at" label="时间" width="180" />
        <el-table-column prop="alert_type" label="类型" width="150" />
        <el-table-column prop="message" label="描述" />
        <el-table-column label="严重程度" width="120">
          <template #default="{ row }">
            <el-tag :type="getSeverityType(row.severity)">
              {{ getSeverityLabel(row.severity) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.resolved ? 'success' : 'warning'">
              {{ row.resolved ? '已处理' : '未处理' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  Refresh, Odometer, TrendCharts, PieChart, Warning, Download, Connection, Bell
} from '@element-plus/icons-vue'
import api from '@/api'
import * as echarts from 'echarts'
import type { EChartsOption } from 'echarts'
import { cabinetApi } from '@/api'
import type { Cabinet, Alert, ProtocolSlice, TrafficSample } from '@/types/api'

const route = useRoute()
const cabinetId = ref(route.params.id as string)
const cabinetInfo = ref<Cabinet>({} as Cabinet)
const timeRange = ref('1h')

const stats = ref({
  currentFlow: '0 KB/s',
  latency: '0 ms',
  packetLoss: '0%',
  mqttSuccess: '0%',
  anomaly: false,
  lastUpdate: '暂无'
})

const alerts = ref<Alert[]>([])
const trendChartRef = ref<HTMLDivElement>()
const protocolChartRef = ref<HTMLDivElement>()
let trendChart: echarts.ECharts | null = null
let protocolChart: echarts.ECharts | null = null
const trendLabels = ref<string[]>([])
const trendValues = ref<number[]>([])
const protocolSeries = ref<ProtocolSlice[]>([
  { name: 'MQTT', value: 0 },
  { name: 'HTTPS', value: 0 },
])

const timeRangeLabel = computed(() => {
  const map: Record<string, string> = {
    '1h': '最近1小时',
    '6h': '最近6小时',
    '24h': '最近24小时',
    '7d': '最近7天'
  }
  return map[timeRange.value] || '最近1小时'
})

const formatFlow = (kbps: number) => {
  if (!kbps || kbps <= 0) return '0 KB/s'
  const mbps = kbps / 1024
  if (mbps >= 1024) return `${(mbps / 1024).toFixed(2)} GB/s`
  if (mbps >= 1) return `${mbps.toFixed(2)} MB/s`
  return `${kbps.toFixed(0)} KB/s`
}

const normalizeProtocolSlices = (slices?: ProtocolSlice[]) => {
  let mqtt = 0
  let https = 0
  ;(slices || []).forEach((slice) => {
    const name = (slice.name || '').toUpperCase()
    const value = Number(slice.value) || 0
    if (name === 'MQTT') {
      mqtt += value
    } else if (name === 'HTTPS') {
      https += value
    } else {
      https += value
    }
  })
  if (mqtt === 0 && https === 0) {
    return [
      { name: 'MQTT', value: 0 },
      { name: 'HTTPS', value: 0 },
    ]
  }
  return [
    { name: 'MQTT', value: Number(mqtt.toFixed(2)) },
    { name: 'HTTPS', value: Number(https.toFixed(2)) },
  ]
}

const loadCabinetInfo = async () => {
  try {
    const res = await cabinetApi.get(cabinetId.value)
    cabinetInfo.value = (res.data || {}) as Cabinet
  } catch (error) {
    console.error('加载储能柜信息失败:', error)
    cabinetInfo.value = {} as Cabinet
  }
}

const loadTrafficDetail = async () => {
  try {
    console.log('[TrafficDetail] Loading traffic detail with params:', {
      cabinetId: cabinetId.value,
      timeRange: timeRange.value,
      timestamp: new Date().toISOString()
    })
    const res = await api.traffic.getCabinetDetail(cabinetId.value, { range: timeRange.value })
    console.log('[TrafficDetail] API response received:', res)
    const detail = res.data
    if (detail?.stat) {
      const stat = detail.stat
      stats.value = {
        currentFlow: formatFlow(stat.flow_kbps || 0),
        latency: `${(stat.latency_ms || 0).toFixed(1)} ms`,
        packetLoss: `${((stat.packet_loss_rate || 0) * 100).toFixed(2)}%`,
        mqttSuccess: `${((stat.mqtt_success_rate || 0) * 100).toFixed(2)}%`,
        anomaly: stat.risk_level !== 'healthy' && stat.risk_level !== 'low',
        lastUpdate: stat.timestamp ? new Date(stat.timestamp).toLocaleString('zh-CN') : '暂无'
      }
    } else {
      stats.value = {
        currentFlow: '0 KB/s',
        latency: '0 ms',
        packetLoss: '0%',
        mqttSuccess: '0%',
        anomaly: false,
        lastUpdate: '暂无'
      }
    }

    if (detail?.history && detail.history.length > 0) {
      const samples: TrafficSample[] = detail.history
      trendLabels.value = samples.map((sample) =>
        new Date(sample.timestamp).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
      )
      trendValues.value = samples.map((sample) =>
        Number(((sample.flow_kbps || 0) / 1024).toFixed(2))
      )
      console.log('[TrafficDetail] Trend data updated:', {
        labelCount: trendLabels.value.length,
        valueCount: trendValues.value.length,
        sampleData: trendValues.value
      })
    } else {
      console.warn('[TrafficDetail] No history data in response')
      trendLabels.value = []
      trendValues.value = []
    }
    updateTrendChart()

    protocolSeries.value = normalizeProtocolSlices(detail?.protocol)
    updateProtocolChart()
  } catch (error) {
    console.error('加载流量详情失败:', error)
    trendLabels.value = []
    trendValues.value = []
    protocolSeries.value = normalizeProtocolSlices()
    updateTrendChart()
    updateProtocolChart()
  }
}

const loadAlerts = async () => {
  try {
    const res = await cabinetApi.getAlerts(cabinetId.value, { page: 1, page_size: 5 })
    alerts.value = res.data || []
  } catch (error) {
    console.error('加载告警信息失败:', error)
    alerts.value = []
  }
}

const initCharts = () => {
  if (trendChartRef.value) {
    trendChart = echarts.init(trendChartRef.value)
  }
  if (protocolChartRef.value) {
    protocolChart = echarts.init(protocolChartRef.value)
  }
}

const updateTrendChart = () => {
  if (!trendChart) return
  const option: EChartsOption = {
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '5%', top: '5%', containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: trendLabels.value },
    yAxis: { type: 'value', name: '流量 (MB/s)' },
    series: [{
      type: 'line',
      smooth: true,
      areaStyle: {},
      data: trendValues.value
    }]
  }
  console.log('[TrafficDetail] Updating trend chart with option:', option)
  trendChart.setOption(option, { notMerge: true })
}

const updateProtocolChart = () => {
  if (!protocolChart) return
  const hasData = protocolSeries.value.some(item => item.value > 0)
  const option: EChartsOption = {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      label: { show: true, formatter: '{b}\n{d}%' },
      data: hasData ? protocolSeries.value : [{ name: '暂无数据', value: 1 }]
    }]
  }
  protocolChart.setOption(option)
}

const refreshData = async () => {
  await Promise.all([
    loadCabinetInfo(),
    loadTrafficDetail(),
    loadAlerts()
  ])
}

const getSeverityType = (severity: string) => {
  const map: Record<string, 'info' | 'warning' | 'danger'> = {
    info: 'info',
    warning: 'warning',
    error: 'danger',
    critical: 'danger'
  }
  return map[severity] || 'info'
}

const getSeverityLabel = (severity: string) => {
  const map: Record<string, string> = {
    info: '提示',
    warning: '警告',
    error: '错误',
    critical: '严重'
  }
  return map[severity] || severity
}

onMounted(async () => {
  initCharts()
  await refreshData()
})

watch(timeRange, async (newValue, oldValue) => {
  console.log('[TrafficDetail] Time range changed:', { oldValue, newValue })
  await loadTrafficDetail()
})

watch(protocolSeries, () => {
  updateProtocolChart()
}, { deep: true })

watch(() => route.params.id, async (newId) => {
  if (typeof newId === 'string' && newId && newId !== cabinetId.value) {
    cabinetId.value = newId
    await refreshData()
  }
})
</script>

<style scoped>
.traffic-detail-page {
  padding: 24px;
  background: #f8fafc;
  min-height: 100vh;
}

.breadcrumb {
  margin-bottom: 12px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-content .title {
  margin: 0;
  font-size: 26px;
  font-weight: 700;
  color: #0f172a;
}

.header-content .subtitle {
  margin: 4px 0 0 0;
  color: #64748b;
}

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 18px;
  margin-bottom: 16px;
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 15px;
  color: #475569;
}

.stat-icon {
  font-size: 20px;
  color: #3b82f6;
}

.stat-value {
  margin: 0;
  font-size: 32px;
  font-weight: 700;
  color: #0f172a;
}

.stat-desc {
  margin: 4px 0 0 0;
  color: #94a3b8;
}

.charts-row {
  margin-bottom: 24px;
}

.chart-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  margin-bottom: 16px;
}

:deep(.el-card__header) {
  border-bottom: 1px solid #e2e8f0;
  padding: 16px 20px;
  background: #fafbfc;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 17px;
  font-weight: 600;
}

.chart-container {
  height: 320px;
}

.chart-container-small {
  height: 280px;
}

.alerts-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

@media (max-width: 768px) {
  .traffic-detail-page {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .stat-value {
    font-size: 24px;
  }
}
</style>
