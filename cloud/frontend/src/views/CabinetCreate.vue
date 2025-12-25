<template>
  <div class="cabinet-create-page">
    <!-- 面包屑导航 -->
    <el-breadcrumb separator="/" class="breadcrumb">
      <el-breadcrumb-item :to="{ path: '/cabinets' }">储能柜管理</el-breadcrumb-item>
      <el-breadcrumb-item>预注册储能柜</el-breadcrumb-item>
    </el-breadcrumb>

    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="title">预注册储能柜</h1>
        <p class="subtitle">Pre-register Energy Storage Cabinet</p>
      </div>
    </div>

    <!-- 表单卡片 -->
    <el-card class="form-card">
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="140px"
        size="large"
      >
        <!-- 基本信息 -->
        <div class="form-section">
          <h3 class="section-title">基本信息</h3>

          <el-form-item label="Cabinet ID" prop="cabinet_id">
            <el-input
              v-model="form.cabinet_id"
              placeholder="CAB-001"
              style="width: 300px"
            >
              <template #append>
                <el-button @click="generateCabinetId">
                  <el-icon><Refresh /></el-icon>
                  自动生成
                </el-button>
              </template>
            </el-input>
            <div class="field-hint">
              储能柜的唯一标识符,建议格式: CAB-XXX (自动生成或手动输入)
            </div>
          </el-form-item>

          <el-form-item label="储能柜名称" prop="name">
            <el-input
              v-model="form.name"
              placeholder="朝阳区机房A-1号柜"
              style="width: 400px"
            />
          </el-form-item>

          <el-form-item label="安装位置" prop="location">
            <LocationPicker
              v-model="form.location"
              v-model:latitude="form.latitude"
              v-model:longitude="form.longitude"
            />
          </el-form-item>

          <el-form-item label="设备容量" prop="capacity_kwh">
            <el-input-number
              v-model="form.capacity_kwh"
              :min="1"
              :max="10000"
              :step="10"
              style="width: 200px"
            />
            <span class="unit">kWh</span>
          </el-form-item>

          <el-form-item label="客户信息" prop="customer_name">
            <el-input
              v-model="form.customer_name"
              placeholder="XX能源公司"
              style="width: 400px"
            />
          </el-form-item>
        </div>

        <!-- 安全信息 -->
        <div class="form-section">
          <h3 class="section-title">
            <el-icon><Lock /></el-icon>
            安全信息
          </h3>

          <el-form-item label="MAC地址" prop="mac_address">
            <el-input
              v-model="form.mac_address"
              placeholder="00:1A:2B:3C:4D:5E"
              style="width: 300px"
              @blur="formatMacAddress"
            >
              <template #append>
                <el-button @click="validateMacFormat">
                  <el-icon><CircleCheck /></el-icon>
                  验证格式
                </el-button>
              </template>
            </el-input>
            <div class="field-hint">
              <el-icon><Warning /></el-icon>
              Edge设备的网卡MAC地址,用于绑定和身份验证,必须准确填写
            </div>
          </el-form-item>

          <el-form-item label="许可证有效期" prop="license_expires_in_days">
            <el-select
              v-model="form.license_expires_in_days"
              placeholder="选择有效期"
              style="width: 300px"
            >
              <el-option label="90天 (试用期)" :value="90" />
              <el-option label="1年 (标准)" :value="365" />
              <el-option label="3年 (推荐)" :value="1095" />
              <el-option label="5年 (长期)" :value="1825" />
            </el-select>
            <div class="field-hint">
              许可证到期后需要续期,可在许可控制页面管理
            </div>
          </el-form-item>

          <el-form-item label="许可权限" prop="permissions">
            <el-checkbox-group v-model="form.permissions">
              <el-checkbox label="data_sync">数据同步</el-checkbox>
              <el-checkbox label="remote_config">远程配置</el-checkbox>
              <el-checkbox label="remote_control">远程控制</el-checkbox>
              <el-checkbox label="firmware_update">固件升级</el-checkbox>
            </el-checkbox-group>
          </el-form-item>
        </div>

        <!-- 可选配置 -->
        <div class="form-section">
          <h3 class="section-title">可选配置</h3>

          <el-form-item label="预期IP地址">
            <el-input
              v-model="form.expected_ip"
              placeholder="192.168.1.100 (可选)"
              style="width: 300px"
            />
            <div class="field-hint">
              Edge设备的预期IP地址,仅用于记录,不影响连接
            </div>
          </el-form-item>

          <el-form-item label="设备型号">
            <el-input
              v-model="form.device_model"
              placeholder="ESC-1000 (可选)"
              style="width: 300px"
            />
          </el-form-item>

          <el-form-item label="备注信息">
            <el-input
              v-model="form.notes"
              type="textarea"
              :rows="4"
              placeholder="可填写设备相关备注信息"
              style="width: 500px"
            />
          </el-form-item>
        </div>

        <!-- 提交按钮 -->
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="submitting"
            @click="handlePreRegister"
          >
            <el-icon><Check /></el-icon>
            预注册储能柜
          </el-button>
          <el-button size="large" @click="handleReset">
            <el-icon><Refresh /></el-icon>
            重置表单
          </el-button>
          <el-button size="large" @click="router.back()">
            <el-icon><Close /></el-icon>
            取消
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 注册成功对话框 -->
    <el-dialog
      v-model="showTokenDialog"
      title="预注册成功"
      width="700px"
      :close-on-click-modal="false"
    >
      <el-alert type="success" :closable="false" style="margin-bottom: 20px">
        <p style="font-size: 16px; margin: 0">
          储能柜 <strong>{{ registeredCabinet.cabinet_id }}</strong> 已成功预注册!
        </p>
      </el-alert>

      <div class="token-section">
        <h4 class="dialog-subtitle">
          <el-icon><Key /></el-icon>
          Registration Token (24小时内有效)
        </h4>
        <el-input
          v-model="registeredCabinet.registration_token"
          type="textarea"
          :rows="4"
          readonly
          class="token-input"
        />
        <el-button
          type="primary"
          :icon="DocumentCopy"
          @click="copyToken"
          style="margin-top: 8px"
        >
          复制Token
        </el-button>

        <el-divider />

        <h4 class="dialog-subtitle">
          <el-icon><Setting /></el-icon>
          Edge端自动激活配置
        </h4>
        <p class="config-instructions">
          将以下配置添加到Edge设备的 <code>Edge/configs/config.yaml</code> 文件中，Edge服务启动时会自动完成激活：
        </p>
        <div class="config-example">
          <pre>{{ edgeConfigContent }}</pre>
          <el-button
            :icon="DocumentCopy"
            size="small"
            @click="copyConfig"
            class="copy-config-btn"
          >
            复制配置
          </el-button>
        </div>

        <el-button
          type="success"
          :icon="Download"
          @click="downloadConfig"
          style="margin-top: 12px"
        >
          下载完整配置文件
        </el-button>

        <el-divider />

        <h4 class="dialog-subtitle">
          <el-icon><InfoFilled /></el-icon>
          后续步骤
        </h4>
        <el-steps direction="vertical" :active="0">
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
          <el-step title="步骤4: 后续使用">
            <template #description>
              激活成功后，registration.enabled 会自动设为 false，api_key 会保存到配置文件<br/>
              后续重启Edge服务将使用 api_key 进行认证，无需重新激活
            </template>
          </el-step>
        </el-steps>
      </div>

      <template #footer>
        <el-button type="primary" @click="goToCabinetList">
          前往储能柜列表
        </el-button>
        <el-button @click="showTokenDialog = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import {
  Refresh, Lock, CircleCheck, Warning, Check, Close,
  Key, Setting, InfoFilled, DocumentCopy, Download
} from '@element-plus/icons-vue'
import { cabinetApi } from '@/api'
import type { PreRegisterResponse } from '@/types/api'
import LocationPicker from '@/components/LocationPicker.vue'

