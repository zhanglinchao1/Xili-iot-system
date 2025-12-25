<template>
  <div class="abac-logs">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>ABAC访问日志</span>
          <el-button type="primary" @click="handleExport">
            <el-icon><Download /></el-icon>
            导出日志
          </el-button>
        </div>
      </template>

      <!-- 筛选器 -->
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="主体类型">
          <el-select v-model="filterForm.subject_type" placeholder="全部" clearable style="width: 150px">
            <el-option label="用户" value="user" />
            <el-option label="储能柜" value="cabinet" />
            <el-option label="设备" value="device" />
          </el-select>
        </el-form-item>
        <el-form-item label="主体ID">
          <el-input v-model="filterForm.subject_id" placeholder="输入主体ID" clearable style="width: 200px" />
        </el-form-item>
        <el-form-item label="资源路径">
          <el-input v-model="filterForm.resource" placeholder="例如: /api/v1/cabinets" clearable style="width: 250px" />
        </el-form-item>
        <el-form-item label="访问结果">
          <el-select v-model="filterForm.allowed" placeholder="全部" clearable style="width: 120px">
            <el-option label="允许" :value="true" />
            <el-option label="拒绝" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 380px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchLogs">查询</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 日志列表 -->
      <el-table :data="logs" border stripe v-loading="loading">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="log-detail">
              <el-descriptions :column="2" border>
                <el-descriptions-item label="策略ID">{{ row.policy_id || '无' }}</el-descriptions-item>
                <el-descriptions-item label="信任度">{{ row.trust_score?.toFixed(2) || 'N/A' }}</el-descriptions-item>
                <el-descriptions-item label="IP地址">{{ row.ip_address || 'N/A' }}</el-descriptions-item>
                <el-descriptions-item label="时间戳">{{ formatDateTime(row.timestamp) }}</el-descriptions-item>
                <el-descriptions-item label="属性" :span="2">
                  <pre v-if="row.attributes">{{ JSON.stringify(row.attributes, null, 2) }}</pre>
                  <span v-else>无</span>
                </el-descriptions-item>
              </el-descriptions>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="id" label="日志ID" width="80" />
        <el-table-column prop="subject_type" label="主体类型" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.subject_type === 'user'" type="primary" size="small">用户</el-tag>
            <el-tag v-else-if="row.subject_type === 'cabinet'" type="success" size="small">储能柜</el-tag>
            <el-tag v-else type="info" size="small">设备</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="subject_id" label="主体ID" min-width="150" show-overflow-tooltip />
        <el-table-column prop="resource" label="资源" min-width="200" show-overflow-tooltip />
        <el-table-column prop="action" label="动作" width="80" />
        <el-table-column prop="allowed" label="结果" width="80">
          <template #default="{ row }">
            <el-tag :type="row.allowed ? 'success' : 'danger'" size="small">
              {{ row.allowed ? '允许' : '拒绝' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="trust_score" label="信任度" width="90">
          <template #default="{ row }">
            <span v-if="row.trust_score !== null && row.trust_score !== undefined">
              {{ row.trust_score.toFixed(1) }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="timestamp" label="时间" width="170">
          <template #default="{ row }">
            {{ formatDateTime(row.timestamp) }}
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.page_size"
        :page-sizes="[20, 50, 100, 200]"
        :total="pagination.total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="fetchLogs"
        @current-change="fetchLogs"
        class="pagination"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listAccessLogs } from '@/api/abac'
import type { AccessLog, AccessLogFilter } from '@/types/abac'

// 数据
const logs = ref<AccessLog[]>([])
const loading = ref(false)

// 分页
const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

// 筛选
const filterForm = reactive<Partial<AccessLogFilter>>({
  subject_type: undefined,
  subject_id: '',
  resource: '',
  allowed: undefined
})

const timeRange = ref<[string, string] | null>(null)

// 获取日志列表
const fetchLogs = async () => {
  loading.value = true
  try {
    const params: AccessLogFilter = {
      page: pagination.page,
      page_size: pagination.page_size,
      ...filterForm
    }

    // 添加时间范围
    if (timeRange.value && timeRange.value.length === 2) {
      params.start_time = timeRange.value[0]
      params.end_time = timeRange.value[1]
    }

    // 清理空字符串
    if (params.subject_id === '') delete params.subject_id
    if (params.resource === '') delete params.resource

    const res = await listAccessLogs(params)
    logs.value = res.data || []
    pagination.total = res.total
  } catch (error) {
    ElMessage.error('获取访问日志失败')
  } finally {
    loading.value = false
  }
}

// 重置筛选
const resetFilter = () => {
  filterForm.subject_type = undefined
  filterForm.subject_id = ''
  filterForm.resource = ''
  filterForm.allowed = undefined
  timeRange.value = null
  pagination.page = 1
  fetchLogs()
}

// 导出日志
const handleExport = () => {
  ElMessage.info('导出功能开发中...')
  // TODO: 实现CSV导出
}

// 格式化日期时间
const formatDateTime = (date: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

// 初始化
onMounted(() => {
  fetchLogs()
})
</script>

<style scoped>
.abac-logs {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-form {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.log-detail {
  padding: 20px;
  background-color: #f5f7fa;
}

.log-detail pre {
  margin: 0;
  padding: 10px;
  background-color: #fff;
  border-radius: 4px;
  overflow-x: auto;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .abac-logs {
    padding: 12px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .card-header .el-button {
    width: 100%;
  }

  .filter-form {
    margin-bottom: 16px;
  }

  .filter-form :deep(.el-form-item) {
    display: block !important;
    margin-right: 0 !important;
    margin-bottom: 12px !important;
    width: 100% !important;
  }

  .filter-form :deep(.el-form-item__label) {
    padding-bottom: 4px;
  }

  .filter-form :deep(.el-form-item__content) {
    width: 100% !important;
  }

  .filter-form :deep(.el-select),
  .filter-form :deep(.el-input),
  .filter-form :deep(.el-date-picker) {
    width: 100% !important;
  }

  .filter-form :deep(.el-form-item:last-child .el-form-item__content) {
    display: flex;
    gap: 8px;
  }

  .filter-form :deep(.el-form-item:last-child .el-button) {
    flex: 1;
  }

  .pagination {
    justify-content: center;
  }

  .log-detail {
    padding: 12px;
  }

  .log-detail pre {
    padding: 8px;
    font-size: 12px;
  }

  /* 描述列表移动端优化 */
  .log-detail :deep(.el-descriptions__cell) {
    padding: 8px !important;
  }

  .log-detail :deep(.el-descriptions__label) {
    min-width: 60px !important;
    font-size: 12px;
  }

  .log-detail :deep(.el-descriptions__content) {
    font-size: 12px;
  }
}

@media (max-width: 480px) {
  .abac-logs {
    padding: 8px;
  }

  .log-detail {
    padding: 8px;
  }
}
</style>
