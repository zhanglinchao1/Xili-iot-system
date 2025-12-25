/**
 * 认证状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '@/api'
import type { UserInfo } from '@/types/api'

export const useAuthStore = defineStore('auth', () => {
  // State
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<UserInfo | null>(null)

  // Getters
  const isAuthenticated = computed(() => !!token.value)
  const userRole = computed(() => user.value?.role || 'guest')
  const isAdmin = computed(() => user.value?.role === 'admin')

  // Actions
  async function login(username: string, password: string) {
    try {
      const response = await api.authApi.login({ username, password })

      if (response.success && response.data) {
        // 保存 Token
        token.value = response.data.token
        localStorage.setItem('token', response.data.token)
        
        // 保存用户信息
        user.value = response.data.user
        localStorage.setItem('user', JSON.stringify(response.data.user))

        return response.data
      } else {
        throw new Error(response.message || '登录失败')
      }
    } catch (error: any) {
      console.error('Login failed:', error)
      throw new Error(error.message || '登录失败')
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  // 从 localStorage 恢复用户信息
  function restoreUser() {
    const storedUser = localStorage.getItem('user')
    if (storedUser) {
      try {
        user.value = JSON.parse(storedUser)
      } catch (error) {
        console.error('Failed to parse stored user:', error)
        localStorage.removeItem('user')
      }
    }
  }

  // 设置用户信息
  function setUser(userInfo: UserInfo) {
    user.value = userInfo
    localStorage.setItem('user', JSON.stringify(userInfo))
  }

  // 初始化时恢复用户信息
  restoreUser()

  return {
    // State
    token,
    user,
    
    // Getters
    isAuthenticated,
    userRole,
    isAdmin,
    
    // Actions
    login,
    logout,
    restoreUser,
    setUser
  }
})