const router = useRouter()
const formRef = ref<FormInstance>()
const submitting = ref(false)
const showTokenDialog = ref(false)

// 表单数据
const form = reactive({
  cabinet_id: '',
  name: '',
  location: '',
  latitude: 0,
  longitude: 0,
  capacity_kwh: 100,
  customer_name: '',
  mac_address: '',
  license_expires_in_days: 365,
  permissions: ['data_sync', 'remote_config'],
  expected_ip: '',
  device_model: '',
  notes: ''
})

// 注册成功后的数据
const registeredCabinet = ref<PreRegisterResponse>({
  cabinet_id: '',
  registration_token: '',
  token_expires_at: ''
})

// 表单验证规则
const validateMac = (_rule: any, value: any, callback: any) => {
  if (!value) {
    callback(new Error('请输入MAC地址'))
  } else {
    const macRegex = /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$/
    if (!macRegex.test(value)) {
      callback(new Error('MAC地址格式不正确,正确格式: 00:1A:2B:3C:4D:5E'))
    } else {
      callback()
    }
  }
}

const rules: FormRules = {
  cabinet_id: [
    { required: true, message: '请输入Cabinet ID', trigger: 'blur' },
    { min: 3, max: 64, message: 'Cabinet ID长度在3到64个字符', trigger: 'blur' },
    { pattern: /^[A-Z0-9-]+$/, message: 'Cabinet ID只能包含大写字母、数字和连字符', trigger: 'blur' }
  ],
  name: [
    { required: true, message: '请输入储能柜名称', trigger: 'blur' },
    { max: 255, message: '名称长度不能超过255个字符', trigger: 'blur' }
  ],
  location: [
    { required: true, message: '请输入安装位置', trigger: 'blur' }
  ],
  capacity_kwh: [
    { required: true, message: '请输入设备容量', trigger: 'blur' }
  ],
  customer_name: [
    { required: true, message: '请输入客户信息', trigger: 'blur' }
  ],
  mac_address: [
    { required: true, validator: validateMac, trigger: 'blur' }
  ],
  license_expires_in_days: [
    { required: true, message: '请选择许可证有效期', trigger: 'change' }
  ],
  permissions: [
    { type: 'array', required: true, message: '请至少选择一个权限', trigger: 'change' }
  ]
}

