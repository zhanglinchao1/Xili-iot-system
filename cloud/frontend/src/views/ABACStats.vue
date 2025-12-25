<template>
  <div class="abac-stats">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>ABAC访问统计</span>
          <el-date-picker
            v-model="timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            @change="fetchStats"
            style="width: 380px"
          />
        </div>
      </template>

      <!-- 概览卡片 -->
      <el-row :gutter="20" class="overview-cards">
        <el-col :xs="12" :sm="12" :lg="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon total">
                <el-icon size="32"><Document /></el-icon>
              </div>
              <div class="stat-data">
                <div class="stat-value">{{ stats?.total_requests || 0 }}</div>
                <div class="stat-label">总请求数</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="12" :sm="12" :lg="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon success">
                <el-icon size="32"><Select /></el-icon>
              </div>
              <div class="stat-data">
                <div class="stat-value">{{ stats?.allowed_requests || 0 }}</div>
                <div class="stat-label">允许请求</div>
                <div class="stat-percent success">{{ (stats?.allow_rate || 0).toFixed(1) }}%</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="12" :sm="12" :lg="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon danger">
                <el-icon size="32"><Close /></el-icon>
              </div>
              <div class="stat-data">
                <div class="stat-value">{{ stats?.denied_requests || 0 }}</div>
                <div class="stat-label">拒绝请求</div>
                <div class="stat-percent danger">{{ (stats?.deny_rate || 0).toFixed(1) }}%</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="12" :sm="12" :lg="6">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-icon warning">
                <el-icon size="32"><TrendCharts /></el-icon>
              </div>
              <div class="stat-data">
                <div class="stat-value">{{ calculateAvgTrust() }}</div>
                <div class="stat-label">平均信任度</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 图表区域 -->
      <el-row :gutter="20" class="charts-row">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>信任度分布</template>
            <div ref="trustChartRef" style="height: 300px"></div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>热点资源TOP5</template>
            <div ref="resourceChartRef" style="height: 300px"></div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="20" class="charts-row">
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>拒绝原因分析</template>
            <div ref="denyChartRef" style="height: 300px"></div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover">
            <template #header>访问结果比例</template>
            <div ref="resultChartRef" style="height: 300px"></div>
          </el-card>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { getAccessStats } from '@/api/abac'
import type { AccessStats } from '@/types/abac'
import * as echarts from 'echarts'

// 数据
const stats = ref<AccessStats | null>(null)
const timeRange = ref<[string, string] | null>(null)

// 图表refs
const trustChartRef = ref<HTMLElement>()
const resourceChartRef = ref<HTMLElement>()
const denyChartRef = ref<HTMLElement>()
const resultChartRef = ref<HTMLElement>()

// 图表实例
let trustChart: echarts.ECharts | null = null
let resourceChart: echarts.ECharts | null = null
let denyChart: echarts.ECharts | null = null
let resultChart: echarts.ECharts | null = null

// 获取统计数据
const fetchStats = async () => {
  try {
    const params: any = {}
    if (timeRange.value && timeRange.value.length === 2) {
      params.start_time = timeRange.value[0]
      params.end_time = timeRange.value[1]
    }

    const res = await getAccessStats(params)
    stats.value = (res as any).data?.stats || (res as any).stats

    // 更新图表
    await nextTick()
    updateCharts()
  } catch (error) {
    ElMessage.error('获取统计数据失败')
  }
}

// 计算平均信任度
const calculateAvgTrust = () => {
  if (!stats.value || !stats.value.trust_score_distribution) return '0.0'

  const dist = stats.value.trust_score_distribution
  const total = dist.range_0_30 + dist.range_30_60 + dist.range_60_80 + dist.range_80_100

  if (total === 0) return '0.0'

  // 使用区间中值计算平均值
  const sum =
    dist.range_0_30 * 15 +
    dist.range_30_60 * 45 +
    dist.range_60_80 * 70 +
    dist.range_80_100 * 90

  return (sum / total).toFixed(1)
}

