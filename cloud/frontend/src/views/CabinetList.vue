<template>
  <div class="cabinet-list">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>储能柜管理</h1>
      <el-button type="primary" @click="handlePreRegister" :icon="Plus">
        预注册储能柜
      </el-button>
    </div>

    <!-- 筛选区域 -->
    <el-card class="filter-card">
      <el-form :inline="true" :model="filterForm" @submit.prevent="handleFilter" class="filter-form">
        <el-form-item label="状态">
          <el-select v-model="filterForm.status" placeholder="请选择状态" clearable style="width: 140px;">
            <el-option label="待激活" value="pending" />
            <el-option label="在线同步中" value="active" />
            <el-option label="离线" value="offline" />
            <el-option label="已停用" value="inactive" />
            <el-option label="维护中" value="maintenance" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="位置">
          <el-input v-model="filterForm.location" placeholder="请输入位置" clearable style="width: 180px;" />
        </el-form-item>
        
        <el-form-item class="filter-buttons">
          <el-button type="primary" @click="handleFilter">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 数据表格 -->
    <el-card class="table-card">
      <el-table
        :data="cabinetStore.cabinets"
        :loading="cabinetStore.loading"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="cabinet_id" label="储能柜ID" width="150" />
        <el-table-column prop="name" label="名称" width="150" />
        <el-table-column prop="location" label="位置" width="180" />
        <el-table-column prop="capacity_kwh" label="容量(kWh)" width="110" />
        <el-table-column prop="mac_address" label="MAC地址" width="160" />
        <el-table-column label="运行状态" width="110">
          <template #default="{ row }">
            <StatusBadge :status="row.status" type="cabinet" />
          </template>
        </el-table-column>
        <el-table-column label="激活状态" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.activation_status === 'pending'" type="warning" size="small">
              待激活
            </el-tag>
            <el-tag v-else-if="row.activation_status === 'activated'" type="success" size="small">
              已激活
            </el-tag>
            <el-tag v-else type="info" size="small">
              未知
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="脆弱性评分" width="140">
          <template #default="{ row }">
            <div style="display: flex; flex-direction: column; align-items: center; gap: 4px;">
              <el-tag :type="getVulnerabilityTagType(row.latest_vulnerability_score)">
                {{ formatScore(row.latest_vulnerability_score) }} 分
              </el-tag>
              <el-tag v-if="row.latest_risk_level && row.latest_risk_level !== 'unknown'" :type="getRiskLevelTagType(row.latest_risk_level)" size="small">
                {{ formatRiskLevel(row.latest_risk_level) }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="最后同步" width="180">
          <template #default="{ row }">
            {{ row.last_sync_at ? formatTime(row.last_sync_at) : '从未同步' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" :width="isMobileView ? 80 : 200" class-name="action-column">
          <template #default="{ row }">
            <!-- 桌面端：显示按钮 -->
            <div v-if="!isMobileView" class="action-buttons">
              <el-button link type="primary" size="small" @click="handleView(row)">查看</el-button>
            <el-button
              v-if="row.activation_status === 'pending'"
              link
              type="warning"
                size="small"
              @click="handleShowActivation(row)"
            >
                激活
              </el-button>
              <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
            </div>
            <!-- 移动端：使用下拉菜单 -->
            <el-dropdown v-else trigger="click" @command="(cmd: string) => handleAction(cmd, row)">
              <el-button type="primary" size="small">
                操作
                <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="view">查看</el-dropdown-item>
                  <el-dropdown-item v-if="row.activation_status === 'pending'" command="activation">激活信息</el-dropdown-item>
                  <el-dropdown-item command="edit">编辑</el-dropdown-item>
                  <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="cabinetStore.currentPage"
          v-model:page-size="cabinetStore.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="cabinetStore.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handlePageSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="600px"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item label="储能柜ID" prop="cabinet_id">
          <el-input
            v-model="formData.cabinet_id"
            placeholder="请输入储能柜ID（如：CAB-001）"
            :disabled="isEdit"
          />
        </el-form-item>
        
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入储能柜名称" />
        </el-form-item>
        
        <el-form-item label="位置" prop="location">
          <el-input v-model="formData.location" placeholder="请输入位置" />
        </el-form-item>
        
        <el-form-item label="容量(kWh)" prop="capacity_kwh">
          <el-input-number v-model="formData.capacity_kwh" :min="0" :precision="2" />
        </el-form-item>
        
        <el-form-item label="MAC地址" prop="mac_address">
          <el-input
            v-model="formData.mac_address"
            placeholder="请输入MAC地址（如：00:1A:2B:3C:4D:5E）"
            :disabled="isEdit"
          />
        </el-form-item>
        
        <el-form-item v-if="isEdit" label="状态" prop="status">
          <el-select v-model="formData.status" placeholder="请选择状态">
            <el-option label="待激活" value="pending" />
            <el-option label="在线同步中" value="active" />
            <el-option label="离线" value="offline" />
            <el-option label="已停用" value="inactive" />
            <el-option label="维护中" value="maintenance" />
          </el-select>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="cabinetStore.loading" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 激活信息对话框 -->
    <el-dialog
      v-model="activationDialogVisible"
      title="储能柜激活信息"
      width="700px"
    >
      <div class="activation-info">
        <el-alert
          v-if="activationInfo.token_expired"
          title="注册Token已过期"
          type="error"
          :closable="false"
          show-icon
          style="margin-bottom: 20px;"
        >
          <template #default>
            <p>当前Token已过期，请重新生成Token以完成激活。</p>
            <el-button type="primary" size="small" @click="handleRegenerateToken" style="margin-top: 10px;">
              重新生成Token
            </el-button>
          </template>
        </el-alert>

        <el-alert
          v-else
          title="储能柜已预注册成功"
          type="success"
          :closable="false"
          show-icon
          style="margin-bottom: 20px;"
        >
          请按照以下步骤配置Edge端设备完成激活
        </el-alert>

        <!-- 储能柜基本信息 -->
        <el-descriptions :column="2" border style="margin-bottom: 20px;">
          <el-descriptions-item label="储能柜ID">
            {{ activationInfo.cabinet_id }}
          </el-descriptions-item>
          <el-descriptions-item label="储能柜名称">
            {{ activationInfo.name }}
          </el-descriptions-item>
          <el-descriptions-item label="MAC地址">
            {{ activationInfo.mac_address }}
          </el-descriptions-item>
          <el-descriptions-item label="Token有效期">
            <el-tag :type="activationInfo.token_expired ? 'danger' : 'success'" size="small">
              {{ formatTime(activationInfo.token_expires_at) }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 注册Token -->
        <div class="info-section">
          <h3>1. 注册Token</h3>
          <p class="hint">此Token用于Edge设备首次激活，有效期24小时，激活后自动失效</p>
          <div class="token-box">
            <el-input
              v-model="activationInfo.registration_token"
              readonly
              type="textarea"
              :rows="3"
            />
            <el-button
              type="primary"
              :icon="DocumentCopy"
              @click="copyToClipboard(activationInfo.registration_token, 'Token')"
              style="margin-top: 10px;"
            >
              复制Token
            </el-button>
          </div>
        </div>

        <!-- Edge配置 -->
        <div class="info-section">
          <h3>2. Edge端自动激活配置</h3>
          <p class="hint">将以下配置添加到Edge设备的 <code>Edge/configs/config.yaml</code> 文件中，Edge服务启动时会自动完成激活</p>
          <div class="config-box">
            <pre class="config-content">{{ edgeConfigContent }}</pre>
            <div class="config-actions">
              <el-button
                type="primary"
                :icon="DocumentCopy"
                @click="copyToClipboard(edgeConfigContent, '配置文件')"
              >
                复制配置
              </el-button>
              <el-button
                type="success"
                :icon="Download"
                @click="downloadConfig"
              >
                下载配置文件
              </el-button>
            </div>
          </div>
        </div>

        <!-- 激活步骤 -->
        <div class="info-section">
          <h3>3. 激活步骤</h3>
          <el-steps direction="vertical" :active="3">
            <el-step title="步骤1: 部署配置文件">
              <template #description>
                将上述配置复制并添加到Edge设备的 <code>Edge/configs/config.yaml</code> 文件中
              </template>
            </el-step>
            <el-step title="步骤2: 启动Edge服务">
              <template #description>
                执行命令: <code>./edge -config ./configs/config.yaml</code><br/>
                Edge服务启动时会检测到 registration.enabled=true，自动调用激活API
              </template>
            </el-step>
            <el-step title="步骤3: 验证激活成功">
              <template #description>
                查看Edge日志确认激活成功: <code>tail -f logs/edge.log | grep 激活</code><br/>
                在Cloud端"储能柜列表"页面查看状态是否从"待激活"变为"在线"
              </template>
            </el-step>
          </el-steps>
        </div>

        <!-- 注意事项 -->
        <el-alert
          title="注意事项"
          type="warning"
          :closable="false"
          style="margin-top: 20px;"
        >
          <ul style="margin: 0; padding-left: 20px;">
            <li>请确保配置文件中的 mac_address 与预注册时填写的MAC地址完全一致</li>
            <li>Token有效期为24小时，过期后需要重新生成</li>
            <li>配置中的 api_key 必须保持为空字符串 ""，激活后会自动填充</li>
            <li>激活成功后，registration.enabled 会自动设为 false，避免重复激活</li>
            <li>后续重启Edge服务将使用 api_key 进行认证，无需重新激活</li>
          </ul>
        </el-alert>
      </div>

      <template #footer>
        <el-button @click="activationDialogVisible = false">关闭</el-button>
        <el-button
          v-if="activationInfo.token_expired"
          type="primary"
          @click="handleRegenerateToken"
        >
          重新生成Token
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus';
import { Plus, DocumentCopy, Download, ArrowDown } from '@element-plus/icons-vue';
import { useCabinetStore } from '@/store/cabinet';
import { cabinetApi } from '@/api';
import StatusBadge from '@/components/StatusBadge.vue';
import type { Cabinet } from '@/types/api';

// 移动端检测
const MOBILE_BREAKPOINT = 768;
const isMobileView = ref(false);

function checkMobileView() {
  isMobileView.value = window.innerWidth < MOBILE_BREAKPOINT;
}

const router = useRouter();
const cabinetStore = useCabinetStore();

// 筛选表单
const filterForm = reactive({
  status: '',
  location: '',
});

// 激活信息对话框状态
const activationDialogVisible = ref(false);
const activationInfo = reactive({
  cabinet_id: '',
  name: '',
  mac_address: '',
  registration_token: '',
  token_expires_at: '',
  token_expired: false,
  cloud_api_url: '',
});

// Edge配置文件内容
const edgeConfigContent = computed(() => {
  return `cloud:
    enabled: true
    endpoint: ${activationInfo.cloud_api_url || window.location.origin + '/api/v1'}
    api_key: ""
    admin_token: ""
    cabinet_id: ${activationInfo.cabinet_id}
    timeout: 30s
    retry_count: 3
    retry_interval: 5s
registration:
    enabled: true
    token: "${activationInfo.registration_token}"
    mac_address: "${activationInfo.mac_address}"`;
});

// 对话框状态
const dialogVisible = ref(false);
const isEdit = ref(false);
const dialogTitle = ref('创建储能柜');

// 表单数据
const formRef = ref<FormInstance>();
const formData = reactive<Partial<Cabinet>>({
  cabinet_id: '',
  name: '',
  location: '',
  capacity_kwh: undefined,
  mac_address: '',
  status: 'offline',
});

// 表单验证规则
const formRules: FormRules = {
  cabinet_id: [
    { required: true, message: '请输入储能柜ID', trigger: 'blur' },
    { pattern: /^[A-Z0-9-]+$/, message: '储能柜ID只能包含大写字母、数字和连字符', trigger: 'blur' },
  ],
  name: [
    { required: true, message: '请输入储能柜名称', trigger: 'blur' },
  ],
  mac_address: [
    { required: true, message: '请输入MAC地址', trigger: 'blur' },
    { pattern: /^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/, message: 'MAC地址格式不正确', trigger: 'blur' },
  ],
};

// 初始化加载
onMounted(() => {
  checkMobileView();
  window.addEventListener('resize', checkMobileView);
  loadCabinets();
});

// 清理
onUnmounted(() => {
  window.removeEventListener('resize', checkMobileView);
});

// 移动端操作处理
function handleAction(command: string, row: Cabinet) {
  switch (command) {
    case 'view':
      handleView(row);
      break;
    case 'activation':
      handleShowActivation(row);
      break;
    case 'edit':
      handleEdit(row);
      break;
    case 'delete':
      handleDelete(row);
      break;
  }
}

// 加载储能柜列表
async function loadCabinets() {
  try {
    await cabinetStore.fetchCabinets({
      status: filterForm.status || undefined,
      location: filterForm.location || undefined,
      page: cabinetStore.currentPage,
      page_size: cabinetStore.pageSize,
    });
  } catch (error: any) {
    // 如果是401错误（未授权），不显示错误消息，因为已经跳转到登录页面
    if (error.code === 'UNAUTHORIZED' || error.status === 401) {
      console.log('未授权，即将跳转到登录页面');
      return;
    }
    // 其他错误才显示错误消息
    ElMessage.error(error.message || '加载储能柜列表失败');
  }
}

// 筛选
function handleFilter() {
  cabinetStore.currentPage = 1;
  loadCabinets();
}

// 重置
function handleReset() {
  filterForm.status = '';
  filterForm.location = '';
  handleFilter();
}

// 分页变化
function handlePageChange() {
  loadCabinets();
}

function handlePageSizeChange() {
  cabinetStore.currentPage = 1;
  loadCabinets();
}

// 查看详情
function handleView(row: Cabinet) {
  router.push(`/cabinets/${row.cabinet_id}`);
}

// 预注册储能柜
function handlePreRegister() {
  router.push('/cabinets/create');
}

// 显示激活信息
async function handleShowActivation(row: Cabinet) {
  try {
    const response = await cabinetApi.getActivationInfo(row.cabinet_id);
    Object.assign(activationInfo, response.data, {
      cloud_api_url: response.data.cloud_api_url || window.location.origin + '/api/v1',
    });
    activationDialogVisible.value = true;
  } catch (error: any) {
    ElMessage.error(error.message || '获取激活信息失败');
  }
}

// 重新生成Token
async function handleRegenerateToken() {
  try {
    await ElMessageBox.confirm(
      '重新生成Token后，之前的Token将立即失效。确定要继续吗？',
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    const response = await cabinetApi.regenerateToken(activationInfo.cabinet_id);
    activationInfo.registration_token = response.data.registration_token;
    activationInfo.token_expires_at = response.data.token_expires_at;
    activationInfo.token_expired = false;

    ElMessage.success('Token已重新生成');
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '重新生成Token失败');
    }
  }
}

// 复制到剪贴板
function copyToClipboard(text: string, label: string) {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success(`${label}已复制到剪贴板`);
  }).catch(() => {
    ElMessage.error('复制失败，请手动复制');
  });
}

// 下载配置文件
function downloadConfig() {
  const blob = new Blob([edgeConfigContent.value], { type: 'text/yaml' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `edge-config-${activationInfo.cabinet_id}.yaml`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
  ElMessage.success('配置文件已下载');
}

// 编辑
function handleEdit(row: Cabinet) {
  isEdit.value = true;
  dialogTitle.value = '编辑储能柜';
  Object.assign(formData, {
    cabinet_id: row.cabinet_id,
    name: row.name,
    location: row.location,
    capacity_kwh: row.capacity_kwh,
    mac_address: row.mac_address,
    status: row.status,
  });
  dialogVisible.value = true;
}

// 删除
async function handleDelete(row: Cabinet) {
  try {
    await ElMessageBox.confirm(
      `确定要删除储能柜 "${row.name}" 吗？`,
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );
    
    await cabinetStore.deleteCabinet(row.cabinet_id);
    ElMessage.success('删除成功');
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败');
    }
  }
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return;
    
    try {
      if (isEdit.value) {
        await cabinetStore.updateCabinet(formData.cabinet_id!, {
          name: formData.name,
          location: formData.location,
          capacity_kwh: formData.capacity_kwh,
          status: formData.status,
        });
        ElMessage.success('更新成功');
      } else {
        await cabinetStore.createCabinet(formData);
        ElMessage.success('创建成功');
      }
      
      dialogVisible.value = false;
    } catch (error: any) {
      ElMessage.error(error.message || '操作失败');
    }
  });
}

