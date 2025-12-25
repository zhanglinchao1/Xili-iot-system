<template>
  <div class="traffic-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="title">流量检测</h1>
        <p class="subtitle">Network Traffic Monitoring</p>
      </div>
      <div class="header-actions">
        <el-select v-model="timeRange" placeholder="时间范围" style="width: 150px; margin-right: 12px">
          <el-option label="最近1小时" value="1h" />
          <el-option label="最近6小时" value="6h" />
          <el-option label="最近24小时" value="24h" />
          <el-option label="最近7天" value="7d" />
        </el-select>
        <el-button type="primary" :icon="Refresh" @click="refreshData">
          刷新
        </el-button>
      </div>
    </div>

    <!-- 流量概览统计 -->
    <el-row :gutter="20" class="overview-row">
      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-icon primary">
            <el-icon :size="24"><Odometer /></el-icon>
          </div>
          <div class="stat-content">
            <h3 class="stat-value">{{ stats.cabinetCount }}</h3>
            <p class="stat-label">监控储能柜</p>
            <span class="stat-trend up">实时</span>
          </div>
        </div>
      </el-col>

      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-icon success">
            <el-icon :size="24"><Download /></el-icon>
          </div>
          <div class="stat-content">
            <h3 class="stat-value">{{ stats.totalFlow }}</h3>
            <p class="stat-label">总流量</p>
            <span class="stat-trend up">Cloud汇总</span>
          </div>
        </div>
      </el-col>

      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-icon info">
            <el-icon :size="24"><Upload /></el-icon>
          </div>
          <div class="stat-content">
            <h3 class="stat-value">{{ stats.avgLatency }}</h3>
            <p class="stat-label">平均延迟</p>
            <span class="stat-trend">上报延迟</span>
          </div>
        </div>
      </el-col>

      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-icon warning">
            <el-icon :size="24"><Warning /></el-icon>
          </div>
          <div class="stat-content">
            <h3 class="stat-value">{{ stats.mqttSuccess }}</h3>
            <p class="stat-label">MQTT成功率</p>
            <span class="stat-trend">{{ stats.anomalies }} 个异常柜</span>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- 流量趋势图表 -->
    <el-row :gutter="20" class="charts-row">
      <el-col :xs="24" :lg="16">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">
                <el-icon><TrendCharts /></el-icon>
                流量趋势
              </span>
            </div>
          </template>
          <div ref="trafficChartRef" class="chart-container"></div>
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

    <!-- 储能柜流量列表 -->
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">储能柜流量监控</span>
          <el-input
            v-model="searchText"
            placeholder="搜索储能柜ID"
            :prefix-icon="Search"
            clearable
            style="width: 200px"
          />
        </div>
      </template>

      <div class="table-wrapper">
        <el-table :data="filteredList" stripe :loading="loading">
          <el-table-column prop="cabinetId" label="储能柜ID" width="150" />
          <el-table-column prop="location" label="位置" width="180" />

          <el-table-column label="流量" width="140">
            <template #default="{ row }">
            <span class="flow-value">{{ row.flowDisplay }}</span>
          </template>
        </el-table-column>

        <el-table-column label="基线对比" width="150">
          <template #default="{ row }">
            <el-tag :type="getBaselineType(row.baselineDeviation)" size="small">
              {{ row.baselineDeviation }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="异常状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.anomaly ? 'danger' : 'success'">
              {{ row.anomaly ? '异常' : '正常' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="lastUpdate" label="最后更新" width="180" />

          <el-table-column label="操作" :width="isMobileView ? 70 : 120" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link size="small" @click="viewDetails(row.cabinetId)">
                {{ isMobileView ? '详情' : '查看详情' }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="filteredList.length"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  Refresh, Odometer, Download, Upload, Warning,
  TrendCharts, PieChart, Search
} from '@element-plus/icons-vue'
import * as echarts from 'echarts'
import type { EChartsOption } from 'echarts'
import api from '@/api'
import type { ProtocolSlice } from '@/types/api'

const router = useRouter()

// 移动端检测
const MOBILE_BREAKPOINT = 768
const isMobileView = ref(typeof window !== 'undefined' ? window.innerWidth < MOBILE_BREAKPOINT : false)

function checkMobileView() {
  isMobileView.value = window.innerWidth < MOBILE_BREAKPOINT
}

// 时间范围和图表类型
const timeRange = ref('1h')

// 统计数据
const stats = ref({
  cabinetCount: 0,
  totalFlow: '--',
  avgLatency: '--',
  mqttSuccess: '--',
  anomalies: 0
})

// 搜索和分页
const searchText = ref('')
const currentPage = ref(1)
const pageSize = ref(20)

// 流量监控列表
interface CabinetTraffic {
  cabinetId: string
  location: string
  flowKbps: number
  flowDisplay: string
  latency: number
  packetLoss: number
  mqttSuccess: number
  anomaly: boolean
  lastUpdate: string
  baselineDeviation: string
}

const loading = ref(false)
const summaryLoading = ref(false)
const trafficList = ref<CabinetTraffic[]>([])

// 过滤后的列表
const filteredList = computed(() => {
  let list = trafficList.value
  if (searchText.value) {
    list = list.filter(item =>
      item.cabinetId.toLowerCase().includes(searchText.value.toLowerCase())
    )
  }
  return list
})

// 获取基线对比类型
const getBaselineType = (deviation: string) => {
  const score = parseFloat(deviation)
  if (!Number.isFinite(score)) return 'info'
  return score >= 80 ? 'success' : score >= 60 ? 'warning' : 'danger'
}

const formatFlow = (kbps: number) => {
  if (!kbps || kbps <= 0) return '0 KB/s'
  // 传入的是 Kbps，需要转换为 KB/s 再格式化
  const kBytesPerSecond = kbps / 8
  const mBytesPerSecond = kBytesPerSecond / 1024

  if (mBytesPerSecond >= 1024) {
    return `${(mBytesPerSecond / 1024).toFixed(2)} GB/s`
  }

  if (mBytesPerSecond >= 1) {
    return `${mBytesPerSecond.toFixed(2)} MB/s`
  }

  if (kBytesPerSecond >= 1) {
    return `${kBytesPerSecond.toFixed(2)} KB/s`
  }

  return `${(kBytesPerSecond * 1024).toFixed(0)} B/s`
}

// 图表引用与数据
const trafficChartRef = ref<HTMLDivElement>()
const protocolChartRef = ref<HTMLDivElement>()
let trafficChart: echarts.ECharts | null = null
let protocolChart: echarts.ECharts | null = null
const trendLabels = ref<string[]>([])
const trendValues = ref<number[]>([])
const trendBaseline = ref<number[]>([])
const protocolSeries = ref<ProtocolSlice[]>([
  { name: 'MQTT', value: 0 },
  { name: 'HTTPS', value: 0 },
])

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

const loadCabinetList = async () => {
  loading.value = true
  try {
    const res = await api.traffic.listCabinets()
    const rows: CabinetTraffic[] = (res.data || []).map((item) => {
      const location = item.location || '未设置'
      const flowKbps = item.flow_kbps || 0
      return {
        cabinetId: item.cabinet_id,
        location,
        flowKbps,
        flowDisplay: formatFlow(flowKbps),
        latency: item.latency_ms || 0,
        packetLoss: (item.packet_loss_rate || 0) * 100,
        mqttSuccess: (item.mqtt_success_rate || 0) * 100,
        anomaly: item.risk_level !== 'healthy' && item.risk_level !== 'low',
        baselineDeviation: item.baseline_deviation || '--',
        lastUpdate: item.timestamp ? new Date(item.timestamp).toLocaleString('zh-CN') : '--'
      }
    })
    trafficList.value = rows
  } finally {
    loading.value = false
  }
}

const loadSummary = async () => {
  summaryLoading.value = true
  try {
    const res = await api.traffic.getSummary({ range: timeRange.value })
    const data = res.data
    if (data?.summary) {
      const summary = data.summary
      const latencyValue = Number(summary.avg_latency_ms) || 0
      const mqttValue = Number(summary.avg_mqtt_success) || 0
      stats.value = {
        cabinetCount: summary.cabinet_count,
        totalFlow: formatFlow(summary.total_flow_kbps),
        avgLatency: `${latencyValue.toFixed(1)} ms`,
        mqttSuccess: `${(mqttValue * 100).toFixed(1)}%`,
        anomalies: summary.anomaly_count
      }
    } else {
      stats.value = {
        cabinetCount: 0,
        totalFlow: '--',
        avgLatency: '--',
        mqttSuccess: '--',
        anomalies: 0
      }
    }

    trendLabels.value = data?.trend?.labels || []
    trendValues.value = data?.trend?.total || []
    trendBaseline.value = data?.trend?.average || []
    protocolSeries.value = normalizeProtocolSlices(data?.protocol)
    updateTrafficChart()
    updateProtocolChart()
  } finally {
    summaryLoading.value = false
  }
}

const initTrafficChart = () => {
  if (!trafficChartRef.value) return
  trafficChart = echarts.init(trafficChartRef.value)
  updateTrafficChart()
}

const updateTrafficChart = () => {
  if (!trafficChart) return
  const option: EChartsOption = {
    tooltip: { trigger: 'axis' },
    legend: { data: ['总流量', '平均值'], bottom: 0 },
    grid: { left: '3%', right: '4%', bottom: '8%', top: '5%', containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: trendLabels.value },
    yAxis: { type: 'value', name: '流量 (MB/s)' },
    series: [
      {
        name: '总流量',
        type: 'line',
        smooth: true,
        data: trendValues.value,
        lineStyle: { width: 3, color: '#3b82f6' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(59, 130, 246, 0.2)' },
            { offset: 1, color: 'rgba(59, 130, 246, 0.01)' }
          ])
        }
      },
      {
        name: '平均值',
        type: 'line',
        smooth: true,
        data: trendBaseline.value,
        lineStyle: { width: 2, color: '#94a3b8', type: 'dashed' }
      }
    ]
  }

  trafficChart.setOption(option)
}

const initProtocolChart = () => {
  if (!protocolChartRef.value) return
  protocolChart = echarts.init(protocolChartRef.value)
  updateProtocolChart()
}

const updateProtocolChart = () => {
  if (!protocolChart) return
  const hasData = protocolSeries.value.some(item => item.value > 0)
  const option: EChartsOption = {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        label: { show: true, formatter: '{b}\n{d}%' },
        data: hasData ? protocolSeries.value : [{ value: 1, name: '暂无数据' }]
      }
    ]
  }
  protocolChart.setOption(option)
}