// 生成Cabinet ID
const generateCabinetId = () => {
  const timestamp = Date.now().toString(36).toUpperCase()
  const random = Math.random().toString(36).substring(2, 5).toUpperCase()
  form.cabinet_id = `CAB-${timestamp}${random}`
  ElMessage.success(`已生成Cabinet ID: ${form.cabinet_id}`)
}

// 格式化MAC地址
const formatMacAddress = () => {
  if (form.mac_address) {
    // 移除所有非十六进制字符
    let mac = form.mac_address.replace(/[^0-9A-Fa-f]/g, '')
    // 每两个字符插入冒号
    if (mac.length === 12) {
      form.mac_address = mac.match(/.{1,2}/g)?.join(':').toUpperCase() || mac
    }
  }
}

// 验证MAC地址格式
const validateMacFormat = () => {
  const macRegex = /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$/
  if (macRegex.test(form.mac_address)) {
    ElMessage.success('MAC地址格式正确')
  } else {
    ElMessage.error('MAC地址格式不正确,正确格式: 00:1A:2B:3C:4D:5E')
  }
}

// Edge配置文件内容
const edgeConfigContent = computed(() => {
  return `cloud:
    enabled: true
    endpoint: ${window.location.origin}/api/v1
    api_key: ""
    admin_token: ""
    cabinet_id: ${registeredCabinet.value.cabinet_id}
    timeout: 30s
    retry_count: 3
    retry_interval: 5s
registration:
    enabled: true
    token: "${registeredCabinet.value.registration_token}"
    mac_address: "${form.mac_address}"`
})

// 预注册储能柜
const calculateLicenseExpiresAt = () => {
  const expireDate = new Date()
  expireDate.setDate(expireDate.getDate() + form.license_expires_in_days)
  return expireDate.toISOString()
}

const buildNotes = () => {
  if (form.customer_name) {
    return `客户: ${form.customer_name}${form.notes ? `\n${form.notes}` : ''}`
  }
  return form.notes || undefined
}

