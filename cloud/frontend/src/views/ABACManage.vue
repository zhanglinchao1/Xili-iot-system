<template>
  <div class="abac-manage">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>ABAC访问控制策略管理</span>
          <el-button type="primary" @click="handleCreate">
            <el-icon><Plus /></el-icon>
            创建策略
          </el-button>
        </div>
      </template>

      <!-- Tab切换 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 策略管理Tab -->
        <el-tab-pane label="策略管理" name="policies">
          <!-- 筛选器 -->
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="主体类型">
          <el-select v-model="filterForm.subject_type" placeholder="全部" clearable style="width: 150px">
            <el-option label="用户" value="user" />
            <el-option label="储能柜" value="cabinet" />
            <el-option label="设备" value="device" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-select v-model="filterForm.enabled" placeholder="全部" clearable style="width: 150px">
            <el-option label="已启用" :value="true" />
            <el-option label="已禁用" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="filterForm.search" placeholder="搜索策略名称或描述" clearable style="width: 250px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchPolicies">查询</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 策略列表 -->
      <el-table :data="policies" border stripe v-loading="loading">
        <el-table-column prop="name" label="策略名称" min-width="150" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="subject_type" label="主体类型" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.subject_type === 'user'" type="primary">用户</el-tag>
            <el-tag v-else-if="row.subject_type === 'cabinet'" type="success">储能柜</el-tag>
            <el-tag v-else type="info">设备</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="priority" label="优先级" width="80" sortable />
        <el-table-column prop="enabled" label="状态" width="80">
          <template #default="{ row }">
            <el-tooltip
              v-if="row.id === 'policy_admin_full' && row.enabled"
              content="系统核心策略，不能禁用"
              placement="top"
            >
              <el-switch
                v-model="row.enabled"
                disabled
              />
            </el-tooltip>
            <el-switch
              v-else
              v-model="row.enabled"
              @change="handleToggle(row)"
              :disabled="toggleLoading[row.id]"
            />
          </template>
        </el-table-column>
        <el-table-column prop="permissions" label="权限数" width="80">
          <template #default="{ row }">
            {{ row.permissions?.length || 0 }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleView(row)">详情</el-button>
            <el-button size="small" type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button
              v-if="row.subject_type === 'device'"
              size="small"
              type="warning"
              @click="handleDistribute(row)"
            >分发</el-button>
            <el-tooltip
              v-if="row.id === 'policy_admin_full'"
              content="系统核心策略，不能删除"
              placement="top"
            >
              <el-button size="small" type="danger" disabled>删除</el-button>
            </el-tooltip>
            <el-button v-else size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.page_size"
        :page-sizes="[10, 20, 50, 100]"
        :total="pagination.total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="fetchPolicies"
        @current-change="fetchPolicies"
        class="pagination"
      />
        </el-tab-pane>

        <!-- 分发历史Tab -->
        <el-tab-pane label="分发历史" name="distribution">
          <!-- 筛选器 -->
          <el-form :inline="true" :model="distributionFilter" class="filter-form">
            <el-form-item label="策略ID">
              <el-input v-model="distributionFilter.policy_id" placeholder="策略ID" clearable style="width: 200px" />
            </el-form-item>
            <el-form-item label="储能柜ID">
              <el-input v-model="distributionFilter.cabinet_id" placeholder="储能柜ID" clearable style="width: 200px" />
            </el-form-item>
            <el-form-item label="状态">
              <el-select v-model="distributionFilter.status" placeholder="全部" clearable style="width: 120px">
                <el-option label="待确认" value="pending" />
                <el-option label="成功" value="success" />
                <el-option label="失败" value="failed" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="fetchDistributionLogs">查询</el-button>
              <el-button @click="resetDistributionFilter">重置</el-button>
            </el-form-item>
          </el-form>

          <!-- 分发历史列表 -->
          <el-table :data="distributionLogs" border stripe v-loading="loadingDistribution">
            <el-table-column prop="policy_id" label="策略ID" width="180" />
            <el-table-column prop="cabinet_id" label="储能柜ID" width="150" />
            <el-table-column prop="operation_type" label="操作类型" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.operation_type === 'distribute'" type="primary">分发</el-tag>
                <el-tag v-else-if="row.operation_type === 'broadcast'" type="warning">广播</el-tag>
                <el-tag v-else type="info">同步</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag v-if="row.status === 'success'" type="success">成功</el-tag>
                <el-tag v-else-if="row.status === 'pending'" type="warning">待确认</el-tag>
                <el-tag v-else type="danger">失败</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="operator_name" label="操作人" width="120" />
            <el-table-column prop="distributed_at" label="分发时间" width="170">
              <template #default="{ row }">
                {{ formatDate(row.distributed_at) }}
              </template>
            </el-table-column>
            <el-table-column prop="acknowledged_at" label="确认时间" width="170">
              <template #default="{ row }">
                {{ row.acknowledged_at ? formatDate(row.acknowledged_at) : '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="error_message" label="错误信息" show-overflow-tooltip />
          </el-table>

          <!-- 分页 -->
          <el-pagination
            v-model:current-page="distributionPagination.page"
            v-model:page-size="distributionPagination.page_size"
            :page-sizes="[10, 20, 50, 100]"
            :total="distributionPagination.total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="fetchDistributionLogs"
            @current-change="fetchDistributionLogs"
            class="pagination"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="700px"
      :close-on-click-modal="false"
    >
      <el-form :model="policyForm" :rules="rules" ref="formRef" label-width="120px">
        <el-form-item label="策略ID" prop="id" v-if="!isEdit">
          <el-input v-model="policyForm.id" placeholder="例如: policy_xxx" />
        </el-form-item>
        <el-form-item label="策略名称" prop="name">
          <el-input v-model="policyForm.name" placeholder="策略名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="policyForm.description" type="textarea" :rows="2" placeholder="策略描述" />
        </el-form-item>
        <el-form-item label="主体类型" prop="subject_type">
          <el-select v-model="policyForm.subject_type" placeholder="选择主体类型" style="width: 100%">
            <el-option label="用户" value="user" />
            <el-option label="储能柜" value="cabinet" />
            <el-option label="设备" value="device" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级" prop="priority">
          <el-input-number v-model="policyForm.priority" :min="0" :max="100" />
        </el-form-item>
        <el-form-item label="匹配条件">
          <div class="conditions-editor">
            <div v-for="(cond, idx) in policyForm.conditions" :key="idx" class="condition-row">
              <el-select v-model="cond.attribute" placeholder="属性" style="width: 140px">
                <el-option-group label="用户属性" v-if="policyForm.subject_type === 'user'">
                  <el-option value="role" label="角色(role)" />
                  <el-option value="status" label="状态(status)" />
                  <el-option value="trust_score" label="信任度(trust_score)" />
                </el-option-group>
                <el-option-group label="储能柜属性" v-if="policyForm.subject_type === 'cabinet'">
                  <el-option value="status" label="状态(status)" />
                  <el-option value="activation_status" label="激活状态" />
                  <el-option value="trust_score" label="信任度(trust_score)" />
                  <el-option value="vulnerability_score" label="脆弱性评分(vulnerability_score)" />
                  <el-option value="risk_level" label="风险等级(risk_level)" />
                </el-option-group>
                <el-option-group label="设备属性" v-if="policyForm.subject_type === 'device'">
                  <el-option value="status" label="状态(status)" />
                  <el-option value="quality" label="质量(quality)" />
                  <el-option value="trust_score" label="信任度(trust_score)" />
                </el-option-group>
              </el-select>
              <el-select v-model="cond.operator" placeholder="操作符" style="width: 100px">
                <el-option value="eq" label="等于" />
                <el-option value="ne" label="不等于" />
                <el-option value="gt" label="大于" />
                <el-option value="gte" label="大于等于" />
                <el-option value="lt" label="小于" />
                <el-option value="lte" label="小于等于" />
                <el-option value="in" label="包含于" />
              </el-select>
              <el-input v-model="cond.value" placeholder="值" style="width: 150px" />
              <el-button type="danger" :icon="Delete" circle @click="removeCondition(idx)" />
            </div>
            <el-button type="primary" text @click="addCondition">+ 添加条件</el-button>
          </div>
        </el-form-item>
        <el-form-item label="权限列表" prop="permissions">
          <el-select v-model="policyForm.permissions" multiple placeholder="选择或输入权限" allow-create filterable style="width: 100%">
            <el-option value="*" label="所有权限(*)" />
            <el-option value="read:*" label="所有读权限(read:*)" />
            <el-option value="write:*" label="所有写权限(write:*)" />
            <el-option value="read:cabinets" label="读储能柜(read:cabinets)" />
            <el-option value="write:cabinets" label="写储能柜(write:cabinets)" />
            <el-option value="read:sensor_data" label="读传感器数据(read:sensor_data)" />
            <el-option value="write:sensor_data" label="写传感器数据(write:sensor_data)" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailVisible" title="策略详情" width="700px">
      <el-descriptions :column="2" border v-if="currentPolicy">
        <el-descriptions-item label="策略ID">{{ currentPolicy.id }}</el-descriptions-item>
        <el-descriptions-item label="策略名称">{{ currentPolicy.name }}</el-descriptions-item>
        <el-descriptions-item label="主体类型">
          <el-tag v-if="currentPolicy.subject_type === 'user'" type="primary">用户</el-tag>
          <el-tag v-else-if="currentPolicy.subject_type === 'cabinet'" type="success">储能柜</el-tag>
          <el-tag v-else type="info">设备</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="优先级">{{ currentPolicy.priority }}</el-descriptions-item>
        <el-descriptions-item label="启用状态">
          <el-tag :type="currentPolicy.enabled ? 'success' : 'danger'">
            {{ currentPolicy.enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ formatDate(currentPolicy.created_at) }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ currentPolicy.description || '无' }}</el-descriptions-item>
        <el-descriptions-item label="权限列表" :span="2">
          <el-tag v-for="perm in currentPolicy.permissions" :key="perm" style="margin-right: 8px; margin-bottom: 8px">
            {{ perm }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="条件" :span="2">
          <pre style="white-space: pre-wrap">{{ JSON.stringify(currentPolicy.conditions, null, 2) }}</pre>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- 分发策略对话框 -->
    <el-dialog
      v-model="distributeDialogVisible"
      title="分发策略到储能柜"
      width="600px"
      :close-on-click-modal="false"
    >
      <div v-if="distributingPolicy" style="margin-bottom: 20px">
        <el-alert type="info" :closable="false" show-icon>
          <template #title>
            正在分发策略: <strong>{{ distributingPolicy.name }}</strong>
          </template>
        </el-alert>
      </div>

      <el-form label-width="100px">
        <el-form-item label="选择储能柜" required>
          <el-select
            v-model="selectedCabinetIds"
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="请选择储能柜"
            style="width: 100%"
            :loading="loadingCabinets"
          >
            <el-option
              v-for="cabinet in cabinetList"
              :key="cabinet.id"
              :label="`${cabinet.name} (${cabinet.cabinet_id})`"
              :value="cabinet.cabinet_id"
            >
              <span>{{ cabinet.name }}</span>
              <span style="float: right; color: var(--el-text-color-secondary); font-size: 13px">
                {{
                  cabinet.status === 'active' ? '🟢 在线' :
                  cabinet.status === 'offline' ? '⚫ 离线' :
                  cabinet.status === 'pending' ? '🟡 待激活' :
                  cabinet.status === 'inactive' ? '⚪ 已停用' :
                  cabinet.status === 'maintenance' ? '🔵 维护中' : '⚫ 未知'
                }}
              </span>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-text type="info" size="small">
            已选择 {{ selectedCabinetIds.length }} 个储能柜
          </el-text>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="distributeDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="distributeLoading"
          :disabled="selectedCabinetIds.length === 0"
          @click="executeDistribute"
        >
          {{ distributeLoading ? '分发中...' : '确认分发' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import { listPolicies, createPolicy, updatePolicy, deletePolicy, togglePolicy } from '@/api/abac'
import type { AccessPolicy, PolicyListFilter, CreatePolicyRequest, UpdatePolicyRequest } from '@/types/abac'

// Tab切换
const activeTab = ref('policies')

// 数据
const policies = ref<AccessPolicy[]>([])
const loading = ref(false)
const toggleLoading = ref<Record<string, boolean>>({})

// 分页
const pagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

// 分发历史相关
const distributionLogs = ref<any[]>([])
const loadingDistribution = ref(false)
const distributionPagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})
const distributionFilter = reactive({
  policy_id: '',
  cabinet_id: '',
  status: ''
})

// 筛选
const filterForm = reactive<Partial<PolicyListFilter>>({
  subject_type: undefined,
  enabled: undefined,
  search: ''
})

// 对话框
const dialogVisible = ref(false)
const detailVisible = ref(false)
const isEdit = ref(false)
const dialogTitle = ref('')
const currentPolicy = ref<AccessPolicy | null>(null)

// 表单
const formRef = ref<FormInstance>()
const policyForm = reactive<Partial<CreatePolicyRequest>>({
  id: '',
  name: '',
  description: '',
  subject_type: 'user',
  conditions: [],
  permissions: [],
  priority: 50
})

const submitLoading = ref(false)

// 表单验证规则
const rules: FormRules = {
  id: [{ required: true, message: '请输入策略ID', trigger: 'blur' }],
  name: [{ required: true, message: '请输入策略名称', trigger: 'blur' }],
  subject_type: [{ required: true, message: '请选择主体类型', trigger: 'change' }],
  priority: [{ required: true, message: '请输入优先级', trigger: 'blur' }],
  permissions: [{ required: true, message: '请至少选择一个权限', trigger: 'change' }]
}

// 获取策略列表
const fetchPolicies = async () => {
  loading.value = true
  try {
    const params: PolicyListFilter = {
      page: pagination.page,
      page_size: pagination.page_size,
      ...filterForm
    }
    const res = await listPolicies(params)
    policies.value = res.data || []
    pagination.total = res.total
  } catch (error) {
    ElMessage.error('获取策略列表失败')
  } finally {
    loading.value = false
  }
}

// 重置筛选
const resetFilter = () => {
  filterForm.subject_type = undefined
  filterForm.enabled = undefined
  filterForm.search = ''
  pagination.page = 1
  fetchPolicies()
}

// 条件编辑
const addCondition = () => {
  if (!policyForm.conditions) policyForm.conditions = []
  policyForm.conditions.push({ attribute: '', operator: 'eq', value: '' })
}

const removeCondition = (idx: number) => {
  policyForm.conditions?.splice(idx, 1)
}

// 创建策略
const handleCreate = () => {
  isEdit.value = false
  dialogTitle.value = '创建策略'
  Object.assign(policyForm, {
    id: '',
    name: '',
    description: '',
    subject_type: 'user',
    conditions: [],
    permissions: [],
    priority: 50
  })
  dialogVisible.value = true
}

// 编辑策略
const handleEdit = (row: AccessPolicy) => {
  isEdit.value = true
  dialogTitle.value = '编辑策略'
  currentPolicy.value = row
  Object.assign(policyForm, {
    name: row.name,
    description: row.description,
    subject_type: row.subject_type,
    conditions: row.conditions,
    permissions: row.permissions,
    priority: row.priority
  })
  dialogVisible.value = true
}

// 查看详情
const handleView = (row: AccessPolicy) => {
  currentPolicy.value = row
  detailVisible.value = true
}

// 切换启用状态
const handleToggle = async (row: AccessPolicy) => {
  // 保护核心策略：禁止禁用管理员完全访问策略
  if (row.id === 'policy_admin_full' && !row.enabled) {
    row.enabled = true // 恢复状态
    ElMessage.warning('管理员完全访问策略是系统核心策略，不能禁用')
    return
  }
  
  toggleLoading.value[row.id] = true
  try {
    await togglePolicy(row.id)
    ElMessage.success('操作成功')
  } catch (error: any) {
    row.enabled = !row.enabled // 恢复状态
    ElMessage.error(error?.message || '操作失败')
  } finally {
    toggleLoading.value[row.id] = false
  }
}

// 删除策略
const handleDelete = (row: AccessPolicy) => {
  // 保护核心策略：禁止删除管理员完全访问策略
  if (row.id === 'policy_admin_full') {
    ElMessage.warning('管理员完全访问策略是系统核心策略，不能删除')
    return
  }
  
  ElMessageBox.confirm(`确定要删除策略"${row.name}"吗?`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deletePolicy(row.id)
      ElMessage.success('删除成功')
      fetchPolicies()
    } catch (error: any) {
      ElMessage.error(error?.message || '删除失败')
    }
  })
}

// 分发对话框相关
const distributeDialogVisible = ref(false)
const distributingPolicy = ref<AccessPolicy | null>(null)
const selectedCabinetIds = ref<string[]>([])
const cabinetList = ref<any[]>([])
const loadingCabinets = ref(false)
const distributeLoading = ref(false)

// 获取储能柜列表
const fetchCabinets = async () => {
  loadingCabinets.value = true
  try {
    const response = await fetch('/api/v1/cabinets?page=1&page_size=100', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    cabinetList.value = data.data || []
  } catch (error) {
    ElMessage.error('获取储能柜列表失败')
  } finally {
    loadingCabinets.value = false
  }
}

// 打开分发对话框
const handleDistribute = async (row: AccessPolicy) => {
  distributingPolicy.value = row
  selectedCabinetIds.value = []
  distributeDialogVisible.value = true
  await fetchCabinets()
}

// 执行分发
const executeDistribute = async () => {
  if (selectedCabinetIds.value.length === 0) {
    ElMessage.warning('请至少选择一个储能柜')
    return
  }

  distributeLoading.value = true
  try {
    const response = await fetch(`/api/v1/abac/policies/${distributingPolicy.value!.id}/distribute`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ cabinet_ids: selectedCabinetIds.value })
    })

    if (response.ok) {
      const result = await response.json()
      ElMessage.success(`策略已分发到 ${result.data.success_count} 个储能柜`)
      distributeDialogVisible.value = false
    } else {
      const errData = await response.json()
      ElMessage.error(errData.message || '分发失败')
    }
  } catch (error) {
    ElMessage.error('分发操作失败')
  } finally {
    distributeLoading.value = false
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitLoading.value = true
    try {
      if (isEdit.value && currentPolicy.value) {
        const data: UpdatePolicyRequest = {
          name: policyForm.name,
          description: policyForm.description,
          permissions: policyForm.permissions,
          priority: policyForm.priority
        }
        await updatePolicy(currentPolicy.value.id, data)
        ElMessage.success('更新成功')
      } else {
        const data: CreatePolicyRequest = {
          id: policyForm.id!,
          name: policyForm.name!,
          description: policyForm.description,
          subject_type: policyForm.subject_type!,
          conditions: policyForm.conditions || [],
          permissions: policyForm.permissions!,
          priority: policyForm.priority!
        }
        await createPolicy(data)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      fetchPolicies()
    } catch (error) {
      ElMessage.error(isEdit.value ? '更新失败' : '创建失败')
    } finally {
      submitLoading.value = false
    }
  })
}