// 关闭对话框
function handleDialogClose() {
  resetForm();
}

// 重置表单
function resetForm() {
  formData.cabinet_id = '';
  formData.name = '';
  formData.location = '';
  formData.capacity_kwh = undefined;
  formData.mac_address = '';
  formData.status = 'offline';
  formRef.value?.clearValidate();
}

// 格式化脆弱性评分
function formatScore(score?: number) {
  if (typeof score === 'number' && Number.isFinite(score)) {
    return score.toFixed(1);
  }
  return '0.0';
}

// 获取脆弱性评分标签类型
function getVulnerabilityTagType(score?: number) {
  if (!score || score <= 0) return 'info';
  if (score >= 90) return 'success';
  if (score >= 75) return 'primary';
  if (score >= 60) return 'warning';
  return 'danger';
}

// 获取风险等级标签类型
function getRiskLevelTagType(level?: string) {
  if (!level || level === 'unknown') return 'info';
  if (level === 'healthy' || level === 'low') return 'success';
  if (level === 'medium') return 'warning';
  if (level === 'high') return 'danger';
  if (level === 'critical') return 'danger';
  return 'info';
}

// 格式化风险等级
function formatRiskLevel(level?: string) {
  const levelMap: Record<string, string> = {
    low: '低风险',
    medium: '中风险',
    high: '高风险',
    critical: '严重',
    healthy: '健康'
  };
  return levelMap[level || ''] || '';
}