const handlePreRegister = async () => {
  if (!formRef.value) return

  const isValid = await formRef.value.validate().catch(() => false)
  if (!isValid) {
    ElMessage.error('请检查表单填写是否正确')
    return
  }

  submitting.value = true
  try {
    const response = await cabinetApi.preRegister({
      cabinet_id: form.cabinet_id,
      name: form.name,
      location: form.location || undefined,
      capacity_kwh: form.capacity_kwh || undefined,
      mac_address: form.mac_address,
      license_expires_at: calculateLicenseExpiresAt(),
      permissions: form.permissions,
      ip_address: form.expected_ip || undefined,
      device_model: form.device_model || undefined,
      notes: buildNotes(),
    })

    registeredCabinet.value = response.data
    showTokenDialog.value = true
    ElMessage.success('储能柜预注册成功!')
  } catch (error: any) {
    if (error.code === 'UNAUTHORIZED' || error.status === 401) {
      console.log('未授权，即将跳转到登录页面')
      return
    }
    ElMessage.error(error.message || '预注册失败,请稍后重试')
  } finally {
    submitting.value = false
  }
}

// 重置表单
const handleReset = () => {
  ElMessageBox.confirm('确定要重置表单吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    formRef.value?.resetFields()
    ElMessage.info('表单已重置')
  }).catch(() => {
    // 用户取消
  })
}

// 复制Token
const copyToken = () => {
  navigator.clipboard.writeText(registeredCabinet.value.registration_token)
  ElMessage.success('Token已复制到剪贴板')
}

// 复制配置
const copyConfig = () => {
  navigator.clipboard.writeText(edgeConfigContent.value)
  ElMessage.success('配置已复制到剪贴板')
}

// 下载配置文件
const downloadConfig = () => {
  const blob = new Blob([edgeConfigContent.value], { type: 'text/yaml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `edge-config-${registeredCabinet.value.cabinet_id}.yaml`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('配置文件已下载')
}

// 前往储能柜列表
const goToCabinetList = () => {
  showTokenDialog.value = false
  router.push('/cabinets')
}
</script>

<style scoped>
.cabinet-create-page {
  padding: 24px;
  background: #f8fafc;
  min-height: 100vh;
}

/* 面包屑 */
.breadcrumb {
  margin-bottom: 16px;
  font-size: 15px;
}

/* 页面头部 */
.page-header {
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

/* 表单卡片 */
.form-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

:deep(.el-card__body) {
  padding: 32px;
}

/* 表单分组 */
.form-section {
  margin-bottom: 32px;
  padding-bottom: 32px;
  border-bottom: 1px solid #e2e8f0;
}

.form-section:last-of-type {
  border-bottom: none;
}

.section-title {
  margin: 0 0 24px 0;
  font-size: 18px;
  font-weight: 600;
  color: #0f172a;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 表单提示 */
.field-hint {
  margin-top: 4px;
  font-size: 13px;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 4px;
}

.field-hint .el-icon {
  font-size: 14px;
  color: #f59e0b;
}

.unit {
  margin-left: 8px;
  font-size: 15px;
  color: #64748b;
  font-weight: 500;
}

/* Token对话框 */
.token-section {
  padding: 20px;
  background: #f8fafc;
  border-radius: 8px;
}

.dialog-subtitle {
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
  display: flex;
  align-items: center;
  gap: 8px;
}

.token-input {
  font-family: 'Courier New', monospace;
  font-size: 13px;
}

.config-instructions {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #475569;
}

.config-instructions code {
  padding: 2px 6px;
  background: #e2e8f0;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 13px;
  color: #0f172a;
}

.config-example {
  position: relative;
  margin: 0;
  padding: 16px;
  background: #1e293b;
  border-radius: 8px;
  overflow-x: auto;
}

.config-example pre {
  margin: 0;
  font-family: 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #e2e8f0;
  white-space: pre;
}

.copy-config-btn {
  position: absolute;
  top: 12px;
  right: 12px;
}

/* 步骤条样式 */
:deep(.el-steps) {
  margin-top: 16px;
}

:deep(.el-step__description) {
  font-size: 14px;
  color: #64748b;
}

/* 响应式 */
@media (max-width: 768px) {
  .cabinet-create-page {
    padding: 16px;
  }

  .header-content .title {
    font-size: 22px;
  }

  :deep(.el-form-item__label) {
    width: 100px !important;
  }

  .el-input,
  .el-select,
  .el-input-number {
    width: 100% !important;
  }
}
</style>