// 格式化日期
const formatDate = (date: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

// Tab切换处理
const handleTabChange = (tabName: string) => {
  if (tabName === 'distribution') {
    fetchDistributionLogs()
  }
}

// 获取分发历史
const fetchDistributionLogs = async () => {
  loadingDistribution.value = true
  try {
    const params = new URLSearchParams({
      page: String(distributionPagination.page),
      page_size: String(distributionPagination.page_size)
    })

    if (distributionFilter.policy_id) params.append('policy_id', distributionFilter.policy_id)
    if (distributionFilter.cabinet_id) params.append('cabinet_id', distributionFilter.cabinet_id)
    if (distributionFilter.status) params.append('status', distributionFilter.status)

    const response = await fetch(`/api/v1/abac/distribution-logs?${params}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()

    if (data.success) {
      distributionLogs.value = data.data || []
      distributionPagination.total = data.total || 0
    } else {
      ElMessage.error(data.message || '获取分发历史失败')
    }
  } catch (error) {
    ElMessage.error('获取分发历史失败')
  } finally {
    loadingDistribution.value = false
  }
}

// 重置分发历史筛选
const resetDistributionFilter = () => {
  distributionFilter.policy_id = ''
  distributionFilter.cabinet_id = ''
  distributionFilter.status = ''
  distributionPagination.page = 1
  fetchDistributionLogs()
}

// 初始化
onMounted(() => {
  fetchPolicies()
})
</script>

<style scoped>
.abac-manage {
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

.conditions-editor {
  width: 100%;
}

.condition-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .abac-manage {
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

  .filter-form :deep(.el-form-item__content) {
    width: 100% !important;
  }

  .filter-form :deep(.el-select),
  .filter-form :deep(.el-input) {
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

  /* 条件编辑器移动端优化 */
  .condition-row {
    flex-wrap: wrap !important;
    gap: 6px !important;
  }

  .condition-row .el-select,
  .condition-row .el-input {
    width: 100% !important;
    min-width: auto !important;
  }

  /* 对话框中的条件编辑器 */
  :deep(.el-dialog) .condition-row {
    flex-wrap: wrap !important;
  }

  :deep(.el-dialog) .condition-row > * {
    width: 100% !important;
    min-width: auto !important;
    margin-right: 0 !important;
  }
}

@media (max-width: 480px) {
  .abac-manage {
    padding: 8px;
  }
}
</style>