// 初始化图表
const initCharts = () => {
  if (trustChartRef.value) {
    trustChart = echarts.init(trustChartRef.value)
  }
  if (resourceChartRef.value) {
    resourceChart = echarts.init(resourceChartRef.value)
  }
  if (denyChartRef.value) {
    denyChart = echarts.init(denyChartRef.value)
  }
  if (resultChartRef.value) {
    resultChart = echarts.init(resultChartRef.value)
  }

  // 监听窗口大小变化
  window.addEventListener('resize', () => {
    trustChart?.resize()
    resourceChart?.resize()
    denyChart?.resize()
    resultChart?.resize()
  })
}

// 格式化资源路径显示
const formatResourceName = (resource: string) => {
  // 提取API路径的关键部分
  const parts = resource.replace('/api/v1/', '').split('/')
  if (parts.length <= 2) return parts.join('/')
  // 保留前两个有意义的部分
  return parts.slice(0, 2).join('/')
}

// 更新图表
const updateCharts = () => {
  if (!stats.value) return

  const dist = stats.value.trust_score_distribution
  const hasTrustData = dist && (dist.range_0_30 + dist.range_30_60 + dist.range_60_80 + dist.range_80_100 > 0)

  // 信任度分布柱状图
  if (trustChart) {
    if (hasTrustData) {
      trustChart.setOption({
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'shadow' },
          formatter: (params: any) => {
            const p = params[0]
            const labels = ['低风险(0-30)', '中低(30-60)', '中高(60-80)', '高信任(80-100)']
            return `${labels[p.dataIndex]}<br/>请求数: <b>${p.value}</b>`
          }
        },
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: {
          type: 'category',
          data: ['0-30', '30-60', '60-80', '80-100'],
          axisLabel: { color: '#606266' }
        },
        yAxis: {
          type: 'value',
          name: '请求数',
          axisLabel: { color: '#606266' }
        },
        series: [{
          name: '信任度分布',
          data: [dist.range_0_30, dist.range_30_60, dist.range_60_80, dist.range_80_100],
          type: 'bar',
          barWidth: '50%',
          itemStyle: {
            color: (params: any) => {
              const colors = ['#f56c6c', '#e6a23c', '#409eff', '#67c23a']
              return colors[params.dataIndex]
            },
            borderRadius: [4, 4, 0, 0]
          }
        }]
      })
    } else {
      trustChart.setOption({
        title: { text: '暂无数据', left: 'center', top: 'center', textStyle: { color: '#909399', fontSize: 14 } },
        xAxis: { show: false }, yAxis: { show: false }, series: []
      })
    }
  }

  // 热点资源条形图
  if (resourceChart) {
    const hasResourceData = stats.value.top_resources && stats.value.top_resources.length > 0
    if (hasResourceData) {
      const resources = stats.value.top_resources!.slice().reverse() // 反转让最高的在上面
      resourceChart.setOption({
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'shadow' },
          formatter: (params: any) => {
            const p = params[0]
            const fullPath = stats.value!.top_resources![stats.value!.top_resources!.length - 1 - p.dataIndex].resource
            return `${fullPath}<br/>访问次数: <b>${p.value}</b>`
          }
        },
        grid: { left: '3%', right: '10%', bottom: '3%', containLabel: true },
        xAxis: {
          type: 'value',
          name: '访问次数',
          axisLabel: { color: '#606266' }
        },
        yAxis: {
          type: 'category',
          data: resources.map(r => formatResourceName(r.resource)),
          axisLabel: { color: '#606266', width: 100, overflow: 'truncate' }
        },
        series: [{
          name: '访问次数',
          data: resources.map(r => r.count),
          type: 'bar',
          barWidth: '60%',
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: '#a0cfff' },
              { offset: 1, color: '#409eff' }
            ]),
            borderRadius: [0, 4, 4, 0]
          },
          label: { show: true, position: 'right', formatter: '{c}' }
        }]
      })
    } else {
      resourceChart.setOption({
        title: { text: '暂无数据', left: 'center', top: 'center', textStyle: { color: '#909399', fontSize: 14 } },
        xAxis: { show: false }, yAxis: { show: false }, series: []
      })
    }
  }

  // 拒绝原因饼图
  if (denyChart) {
    const hasDenyData = stats.value.deny_reasons && stats.value.deny_reasons.length > 0
    if (hasDenyData) {
      denyChart.setOption({
        tooltip: {
          trigger: 'item',
          formatter: '{b}: {c}次 ({d}%)'
        },
        legend: {
          orient: 'vertical',
          left: 'left',
          textStyle: { color: '#606266' }
        },
        series: [{
          type: 'pie',
          radius: ['30%', '60%'],
          center: ['60%', '50%'],
          data: stats.value.deny_reasons!.map((r, i) => ({
            name: r.reason,
            value: r.count,
            itemStyle: { color: ['#f56c6c', '#e6a23c', '#909399', '#f78989'][i % 4] }
          })),
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.5)' }
          },
          label: { formatter: '{b}\n{d}%' }
        }]
      })
    } else {
      denyChart.setOption({
        title: { text: '无拒绝记录', left: 'center', top: 'center', textStyle: { color: '#67c23a', fontSize: 14 } },
        series: []
      })
    }
  }

  // 访问结果饼图
  if (resultChart) {
    const total = stats.value.allowed_requests + stats.value.denied_requests
    if (total > 0) {
      resultChart.setOption({
        tooltip: {
          trigger: 'item',
          formatter: '{b}: {c}次 ({d}%)'
        },
        legend: {
          orient: 'vertical',
          left: 'left',
          textStyle: { color: '#606266' }
        },
        series: [{
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['60%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
          label: { show: true, formatter: '{b}\n{d}%' },
          emphasis: {
            label: { show: true, fontSize: 16, fontWeight: 'bold' }
          },
          data: [
            { value: stats.value.allowed_requests, name: '允许', itemStyle: { color: '#67c23a' } },
            { value: stats.value.denied_requests, name: '拒绝', itemStyle: { color: '#f56c6c' } }
          ]
        }]
      })
    } else {
      resultChart.setOption({
        title: { text: '暂无数据', left: 'center', top: 'center', textStyle: { color: '#909399', fontSize: 14 } },
        series: []
      })
    }
  }
}