const refreshData = async (showToast = true) => {
  await Promise.all([
    loadSummary(),
    loadCabinetList(),
  ])
  if (showToast) {
    ElMessage.success('数据已刷新')
  }
}

const viewDetails = (cabinetId: string) => {
  router.push(`/traffic/${cabinetId}`)
}

onMounted(async () => {
  initTrafficChart()
  initProtocolChart()
  await refreshData(false)
  window.addEventListener('resize', checkMobileView)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobileView)
})

watch(timeRange, async () => {
  await refreshData(false)
})

watch(protocolSeries, () => {
  updateProtocolChart()
}, { deep: true })
</script>

<style scoped>
.traffic-page {
  padding: 24px;
  background: #f8fafc;
  min-height: 100vh;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-content .title {
  margin: 0 0 4px 0;
  font-size: 28px;
  color: #0f172a;
  font-weight: 700;
  letter-spacing: -0.5px;
}

.header-content .subtitle {
  margin: 0;
  font-size: 15px;
  color: #64748b;
}

.header-actions {
  display: flex;
  align-items: center;
}

/* 统计卡片 */
.overview-row {
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: white;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  transition: all 0.3s;
  margin-bottom: 16px;
}

.stat-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  transform: translateY(-2px);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.stat-icon.primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-icon.success {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.stat-icon.info {
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
}

.stat-icon.warning {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.stat-content {
  flex: 1;
}

.stat-value {
  margin: 0 0 4px 0;
  font-size: 32px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -1px;
}

.stat-label {
  margin: 0 0 4px 0;
  font-size: 15px;
  color: #64748b;
  font-weight: 500;
}

.stat-trend {
  font-size: 13px;
  font-weight: 600;
  color: #64748b;
}

.stat-trend.up {
  color: #10b981;
}

.stat-trend.down {
  color: #ef4444;
}

/* 图表卡片 */
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
  color: #0f172a;
}

.chart-container {
  height: 320px;
}

.chart-container-small {
  height: 280px;
}

/* 列表卡片 */
.list-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

.table-wrapper {
  overflow-x: auto;
}

/* 流量样式 */
.flow-value {
  font-weight: 600;
  color: #0f172a;
}

/* 分页 */
.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .traffic-page {
    padding: 12px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .header-content .title {
    font-size: 20px;
  }

  .header-content .subtitle {
    font-size: 13px;
  }

  .header-actions {
    width: 100%;
    flex-direction: column;
    gap: 8px;
  }

  .header-actions .el-select {
    width: 100% !important;
    margin-right: 0 !important;
  }

  .header-actions .el-button {
    width: 100%;
  }

  .overview-row {
    margin-bottom: 16px;
  }

  .stat-card {
    padding: 14px;
    gap: 12px;
    margin-bottom: 8px;
  }

  .stat-icon {
    width: 44px;
    height: 44px;
  }

  .stat-value {
    font-size: 24px;
  }

  .stat-label {
    font-size: 13px;
  }

  .stat-trend {
    font-size: 12px;
  }

  .charts-row {
    margin-bottom: 16px;
  }

  .chart-card {
    margin-bottom: 12px;
  }

  .card-title {
    font-size: 15px;
  }

  .chart-container {
    height: 250px;
  }

  .chart-container-small {
    height: 220px;
  }

  .list-card {
    margin-bottom: 12px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .card-header .el-input {
    width: 100% !important;
  }

  .flow-value {
    font-size: 13px;
  }

  .pagination {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .traffic-page {
    padding: 8px;
  }

  .header-content .title {
    font-size: 18px;
  }

  .stat-card {
    padding: 10px;
    gap: 10px;
  }

  .stat-icon {
    width: 36px;
    height: 36px;
  }

  .stat-value {
    font-size: 20px;
  }

  .stat-label {
    font-size: 12px;
  }

  .chart-container {
    height: 200px;
  }

  .chart-container-small {
    height: 180px;
  }
}
</style>
