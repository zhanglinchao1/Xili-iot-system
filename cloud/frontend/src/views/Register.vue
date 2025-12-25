<template>
  <div class="register-container">
    <el-card class="register-card">
      <template #header>
        <div class="card-header">
          <h2>创建账号</h2>
          <p>Create your Cloud Account</p>
        </div>
      </template>

      <el-form
        ref="registerFormRef"
        :model="registerForm"
        :rules="registerRules"
        class="register-form"
        @submit.prevent="handleRegister"
      >
        <el-form-item prop="username">
          <el-input
            v-model="registerForm.username"
            placeholder="用户名 (3-64字符)"
            prefix-icon="User"
            size="large"
          />
        </el-form-item>

        <el-form-item prop="email">
          <el-input
            v-model="registerForm.email"
            placeholder="邮箱地址"
            prefix-icon="Message"
            size="large"
          />
        </el-form-item>

        <el-form-item prop="password" class="password-item">
          <el-input
            v-model="registerForm.password"
            type="password"
            placeholder="密码 (至少8位)"
            prefix-icon="Lock"
            size="large"
            show-password
          />
          <p class="hint">密码需至少8位，并同时包含大写字母、小写字母与数字。</p>
        </el-form-item>

        <el-form-item prop="confirmPassword">
          <el-input
            v-model="registerForm.confirmPassword"
            type="password"
            placeholder="确认密码"
            prefix-icon="Lock"
            size="large"
            show-password
            @keyup.enter="handleRegister"
          />
        </el-form-item>

        <el-form-item prop="agree">
          <el-checkbox v-model="registerForm.agree" size="large">
            我已阅读并同意
            <el-link type="primary" :underline="'hover'">《用户协议》</el-link>
            和
            <el-link type="primary" :underline="'hover'">《隐私政策》</el-link>
          </el-checkbox>
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="register-button"
            @click="handleRegister"
          >
            {{ loading ? '注册中...' : '立即注册' }}
          </el-button>
        </el-form-item>
      </el-form>

      <div class="register-footer">
        <el-text size="small" type="info">
          已有账号?
          <el-link type="primary" :underline="'never'" @click="goToLogin">
            立即登录
          </el-link>
        </el-text>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { authApi } from '@/api'

const router = useRouter()
const registerFormRef = ref<FormInstance>()
const loading = ref(false)

const registerForm = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  agree: false
})

// 密码确认验证
const validateConfirmPassword = (_rule: any, value: any, callback: any) => {
  if (value === '') {
    callback(new Error('请再次输入密码'))
  } else if (value !== registerForm.password) {
    callback(new Error('两次输入密码不一致'))
  } else {
    callback()
  }
}

// 协议勾选验证
const validateAgree = (_rule: any, value: any, callback: any) => {
  if (!value) {
    callback(new Error('请阅读并同意用户协议和隐私政策'))
  } else {
    callback()
  }
}

const registerRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 64, message: '用户名长度在 3 到 64 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_-]+$/, message: '用户名只能包含字母、数字、下划线和连字符', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, max: 128, message: '密码长度至少8个字符', trigger: 'blur' },
    { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/, message: '密码必须包含大小写字母和数字', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, validator: validateConfirmPassword, trigger: 'blur' }
  ],
  agree: [
    { required: true, validator: validateAgree, trigger: 'change' }
  ]
}

const handleRegister = async () => {
  if (!registerFormRef.value) return

  await registerFormRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      const response = await authApi.register({
        username: registerForm.username,
        email: registerForm.email,
        password: registerForm.password
      })

      if (response.success) {
        ElMessage.success('注册成功! 请登录')
        setTimeout(() => {
          router.push('/login')
        }, 1500)
      } else {
        ElMessage.error(response.message || '注册失败')
      }
    } catch (error: any) {
      ElMessage.error(error.message || '注册失败，请稍后重试')
    } finally {
      loading.value = false
    }
  })
}

const goToLogin = () => {
  router.push('/login')
}
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  position: relative;
  overflow: hidden;
  padding: 40px 20px;
}

/* 几何背景装饰 */
.register-container::before {
  content: '';
  position: absolute;
  width: 600px;
  height: 600px;
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
  border-radius: 50%;
  top: -300px;
  left: -300px;
  opacity: 0.15;
  filter: blur(60px);
}

.register-container::after {
  content: '';
  position: absolute;
  width: 500px;
  height: 500px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  bottom: -250px;
  right: -250px;
  opacity: 0.15;
  filter: blur(60px);
}

.register-card {
  width: 480px;
  border-radius: 20px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  position: relative;
  z-index: 10;
}

:deep(.el-card__header) {
  background: transparent;
  border-bottom: 1px solid #e2e8f0;
}

.card-header {
  text-align: center;
  padding: 8px 0;
}

.card-header h2 {
  margin: 0 0 8px 0;
  font-size: 28px;
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  font-weight: 700;
  letter-spacing: -0.5px;
}

.card-header p {
  margin: 0;
  font-size: 14px;
  color: #64748b;
  letter-spacing: 0.3px;
  font-weight: 500;
}

.register-form {
  padding: 24px 0;
}

.password-item .hint {
  margin-top: 6px;
  font-size: 12px;
  color: #64748b;
}

:deep(.el-input__wrapper) {
  border-radius: 10px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s;
  border: 1px solid #e2e8f0;
  background: white;
}

:deep(.el-input__wrapper:hover) {
  border-color: #94a3b8;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 3px rgba(6, 182, 212, 0.1);
  border-color: #06b6d4;
}

.register-button {
  width: 100%;
  height: 48px;
  border-radius: 10px;
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
  border: none;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.3px;
  transition: all 0.3s;
  box-shadow: 0 4px 14px rgba(6, 182, 212, 0.4);
}

.register-button:hover {
  background: linear-gradient(135deg, #0891b2 0%, #0e7490 100%);
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(6, 182, 212, 0.5);
}

.register-button:active {
  transform: translateY(0);
}

:deep(.el-checkbox__label) {
  font-size: 14px;
  color: #475569;
}

:deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #06b6d4;
  border-color: #06b6d4;
}

:deep(.el-link.el-link--primary) {
  color: #06b6d4;
  font-weight: 600;
}

:deep(.el-link.el-link--primary:hover) {
  color: #0891b2;
}

.register-footer {
  text-align: center;
  padding-top: 20px;
  border-top: 1px solid #e2e8f0;
}

:deep(.el-text) {
  font-size: 15px;
  color: #64748b;
}
</style>
