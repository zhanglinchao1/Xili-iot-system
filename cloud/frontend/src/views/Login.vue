<template>
  <div class="login-container">
    <el-card class="login-card">
      <template #header>
        <div class="card-header">
          <h2>Cloud 端储能柜管理系统</h2>
          <p>Cloud-side Energy Storage Cabinet Management System</p>
        </div>
      </template>

      <el-form
        ref="loginFormRef"
        :model="loginForm"
        :rules="loginRules"
        class="login-form"
        @submit.prevent="handleLogin"
      >
        <el-form-item prop="username">
          <el-input
            v-model="loginForm.username"
            placeholder="请输入用户名"
            prefix-icon="User"
            size="large"
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="请输入密码"
            prefix-icon="Lock"
            size="large"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="login-button"
            @click="handleLogin"
          >
            {{ loading ? '登录中...' : '登录' }}
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-footer">
        <el-text size="small" type="info">
          还没有账号?
          <el-link type="primary" :underline="'never'" @click="goToRegister">
            立即注册
          </el-link>
        </el-text>
        <el-divider direction="horizontal" style="margin: 12px 0" />
        <el-text size="small" type="info">
          默认测试账号: admin / admin
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
import { useAuthStore } from '@/store/auth'

const router = useRouter()
const authStore = useAuthStore()
const loginFormRef = ref<FormInstance>()
const loading = ref(false)

const loginForm = reactive({
  username: '',
  password: ''
})

const loginRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 64, message: '用户名长度在 3 到 64 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 4, max: 128, message: '密码长度在 4 到 128 个字符', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (!loginFormRef.value) return

  await loginFormRef.value.validate(async (valid) => {
    if (!valid) return

    loading.value = true
    try {
      await authStore.login(loginForm.username, loginForm.password)
      ElMessage.success('登录成功')
      router.push('/')
    } catch (error: any) {
      ElMessage.error(error.message || '登录失败')
    } finally {
      loading.value = false
    }
  })
}

const goToRegister = () => {
  router.push('/register')
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  position: relative;
  overflow: hidden;
}

/* 几何背景装饰 */
.login-container::before {
  content: '';
  position: absolute;
  width: 600px;
  height: 600px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  top: -300px;
  right: -300px;
  opacity: 0.15;
  filter: blur(60px);
}

.login-container::after {
  content: '';
  position: absolute;
  width: 500px;
  height: 500px;
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
  border-radius: 50%;
  bottom: -250px;
  left: -250px;
  opacity: 0.15;
  filter: blur(60px);
}

.login-card {
  width: 450px;
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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

.login-form {
  padding: 24px 0;
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
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
  border-color: #667eea;
}

.login-button {
  width: 100%;
  height: 48px;
  border-radius: 10px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.3px;
  transition: all 0.3s;
  box-shadow: 0 4px 14px rgba(102, 126, 234, 0.4);
}

.login-button:hover {
  background: linear-gradient(135deg, #5568d3 0%, #6a3f8f 100%);
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(102, 126, 234, 0.5);
}

.login-button:active {
  transform: translateY(0);
}

.login-footer {
  text-align: center;
  padding-top: 20px;
  border-top: 1px solid #e2e8f0;
}

:deep(.el-divider--horizontal) {
  border-color: #e2e8f0;
  margin: 16px 0;
}

:deep(.el-link.el-link--primary) {
  color: #667eea;
  font-weight: 600;
}

:deep(.el-link.el-link--primary:hover) {
  color: #5568d3;
}

:deep(.el-text) {
  font-size: 15px;
  color: #64748b;
}

/* 移动端响应式 */
@media (max-width: 768px) {
  .login-container {
    padding: 16px;
  }

  .login-card {
    width: 100%;
    max-width: 400px;
    border-radius: 16px;
  }

  .card-header h2 {
    font-size: 22px;
  }

  .card-header p {
    font-size: 12px;
  }

  .login-form {
    padding: 16px 0;
  }

  .login-button {
    height: 44px;
    font-size: 15px;
  }
}

@media (max-width: 480px) {
  .login-container {
    padding: 12px;
  }

  .login-card {
    border-radius: 12px;
  }

  :deep(.el-card__header),
  :deep(.el-card__body) {
    padding: 16px !important;
  }

  .card-header h2 {
    font-size: 20px;
  }

  .card-header p {
    font-size: 11px;
  }

  .login-button {
    height: 42px;
  }
}
</style>
