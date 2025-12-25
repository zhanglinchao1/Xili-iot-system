<template>
  <div class="alert-manage">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="title">告警管理</h1>
        <p class="subtitle">Alarm Management</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" @click="loadData" :loading="loading" :icon="Refresh">
          刷新
        </el-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <el-card class="filter-card" shadow="never">
      <el-form :inline="true" :model="filters" class="filter-form">
        <el-form-item label="储能柜">
          <el-select
            v-model="filters.cabinet_id"
            placeholder="全部储能柜"
            clearable
            filterable
            style="width: 200px"
            @change="loadData"
          >
            <el-option
              v-for="cabinet in cabinetList"
              :key="cabinet.cabinet_id"
              :label="`${cabinet.cabinet_id} - ${cabinet.name}`"
              :value="cabinet.cabinet_id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="严重程度">
          <el-select
            v-model="filters.severity"
            placeholder="全部"
            clearable
            style="width: 150px"
            @change="loadData"
          >
            <el-option label="信息" value="info" />
            <el-option label="警告" value="warning" />
            <el-option label="错误" value="error" />
            <el-option label="严重" value="critical" />
          </el-select>
        </el-form-item>

        <el-form-item label="状态">
          <el-select
            v-model="filters.status"
            placeholder="全部"
            clearable
            style="width: 150px"
            @change="loadData"
          >
            <el-option label="活跃" value="active" />
            <el-option label="已解决" value="resolved" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="loadData">
            <el-icon><Search /></el-icon>
            查询
          </el-button>
          <el-button @click="resetFilters">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 告警列表 -->
    <el-card class="table-card" shadow="never">
      <!-- 工具栏 -->
      <div class="toolbar">
        <el-button
          type="primary"
          :disabled="selectedAlerts.length === 0"
          @click="handleBatchResolve"
        >
          <el-icon><Check /></el-icon>
          批量解决 ({{ selectedAlerts.length }})
        </el-button>
      </div>

      <!-- 表格 -->
      <el-table
        v-loading="loading"
        :data="alertList"
        @selection-change="handleSelectionChange"
        stripe
        style="width: 100%"
      >
        <el-table-column type="selection" width="55" />

        <el-table-column prop="cabinet_id" label="储能柜ID" width="150">
          <template #default="{ row }">
            <el-link type="primary" @click="viewCabinet(row.cabinet_id)">
              {{ row.cabinet_id }}
            </el-link>
          </template>
        </el-table-column>

        <el-table-column prop="location" label="位置" width="150">
          <template #default="{ row }">
            {{ row.location || '未设置' }}
          </template>
        </el-table-column>

        <el-table-column prop="alert_type" label="告警类型" width="180">
          <template #default="{ row }">
            <el-tag>{{ formatAlertType(row.alert_type) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="severity" label="严重程度" width="120">
          <template #default="{ row }">
            <el-tag :type="getSeverityType(row.severity)">
              {{ formatSeverity(row.severity) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="message" label="告警消息" min-width="250" show-overflow-tooltip />

        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'danger' : 'success'">
              {{ row.status === 'active' ? '活跃' : '已解决' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobileView ? 80 : 200" fixed="right">
          <template #default="{ row }">
            <!-- 移动端：下拉菜单 -->
            <el-dropdown v-if="isMobileView" trigger="click" @command="(cmd: string) => handleMobileAction(cmd, row)">
              <el-button type="primary" size="small">
                操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="detail">详情</el-dropdown-item>
                  <el-dropdown-item v-if="row.status === 'active'" command="resolve">解决</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <!-- 桌面端：按钮组 -->
            <template v-else>
            <el-button type="primary" link @click="viewDetail(row)">
              <el-icon><View /></el-icon>
              详情
            </el-button>
            <el-button
              v-if="row.status === 'active'"
              type="success"
              link
              @click="resolveAlert(row)"
            >
              <el-icon><Check /></el-icon>
              解决
            </el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadData"
          @current-change="loadData"
        />
      </div>
    </el-card>

    <!-- 告警详情对话框 -->
    <el-dialog v-model="detailDialogVisible" title="告警详情" width="700px">
      <el-descriptions v-if="currentAlert" :column="2" border>
        <el-descriptions-item label="告警ID">
          {{ currentAlert.alert_id }}
        </el-descriptions-item>
        <el-descriptions-item label="储能柜ID">
          <el-link type="primary" @click="viewCabinet(currentAlert.cabinet_id)">
            {{ currentAlert.cabinet_id }}
          </el-link>
        </el-descriptions-item>
        <el-descriptions-item label="位置">
          {{ currentAlert.location || '未设置' }}
        </el-descriptions-item>
        <el-descriptions-item label="告警类型">
          <el-tag>{{ formatAlertType(currentAlert.alert_type) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="严重程度">
          <el-tag :type="getSeverityType(currentAlert.severity)">
            {{ formatSeverity(currentAlert.severity) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="告警消息" :span="2">
          {{ currentAlert.message }}
        </el-descriptions-item>
        <el-descriptions-item label="设备ID" v-if="currentAlert.device_id">
          {{ currentAlert.device_id }}
        </el-descriptions-item>
        <el-descriptions-item label="传感器数值" v-if="currentAlert.sensor_value">
          {{ currentAlert.sensor_value }}
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="currentAlert.status === 'active' ? 'danger' : 'success'">
            {{ currentAlert.status === 'active' ? '活跃' : '已解决' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">
          {{ formatDateTime(currentAlert.created_at) }}
        </el-descriptions-item>
        <el-descriptions-item label="解决时间" v-if="currentAlert.resolved_at">
          {{ formatDateTime(currentAlert.resolved_at) }}
        </el-descriptions-item>
        <el-descriptions-item label="解决人" v-if="currentAlert.resolved_by">
          {{ currentAlert.resolved_by }}
        </el-descriptions-item>
      </el-descriptions>

      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
        <el-button
          v-if="currentAlert && currentAlert.status === 'active'"
          type="primary"
          @click="resolveFromDetail"
        >
          <el-icon><Check /></el-icon>
          解决告警
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, View, Check, ArrowDown } from '@element-plus/icons-vue'
import * as api from '@/api'
import type { Alert, Cabinet, PaginatedResponse } from '@/types/api'

// 路由
const router = useRouter()

// 移动端检测
const MOBILE_BREAKPOINT = 768
const isMobileView = ref(typeof window !== 'undefined' ? window.innerWidth < MOBILE_BREAKPOINT : false)

function checkMobileView() {
  isMobileView.value = window.innerWidth < MOBILE_BREAKPOINT
}

// 状态
const loading = ref(false)
const cabinetList = ref<Cabinet[]>([])
const alertList = ref<Alert[]>([])
const selectedAlerts = ref<Alert[]>([])
const currentAlert = ref<Alert | null>(null)
const detailDialogVisible = ref(false)

// 筛选条件
const filters = reactive({
  cabinet_id: '',
  severity: '',
  status: 'active', // 默认显示活跃告警
})

// 分页
const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
})

// 加载数据
const loadCabinetsOnce = async () => {
  if (cabinetList.value.length > 0) return
  let page = 1
  const pageSize = 100
  let totalPages = 1
  const allCabinets: Cabinet[] = []

  while (page <= totalPages) {
    const resp: PaginatedResponse<Cabinet> = await api.cabinetApi.list({ page, page_size: pageSize })
    allCabinets.push(...resp.data)
    totalPages = Math.max(1, Math.ceil((resp.total || 0) / resp.page_size))
    page += 1
    if (resp.data.length === 0) break
  }

  cabinetList.value = allCabinets
}

const loadData = async () => {
  loading.value = true
  try {
    await loadCabinetsOnce()

    const params: Record<string, any> = {
      page: pagination.page,
      page_size: pagination.page_size,
    }
    if (filters.cabinet_id) params.cabinet_id = filters.cabinet_id
    if (filters.severity) params.severity = filters.severity
    if (filters.status) params.status = filters.status

    const alertRes = await api.alertApi.list(params)
    alertList.value = alertRes.data || []
    pagination.total = alertRes.total || 0
    pagination.page = alertRes.page || pagination.page
    pagination.page_size = alertRes.page_size || pagination.page_size
  } catch (error: any) {
    console.error('加载数据失败:', error)
    ElMessage.error(error.message || '加载数据失败')
  } finally {
    loading.value = false
  }
}

// 重置筛选条件
const resetFilters = () => {
  filters.cabinet_id = ''
  filters.severity = ''
  filters.status = 'active'
  pagination.page = 1
  loadData()
}

// 表格选择变化
const handleSelectionChange = (selection: Alert[]) => {
  selectedAlerts.value = selection.filter(alert => alert.status === 'active')
}

// 批量解决告警
const handleBatchResolve = async () => {
  if (selectedAlerts.value.length === 0) {
    ElMessage.warning('请选择要解决的告警')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要解决选中的 ${selectedAlerts.value.length} 条告警吗？`,
      '批量解决告警',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const alertIds = selectedAlerts.value.map(alert => String(alert.alert_id))
    await api.alertApi.batchResolve(alertIds)

    ElMessage.success('批量解决告警成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('批量解决告警失败:', error)
      ElMessage.error(error.response?.data?.message || '批量解决告警失败')
    }
  }
}

// 查看告警详情
const viewDetail = async (alert: Alert) => {
  try {
    const res = await api.alertApi.get(String(alert.alert_id))
    currentAlert.value = res.data
    detailDialogVisible.value = true
  } catch (error: any) {
    console.error('获取告警详情失败:', error)
    ElMessage.error(error.response?.data?.message || '获取告警详情失败')
  }
}

// 解决单个告警
const resolveAlert = async (alert: Alert) => {
  try {
    await ElMessageBox.confirm('确定要解决这条告警吗？', '解决告警', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await api.alertApi.resolve(String(alert.alert_id))
    ElMessage.success('告警已解决')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('解决告警失败:', error)
      ElMessage.error(error.response?.data?.message || '解决告警失败')
    }
  }
}

// 从详情对话框解决告警
const resolveFromDetail = async () => {
  if (!currentAlert.value) return

  try {
    await api.alertApi.resolve(String(currentAlert.value.alert_id))
    ElMessage.success('告警已解决')
    detailDialogVisible.value = false
    loadData()
  } catch (error: any) {
    console.error('解决告警失败:', error)
    ElMessage.error(error.response?.data?.message || '解决告警失败')
  }
}

// 查看储能柜详情
const viewCabinet = (cabinetId: string) => {
  router.push(`/cabinets/${cabinetId}`)
}

// 格式化告警类型
const formatAlertType = (type: string) => {
  const typeMap: Record<string, string> = {
    sensor_abnormal: '传感器异常',
    device_offline: '设备离线',
    threshold_exceeded: '阈值超限',
    communication_error: '通信错误',
    data_anomaly: '数据异常',
  }
  return typeMap[type] || type
}

// 格式化严重程度
const formatSeverity = (severity: string) => {
  const severityMap: Record<string, string> = {
    info: '信息',
    warning: '警告',
    error: '错误',
    critical: '严重',
  }
  return severityMap[severity] || severity
}

// 获取严重程度标签类型
const getSeverityType = (severity: string) => {
  const typeMap: Record<string, string> = {
    info: 'info',
    warning: 'warning',
    error: 'danger',
    critical: 'danger',
  }
  return typeMap[severity] || 'info'
}

// 格式化日期时间
const formatDateTime = (datetime: string) => {
  if (!datetime) return '-'
  return new Date(datetime).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// 移动端操作处理
const handleMobileAction = (command: string, row: Alert) => {
  switch (command) {
    case 'detail':
      viewDetail(row)
      break
    case 'resolve':
      resolveAlert(row)
      break
  }
}

// 初始化
onMounted(() => {
  loadData()
  window.addEventListener('resize', checkMobileView)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobileView)
})
</script>

<style scoped>
.alert-manage {
  padding: 24px;
  background: #f8fafc;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-content {
  flex: 1;
}

.page-header .title {
  margin: 0 0 4px 0;
  font-size: 28px;
  color: #0f172a;
  font-weight: 700;
  letter-spacing: -0.5px;
  line-height: 1.2;
}

.page-header .subtitle {
  margin: 0;
  font-size: 15px;
  color: #64748b;
  line-height: 1.5;
}

.header-actions {
  display: flex;
  align-items: center;
}

.filter-card {
  margin-bottom: 20px;
}

.filter-card :deep(.el-form) {
  margin-bottom: 0;
}

.table-card {
  margin-bottom: 20px;
}

.toolbar {
  margin-bottom: 16px;
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .alert-manage {
    padding: 12px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .page-header .title {
    font-size: 20px;
  }

  .page-header .subtitle {
    font-size: 13px;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions .el-button {
    width: 100%;
  }

  .filter-card {
    margin-bottom: 12px;
  }

  .filter-card :deep(.el-form-item) {
    display: block !important;
    margin-right: 0 !important;
    margin-bottom: 12px !important;
    width: 100% !important;
  }

  .filter-card :deep(.el-form-item__content) {
    width: 100% !important;
  }

  .filter-card :deep(.el-select),
  .filter-card :deep(.el-input) {
    width: 100% !important;
  }

  .filter-card :deep(.el-form-item:last-child .el-form-item__content) {
    display: flex;
    gap: 8px;
  }

  .filter-card :deep(.el-form-item:last-child .el-button) {
    flex: 1;
  }

  .table-card {
    margin-bottom: 12px;
  }

  .toolbar {
    margin-bottom: 12px;
  }

  .toolbar .el-button {
    width: 100%;
  }

  .pagination {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .alert-manage {
    padding: 8px;
  }

  .page-header .title {
    font-size: 18px;
  }
}
</style>
