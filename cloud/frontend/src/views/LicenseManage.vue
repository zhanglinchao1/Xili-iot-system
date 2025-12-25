<template>
  <div class="license-page">
    <div class="page-header">
      <div>
        <h1 class="title">许可控制</h1>
        <p class="subtitle">集中签发、续期与吊销每个储能柜的许可证，并可对Edge端下发同步指令</p>
      </div>
      <div class="header-actions">
        <el-select v-model="filterStatus" placeholder="状态筛选" clearable style="width: 160px" @change="handleFilterChange">
          <el-option label="全部状态" value="" />
          <el-option label="激活" value="active" />
          <el-option label="已过期" value="expired" />
          <el-option label="已吊销" value="revoked" />
        </el-select>
        <el-input
          v-model="searchText"
          placeholder="按储能柜ID或MAC搜索"
          :prefix-icon="Search"
          style="width: 220px; margin-left: 12px"
          clearable
        />
        <el-button type="primary" :icon="Plus" style="margin-left: 12px" @click="openCreateDialog">
          签发许可证
        </el-button>
      </div>
    </div>

    <el-row :gutter="16" class="stat-row">
      <el-col :xs="12" :sm="12" :lg="6" v-for="card in statCards" :key="card.label">
        <div class="stat-card" :class="card.type">
          <div class="stat-value">{{ card.value }}</div>
          <div class="stat-label">{{ card.label }}</div>
        </div>
      </el-col>
    </el-row>

    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <div class="card-title">
            <el-icon><Collection /></el-icon>
            许可证列表
          </div>
          <div class="card-actions">
            <el-button text type="primary" :icon="Refresh" @click="refreshLicenses">刷新</el-button>
            <el-button text type="warning" :loading="syncing" @click="syncHistoricalLicenses">
              同步历史数据
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="filteredLicenses" style="width: 100%" stripe v-loading="loading">
        <el-table-column prop="licenseId" label="许可证ID" width="180">
          <template #default="{ row }">
            <el-tooltip :content="row.licenseId" placement="top">
              <span class="license-id-cell">{{ row.licenseId }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="cabinetId" label="储能柜ID" width="150">
          <template #default="{ row }">
            <div class="cabinet-id">
              <span>{{ row.cabinetId }}</span>
              <small class="cabinet-name">{{ cabinetMap[row.cabinetId]?.name || '未命名' }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="macAddress" label="MAC地址" width="160" />
        <el-table-column label="状态" width="130">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="expiresAt" label="过期时间" width="180">
          <template #default="{ row }">
            <span :class="{ expired: row.status === 'expired' }">
              {{ formatDate(row.expiresAt) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="权限" min-width="220">
          <template #default="{ row }">
            <el-tag v-for="perm in row.permissions" :key="perm" type="info" size="small" effect="plain" style="margin-right: 4px">
              {{ perm }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Edge同步" width="210">
          <template #default="{ row }">
            <el-popover
              v-if="row.sync"
              placement="top"
              trigger="hover"
              :width="280"
            >
              <template #reference>
                <div class="sync-status">
                  <el-tag size="small" :type="syncStatusTagType(row.sync.status)">
                    {{ getCommandStatusLabel(row.sync.status, row.sync.commandType) }}
                  </el-tag>
                  <div class="sync-meta">
                    {{ row.sync.commandType === 'license_revoke' ? '吊销' : '下发' }} · {{ formatDate(row.sync.updatedAt || row.sync.completedAt || row.sync.sentAt) }}
                  </div>
                  <div v-if="row.sync.result" class="sync-result">
                    {{ row.sync.result }}
                  </div>
                </div>
              </template>
              <div v-if="row.sync.history?.length">
                <div
                  v-for="entry in row.sync.history"
                  :key="entry.commandId"
                  class="sync-history-item"
                >
                  <div class="history-header">
                    <span>{{ commandTypeLabel(entry.commandType) }}</span>
                    <el-tag size="small" :type="syncStatusTagType(entry.status)">
                      {{ getCommandStatusLabel(entry.status, entry.commandType) }}
                    </el-tag>
                  </div>
                  <div class="history-meta">{{ formatDate(entry.updatedAt) }}</div>
                  <div v-if="entry.message" class="history-message">{{ entry.message }}</div>
                </div>
              </div>
              <div v-else class="text-muted">
                暂无命令记录
              </div>
            </el-popover>
            <span v-else class="text-muted">--</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" :width="isMobileView ? 80 : 300" fixed="right">
          <template #default="{ row }">
            <!-- 移动端：下拉菜单 -->
            <el-dropdown v-if="isMobileView" trigger="click" @command="(cmd: string) => handleMobileAction(cmd, row)">
              <el-button type="primary" size="small">
                操作<el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="detail">详情</el-dropdown-item>
                  <template v-if="row.status === 'revoked'">
                    <el-dropdown-item command="reissue">重新签发</el-dropdown-item>
                    <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                  </template>
                  <template v-else>
                    <el-dropdown-item command="renew">续期</el-dropdown-item>
                    <el-dropdown-item command="revoke">吊销</el-dropdown-item>
                    <el-dropdown-item command="push">下发</el-dropdown-item>
                    <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                  </template>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <!-- 桌面端：按钮组 -->
            <template v-else>
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <template v-if="row.status === 'revoked'">
              <el-button link type="success" @click="reissue(row)">重新签发</el-button>
              <el-button link type="danger" @click="deleteLicense(row)">删除</el-button>
            </template>
            <template v-else>
              <el-button link type="success" @click="openRenewDialog(row)">续期</el-button>
              <el-button link type="danger" @click="openRevokeDialog(row)">吊销</el-button>
              <el-button link type="primary" @click="pushLicense(row)">下发</el-button>
              <el-button link type="danger" @click="deleteLicense(row)">删除</el-button>
              </template>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </el-card>

    <el-drawer
      v-model="detailVisible"
      :title="detailLicense ? `储能柜 ${detailLicense?.cabinetId} 许可证详情` : ''"
      size="420px"
    >
      <el-descriptions v-if="detailLicense" :column="1" border>
        <el-descriptions-item label="储能柜">
          {{ detailLicense.cabinetId }} / {{ cabinetMap[detailLicense.cabinetId]?.name || '未命名' }}
        </el-descriptions-item>
        <el-descriptions-item label="MAC地址">
          {{ detailLicense.macAddress }}
        </el-descriptions-item>
        <el-descriptions-item label="签发时间">
          {{ formatDate(detailLicense.issuedAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="到期时间">
          {{ formatDate(detailLicense.expiresAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="最大设备数">
          {{ detailLicense.maxDevices }}
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusTagType(detailLicense.status)">
            {{ getStatusLabel(detailLicense.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建人">
          {{ detailLicense.createdBy || '系统' }}
        </el-descriptions-item>
        <el-descriptions-item label="权限">
          <el-space wrap>
            <el-tag v-for="perm in detailLicense.permissions" :key="perm" size="small" effect="dark">
              {{ perm }}
            </el-tag>
          </el-space>
        </el-descriptions-item>
        <el-descriptions-item v-if="detailLicense.revokeReason" label="吊销原因">
          {{ detailLicense.revokeReason }}
        </el-descriptions-item>
      </el-descriptions>
    </el-drawer>

    <el-dialog v-model="createDialogVisible" title="签发许可证" width="520px">
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="110px">
        <el-form-item label="储能柜" prop="cabinet_id">
          <el-select v-model="createForm.cabinet_id" placeholder="选择储能柜" filterable>
            <el-option
              v-for="cabinet in cabinetOptions"
              :key="cabinet.cabinet_id"
              :label="`${cabinet.cabinet_id}（${cabinet.name || '未命名'}）`"
              :value="cabinet.cabinet_id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="有效天数" prop="valid_days">
          <el-input-number v-model="createForm.valid_days" :min="1" :max="365" />
        </el-form-item>
        <el-form-item label="最大设备数" prop="max_devices">
          <el-input-number v-model="createForm.max_devices" :min="1" :max="1000" />
        </el-form-item>
        <el-form-item label="权限" prop="permissions">
          <el-select v-model="createForm.permissions" multiple placeholder="选择权限" style="width: 100%">
            <el-option v-for="perm in permissionOptions" :key="perm.value" :label="perm.label" :value="perm.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitCreate">签发</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="renewDialogVisible" title="续期许可证" width="420px">
      <el-form :model="renewForm" :rules="renewRules" ref="renewFormRef" label-width="120px">
        <el-form-item label="储能柜">
          {{ targetLicense?.cabinetId }}
        </el-form-item>
        <el-form-item label="延长天数" prop="extend_days">
          <el-input-number v-model="renewForm.extend_days" :min="1" :max="365" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renewDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitRenew">确认续期</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="revokeDialogVisible" title="吊销许可证" width="420px">
      <el-form :model="revokeForm" :rules="revokeRules" ref="revokeFormRef" label-width="100px">
        <el-form-item label="储能柜">
          {{ targetLicense?.cabinetId }}
        </el-form-item>
        <el-form-item label="吊销原因" prop="reason">
          <el-input
            v-model="revokeForm.reason"
            type="textarea"
            placeholder="说明吊销原因"
            :rows="3"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="revokeDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="saving" @click="submitRevoke">确认吊销</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="reissueDialogVisible" title="重新签发许可证" width="520px">
      <el-alert
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 16px"
      >
        <template #title>
          将先删除已吊销的旧许可证，然后创建新许可证并自动下发到Edge端
        </template>
      </el-alert>
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="110px">
        <el-form-item label="储能柜">
          <el-input :value="createForm.cabinet_id" disabled />
        </el-form-item>
        <el-form-item label="有效天数" prop="valid_days">
          <el-input-number v-model="createForm.valid_days" :min="1" :max="365" />
        </el-form-item>
        <el-form-item label="最大设备数" prop="max_devices">
          <el-input-number v-model="createForm.max_devices" :min="1" :max="1000" />
        </el-form-item>
        <el-form-item label="权限" prop="permissions">
          <el-select v-model="createForm.permissions" multiple placeholder="选择权限" style="width: 100%">
            <el-option v-for="perm in permissionOptions" :key="perm.value" :label="perm.label" :value="perm.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="reissueDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitReissue">确认重新签发</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  Refresh, Search, Plus, Collection, ArrowDown
} from '@element-plus/icons-vue'
import { licenseApi, cabinetApi, commandApi } from '@/api'
import type {
  License,
  CreateLicenseRequest,
  RenewLicenseRequest,
  RevokeLicenseRequest,
  PaginatedResponse,
  Cabinet,
  Command
} from '@/types/api'

interface LicenseRow {
  licenseId: string
  cabinetId: string
  macAddress: string
  issuedAt: string
  expiresAt: string
  status: 'active' | 'expired' | 'revoked'
  permissions: string[]
  maxDevices: number
  revokedAt?: string
  revokeReason?: string
  createdBy: string
  createdAt: string
  updatedAt: string
  sync?: LicenseSyncState
}

interface LicenseSyncState {
  commandId: string
  status: Command['status']
  commandType: Command['command_type']
  updatedAt?: string
  sentAt?: string
  completedAt?: string
  result?: string
  history?: CommandHistoryEntry[]
}

interface CommandHistoryEntry {
  commandId: string
  commandType: Command['command_type']
  status: Command['status']
  updatedAt?: string
  message?: string
}

interface CabinetOption {
  cabinet_id: string
  name?: string
}

const loading = ref(false)
const saving = ref(false)
const syncing = ref(false)

// 移动端检测
const MOBILE_BREAKPOINT = 768
const isMobileView = ref(typeof window !== 'undefined' ? window.innerWidth < MOBILE_BREAKPOINT : false)

function checkMobileView() {
  isMobileView.value = window.innerWidth < MOBILE_BREAKPOINT
}

onMounted(() => {
  window.addEventListener('resize', checkMobileView)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobileView)
})
const licenses = ref<LicenseRow[]>([])
const filterStatus = ref<string | ''>('')
const searchText = ref('')
const detailVisible = ref(false)
const detailLicense = ref<LicenseRow | null>(null)
const targetLicense = ref<LicenseRow | null>(null)

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0,
})

const createDialogVisible = ref(false)
const renewDialogVisible = ref(false)
const revokeDialogVisible = ref(false)
const reissueDialogVisible = ref(false)
const createFormRef = ref<FormInstance>()
const renewFormRef = ref<FormInstance>()
const revokeFormRef = ref<FormInstance>()

const permissionOptions = [
  { label: '传感器读写', value: 'sensor:rw' },
  { label: '告警管理', value: 'alert:manage' },
  { label: '命令下发', value: 'command:send' },
  { label: '配置调整', value: 'config:update' },
]

const cabinetOptions = ref<CabinetOption[]>([])
const cabinetMap = reactive<Record<string, { name?: string; location?: string }>>({})

const createForm = reactive<CreateLicenseRequest>({
  cabinet_id: '',
  valid_days: 90,
  max_devices: 100,
  permissions: ['sensor:rw'],
})

const renewForm = reactive<RenewLicenseRequest>({
  extend_days: 30,
})

const revokeForm = reactive<RevokeLicenseRequest>({
  reason: '',
})

const createRules: FormRules = {
  cabinet_id: [{ required: true, message: '请选择储能柜', trigger: 'change' }],
  valid_days: [{ required: true, message: '请输入有效天数', trigger: 'change' }],
  max_devices: [{ required: true, message: '请输入最大设备数', trigger: 'change' }],
  permissions: [{ required: true, type: 'array', min: 1, message: '至少选择一个权限', trigger: 'change' }],
}

const renewRules: FormRules = {
  extend_days: [{ required: true, message: '请输入延长天数', trigger: 'change' }],
}

const revokeRules: FormRules = {
  reason: [{ required: true, message: '请输入吊销原因', trigger: 'blur' }],
}

const statCards = computed(() => {
  const total = licenses.value.length
  const active = licenses.value.filter(item => item.status === 'active').length
  const expired = licenses.value.filter(item => item.status === 'expired').length
  const revoked = licenses.value.filter(item => item.status === 'revoked').length
  return [
    { label: '许可证总数', value: total, type: 'primary' },
    { label: '激活', value: active, type: 'success' },
    { label: '已过期', value: expired, type: 'warning' },
    { label: '已吊销', value: revoked, type: 'danger' },
  ]
})

const filteredLicenses = computed(() => {
  const keyword = searchText.value.trim().toLowerCase()
  return licenses.value.filter((item) => {
    if (filterStatus.value && item.status !== filterStatus.value) {
      return false
    }
    if (!keyword) return true
    return item.cabinetId.toLowerCase().includes(keyword) || item.macAddress.toLowerCase().includes(keyword)
  })
})

const normalizePermissions = (value: License['permissions']): string[] => {
  if (Array.isArray(value)) return value
  if (!value) return []
  try {
    const parsed = JSON.parse(value)
    if (Array.isArray(parsed)) {
      return parsed.map((item: any) => String(item))
    }
    if (typeof parsed === 'object' && parsed !== null) {
      return Object.keys(parsed).filter((key) => parsed[key])
    }
  } catch {
    return value.split(',').map(item => item.trim()).filter(Boolean)
  }
  return []
}

const deriveStatus = (status: string, expiresAt: string): 'active' | 'expired' | 'revoked' => {
  if (status === 'revoked') return 'revoked'
  if (new Date(expiresAt).getTime() < Date.now()) {
    return 'expired'
  }
  return status === 'active' ? 'active' : 'expired'
}

const mapLicense = (license: License): LicenseRow => ({
  licenseId: license.license_id,
  cabinetId: license.cabinet_id,
  macAddress: license.mac_address,
  issuedAt: license.issued_at,
  expiresAt: license.expires_at,
  status: deriveStatus(license.status, license.expires_at),
  permissions: normalizePermissions(license.permissions),
  maxDevices: license.max_devices || 0,
  revokedAt: license.revoked_at,
  revokeReason: license.revoke_reason,
  createdBy: license.created_by,
  createdAt: license.created_at,
  updatedAt: license.updated_at,
})

const getTimestamp = (value?: string) => (value ? new Date(value).getTime() : 0)

const buildSyncStateMap = (commands: Command[], cabinetIds: string[]): Record<string, LicenseSyncState> => {
  const cabinetSet = new Set(cabinetIds)
  const grouped: Record<string, Command[]> = {}
  commands.forEach(command => {
    if (!cabinetSet.has(command.cabinet_id)) return
    if (!grouped[command.cabinet_id]) {
      grouped[command.cabinet_id] = []
    }
    grouped[command.cabinet_id].push(command)
  })

  const result: Record<string, LicenseSyncState> = {}
  Object.entries(grouped).forEach(([cabinetId, list]) => {
    const sorted = [...list].sort((a, b) => getTimestamp(b.updated_at || b.completed_at || b.sent_at || b.created_at) - getTimestamp(a.updated_at || a.completed_at || a.sent_at || a.created_at))
    const latest = sorted[0]
    result[cabinetId] = {
      commandId: latest.command_id,
      status: latest.status,
      commandType: latest.command_type,
      updatedAt: latest.updated_at || latest.completed_at || latest.sent_at || latest.created_at,
      sentAt: latest.sent_at,
      completedAt: latest.completed_at,
      result: latest.result,
      history: sorted.slice(0, 5).map(item => ({
        commandId: item.command_id,
        commandType: item.command_type,
        status: item.status,
        updatedAt: item.updated_at || item.completed_at || item.sent_at || item.created_at,
        message: item.result,
      })),
    }
  })

  return result
}

const loadLicenseSyncStates = async (cabinetIds: string[]) => {
  if (!cabinetIds.length) {
    return {}
  }
  const fetchCommands = async (commandType: 'license_push' | 'license_revoke') => {
    try {
      const resp: PaginatedResponse<Command> = await commandApi.list({
        page: 1,
        page_size: 100,
        command_type: commandType,
      })
      return resp.data
    } catch (error) {
      console.warn(`加载${commandType}命令失败`, error)
      return []
    }
  }

  const [pushCommands, revokeCommands] = await Promise.all([
    fetchCommands('license_push'),
    fetchCommands('license_revoke'),
  ])

  return buildSyncStateMap([...pushCommands, ...revokeCommands], cabinetIds)
}

const loadCabinets = async () => {
  try {
    const resp: PaginatedResponse<Cabinet> = await cabinetApi.list({ page: 1, page_size: 100 })
    cabinetOptions.value = resp.data
    resp.data.forEach((item) => {
      cabinetMap[item.cabinet_id] = {
        name: item.name,
        location: item.location,
      }
    })
  } catch (error) {
    console.warn('加载储能柜列表失败', error)
  }
}

const loadLicenses = async () => {
  loading.value = true
  try {
    const response: PaginatedResponse<License> = await licenseApi.list({
      page: pagination.page,
      page_size: pagination.pageSize,
      status: filterStatus.value || undefined,
    })
    const mapped = response.data.map(mapLicense)
    const syncState = await loadLicenseSyncStates(mapped.map(item => item.cabinetId))
    licenses.value = mapped.map(item => ({
      ...item,
      sync: syncState[item.cabinetId],
    }))
    pagination.total = response.total
    pagination.page = response.page
    pagination.pageSize = response.page_size
  } catch (error: any) {
    ElMessage.error(error?.message || '加载许可证失败')
  } finally {
    loading.value = false
  }
}

const refreshLicenses = async () => {
  await loadLicenses()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadLicenses()
}

const handleSizeChange = (size: number) => {
  pagination.pageSize = size
  loadLicenses()
}

const handleFilterChange = () => {
  pagination.page = 1
  loadLicenses()
}

const openCreateDialog = () => {
  createForm.cabinet_id = ''
  createForm.valid_days = 90
  createForm.permissions = ['sensor:rw']
  createDialogVisible.value = true
}

const submitCreate = async () => {
  if (!createFormRef.value) return
  await createFormRef.value.validate()
  saving.value = true
  try {
    await licenseApi.create({ ...createForm })
    ElMessage.success('许可证签发成功')
    createDialogVisible.value = false
    await loadLicenses()
  } catch (error: any) {
    ElMessage.error(error?.message || '签发失败')
  } finally {
    saving.value = false
  }
}

const submitReissue = async () => {
  if (!createFormRef.value || !targetLicense.value) return
  await createFormRef.value.validate()
  saving.value = true

  const cabinetId = targetLicense.value.cabinetId

  try {
    // 步骤1: 删除已吊销的旧许可证
    await licenseApi.delete(cabinetId)

    // 步骤2: 创建新许可证
    await licenseApi.create({ ...createForm })

    // 步骤3: 自动下发到Edge端
    await licenseApi.push(cabinetId)

    ElMessage.success('许可证重新签发成功并已下发到Edge端')
    reissueDialogVisible.value = false
    await loadLicenses()
  } catch (error: any) {
    ElMessage.error(error?.message || '重新签发失败')
  } finally {
    saving.value = false
  }
}

const openRenewDialog = (row: LicenseRow) => {
  targetLicense.value = row
  renewForm.extend_days = 30
  renewDialogVisible.value = true
}

const submitRenew = async () => {
  if (!renewFormRef.value || !targetLicense.value) return
  await renewFormRef.value.validate()
  saving.value = true
  try {
    await licenseApi.renew(targetLicense.value.cabinetId, { ...renewForm })
    ElMessage.success('续期成功')
    renewDialogVisible.value = false
    await loadLicenses()
  } catch (error: any) {
    ElMessage.error(error?.message || '续期失败')
  } finally {
    saving.value = false
  }
}

const openRevokeDialog = (row: LicenseRow) => {
  targetLicense.value = row
  revokeForm.reason = ''
  revokeDialogVisible.value = true
}

const submitRevoke = async () => {
  if (!revokeFormRef.value || !targetLicense.value) return
  await revokeFormRef.value.validate()
  saving.value = true
  try {
    await licenseApi.revoke(targetLicense.value.cabinetId, { ...revokeForm })
    ElMessage.success('许可证已吊销')
    revokeDialogVisible.value = false
    await loadLicenses()
  } catch (error: any) {
    ElMessage.error(error?.message || '吊销失败')
  } finally {
    saving.value = false
  }
}

const pushLicense = async (row: LicenseRow) => {
  try {
    await licenseApi.push(row.cabinetId)
    ElMessage.success('已生成并下发许可证')
    await loadLicenses()
  } catch (error: any) {
    ElMessage.error(error?.message || '命令下发失败')
  }
}

const syncHistoricalLicenses = async () => {
  try {
    await ElMessageBox.confirm('将为尚未创建许可证的储能柜自动生成许可证，确定继续吗？', '同步历史许可证', {
      confirmButtonText: '同步',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }

  syncing.value = true
  try {
    const resp = await licenseApi.sync()
    ElMessage.success(resp.message || '同步完成')
    await loadLicenses()
  } catch (error: any) {
    ElMessage.error(error?.message || '同步失败')
  } finally {
    syncing.value = false
  }
}

const openDetail = (row: LicenseRow) => {
  detailLicense.value = row
  detailVisible.value = true
}

const deleteLicense = (row: LicenseRow) => {
  ElMessageBox.confirm(`确定删除储能柜 ${row.cabinetId} 的许可证吗？该操作不可恢复。`, '删除许可证', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning',
  }).then(async () => {
    try {
      await licenseApi.delete(row.cabinetId)
      ElMessage.success('许可证已删除')
      await loadLicenses()
    } catch (error: any) {
      ElMessage.error(error?.message || '删除失败')
    }
  }).catch(() => {})
}

// 移动端操作处理
const handleMobileAction = (command: string, row: LicenseRow) => {
  switch (command) {
    case 'detail':
      openDetail(row)
      break
    case 'renew':
      openRenewDialog(row)
      break
    case 'revoke':
      openRevokeDialog(row)
      break
    case 'push':
      pushLicense(row)
      break
    case 'delete':
      deleteLicense(row)
      break
    case 'reissue':
      reissue(row)
      break
  }
}

const reissue = async (row: LicenseRow) => {
  try {
    await ElMessageBox.confirm(
      `储能柜 ${row.cabinetId} 的许可证已吊销，是否重新签发？\n\n操作将:\n1. 删除已吊销的旧许可证\n2. 创建新的许可证\n3. 自动下发到Edge端`,
      '重新签发许可证',
      {
        confirmButtonText: '重新签发',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: false,
      }
    )
  } catch {
    return
  }

  // 打开重新签发对话框,预填储能柜信息
  targetLicense.value = row
  createForm.cabinet_id = row.cabinetId
  createForm.valid_days = 90
  createForm.max_devices = row.maxDevices
  createForm.permissions = row.permissions.length > 0 ? row.permissions : ['sensor:rw']
  reissueDialogVisible.value = true
}

const statusTagType = (status: LicenseRow['status']) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'expired':
      return 'warning'
    case 'revoked':
      return 'danger'
    default:
      return 'info'
  }
}

const getStatusLabel = (status: LicenseRow['status']) => {
  switch (status) {
    case 'active':
      return '激活'
    case 'expired':
      return '已过期'
    case 'revoked':
      return '已吊销'
    default:
      return status
  }
}

const syncStatusTagType = (status?: Command['status']) => {
  switch (status) {
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    case 'timeout':
      return 'warning'
    case 'sent':
      return 'info'
    case 'pending':
    default:
      return 'info'
  }
}

const getCommandStatusLabel = (status?: Command['status'], commandType?: string) => {
  const action = commandType === 'license_revoke' ? '吊销' : '同步'
  switch (status) {
    case 'pending':
      return `${action}待发送`
    case 'sent':
      return `${action}已发送`
    case 'success':
      return `${action}完成`
    case 'failed':
      return `${action}失败`
    case 'timeout':
      return `${action}超时`
    default:
      return '无命令'
  }
}

const commandTypeLabel = (type?: string) => {
  switch (type) {
    case 'license_push':
    case 'license_update':
      return '下发许可证'
    case 'license_revoke':
      return '吊销许可证'
    default:
      return type || '未知命令'
  }
}

const formatDate = (value?: string) => {
  if (!value) return '--'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

onMounted(async () => {
  await Promise.all([loadCabinets(), loadLicenses()])
})

watch(searchText, () => {
  pagination.page = 1
})
</script>

<style scoped>
.license-page {
  padding: 24px;
  background: #f8fafc;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.title {
  margin: 0;
  font-size: 26px;
  color: #0f172a;
}

.subtitle {
  margin: 6px 0 0 0;
  color: #64748b;
}

.header-actions {
  display: flex;
  align-items: center;
}

.stat-row {
  margin-bottom: 20px;
}

.stat-card {
  border-radius: 12px;
  padding: 20px;
  background: white;
  border: 1px solid #e2e8f0;
  box-shadow: 0 4px 20px rgba(15, 23, 42, 0.06);
}

.stat-card .stat-value {
  font-size: 30px;
  font-weight: 700;
  color: #0f172a;
}

.stat-card .stat-label {
  font-size: 14px;
  color: #64748b;
}

.stat-card.primary .stat-value {
  color: #2563eb;
}

.stat-card.success .stat-value {
  color: #10b981;
}

.stat-card.warning .stat-value {
  color: #f97316;
}

.stat-card.danger .stat-value {
  color: #ef4444;
}

.list-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-actions {
  display: flex;
  gap: 8px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #0f172a;
}

.pagination {
  margin-top: 16px;
  text-align: right;
}

.cabinet-id {
  display: flex;
  flex-direction: column;
}

.cabinet-name {
  color: #94a3b8;
  font-size: 12px;
}

.expired {
  color: #f97316;
}

.stat-card.danger {
  border-color: #fecaca;
}

.sync-status {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sync-meta {
  font-size: 12px;
  color: #94a3b8;
}

.sync-result {
  font-size: 12px;
  color: #64748b;
}

.sync-history-item {
  border-bottom: 1px solid #e2e8f0;
  padding: 8px 0;
}

.sync-history-item:last-child {
  border-bottom: none;
}

.history-header {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  color: #0f172a;
}

.history-meta {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 2px;
}

.history-message {
  font-size: 12px;
  color: #475569;
  margin-top: 4px;
  word-break: break-all;
}

.text-muted {
  color: #94a3b8;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .license-page {
    padding: 12px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .title {
    font-size: 20px;
  }

  .subtitle {
    font-size: 13px;
  }

  .header-actions {
    width: 100%;
    flex-direction: column;
    gap: 8px;
  }

  .header-actions .el-select,
  .header-actions .el-input {
    width: 100% !important;
    margin-left: 0 !important;
  }

  .header-actions .el-button {
    width: 100%;
    margin-left: 0 !important;
  }

  .stat-row {
    margin-bottom: 12px;
  }

  .stat-card {
    padding: 14px;
    margin-bottom: 8px;
  }

  .stat-card .stat-value {
    font-size: 24px;
  }

  .stat-card .stat-label {
    font-size: 13px;
  }

  .list-card {
    margin-bottom: 12px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .card-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .pagination {
    text-align: center;
    justify-content: center;
  }

  /* 表格操作列优化 - 移动端使用更紧凑的布局 */
  .license-id-cell {
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cabinet-id {
    max-width: 100px;
  }

  .sync-status {
    max-width: 120px;
  }

  .sync-meta {
    font-size: 11px;
  }
}

@media (max-width: 480px) {
  .license-page {
    padding: 8px;
  }

  .title {
    font-size: 18px;
  }

  .stat-card {
    padding: 10px;
  }

  .stat-card .stat-value {
    font-size: 20px;
  }

  .stat-card .stat-label {
    font-size: 12px;
  }

  .card-title {
    font-size: 14px;
  }
}
</style>