// 格式化时间
function formatTime(time: string) {
  return new Date(time).toLocaleString('zh-CN');
}
</script>

<style scoped>
.cabinet-list {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: bold;
}

.filter-card {
  margin-bottom: 20px;
}

.table-card {
  margin-bottom: 20px;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

/* 激活信息对话框样式 */
.activation-info {
  font-size: 14px;
}

.info-section {
  margin-bottom: 24px;
}

.info-section h3 {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: #303133;
}

.info-section .hint {
  font-size: 13px;
  color: #909399;
  margin: 0 0 12px 0;
}

.token-box {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 4px;
}

.config-box {
  background: #f5f7fa;
  padding: 16px;
  border-radius: 4px;
}

.config-content {
  background: #ffffff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 12px;
  margin: 0 0 12px 0;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  max-height: 300px;
  overflow-y: auto;
}

.config-actions {
  display: flex;
  gap: 12px;
}

/* 筛选表单样式 */
.filter-form :deep(.el-form-item__label) {
  font-weight: 500;
}

/* 操作按钮区域 */
.action-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

/* 移动端响应式样式 */
@media (max-width: 768px) {
  .cabinet-list {
    padding: 12px;
  }

  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .page-header h1 {
    font-size: 20px;
  }

  .page-header .el-button {
    width: 100%;
  }

  .filter-card,
  .table-card {
    margin-bottom: 12px;
  }

  /* 筛选表单移动端优化 */
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

  .filter-buttons {
    width: 100%;
  }

  .filter-buttons :deep(.el-form-item__content) {
    display: flex;
    gap: 8px;
  }

  .filter-buttons :deep(.el-button) {
    flex: 1;
  }

  /* 表格在移动端可横向滚动 */
  .table-card :deep(.el-table) {
    font-size: 13px;
  }

  .table-card :deep(.el-table__body-wrapper) {
    overflow-x: auto;
  }

  /* 移动端取消固定列，以便横向滚动 */
  .table-card :deep(.el-table-fixed-column--right) {
    position: relative !important;
    right: auto !important;
    box-shadow: none !important;
  }

  /* 分页居中 */
  .pagination {
    justify-content: center;
  }

  /* 对话框优化 */
  .activation-info {
    font-size: 13px;
  }

  .info-section h3 {
    font-size: 15px;
  }

  .config-content {
    font-size: 12px;
    max-height: 200px;
  }

  .config-actions {
    flex-direction: column;
  }

  .config-actions .el-button {
    width: 100%;
  }
}

/* 超小屏幕 */
@media (max-width: 480px) {
  .cabinet-list {
    padding: 8px;
  }

  .page-header h1 {
    font-size: 18px;
  }

  .table-card :deep(.el-table) {
    font-size: 12px;
  }
}
</style>