// 初始化
onMounted(async () => {
  await nextTick()
  initCharts()
  await fetchStats()
})
</script>

<style scoped>
.abac-stats {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.overview-cards {
  margin-bottom: 20px;
}

.stat-card {
  border-radius: 8px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon.total {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.stat-icon.success {
  background: linear-gradient(135deg, #67c23a 0%, #85ce61 100%);
  color: white;
}

.stat-icon.danger {
  background: linear-gradient(135deg, #f56c6c 0%, #f78989 100%);
  color: white;
}

.stat-icon.warning {
  background: linear-gradient(135deg, #e6a23c 0%, #f0c78a 100%);
  color: white;
}

.stat-data {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.stat-percent {
  font-size: 14px;
  font-weight: 600;
  margin-top: 4px;
}

.stat-percent.success {
  color: #67c23a;
}

.stat-percent.danger {
  color: #f56c6c;
}

.charts-row {
  margin-top: 20px;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .abac-stats {
    padding: 12px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .card-header .el-date-picker {
    width: 100% !important;
  }

  .overview-cards {
    margin-bottom: 16px;
  }

  .stat-card {
    margin-bottom: 12px;
  }

  .stat-content {
    gap: 12px;
  }

  .stat-icon {
    width: 44px;
    height: 44px;
  }

  .stat-icon :deep(.el-icon) {
    font-size: 22px !important;
  }

  .stat-value {
    font-size: 22px;
  }

  .stat-label {
    font-size: 13px;
  }

  .stat-percent {
    font-size: 13px;
  }

  .charts-row {
    margin-top: 12px;
  }

  .charts-row .el-card {
    margin-bottom: 12px;
  }

  /* 图表容器高度调整 */
  .charts-row [style*="height: 300px"] {
    height: 250px !important;
  }
}

@media (max-width: 480px) {
  .abac-stats {
    padding: 8px;
  }

  .stat-icon {
    width: 36px;
    height: 36px;
  }

  .stat-icon :deep(.el-icon) {
    font-size: 18px !important;
  }

  .stat-value {
    font-size: 18px;
  }

  .stat-label {
    font-size: 12px;
  }

  /* 图表容器高度进一步调整 */
  .charts-row [style*="height: 300px"] {
    height: 200px !important;
  }
}
</style>
