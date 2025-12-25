/**
 * 配置Store
 * 管理前端配置的加载和缓存
 */

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { configApi } from '@/api';
import type { FrontendConfig } from '@/types/api';

export const useConfigStore = defineStore('config', () => {
  // 状态
  const config = ref<FrontendConfig | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // 计算属性
  // 优先使用配置中的地址，如果没有则使用相对路径（通过nginx代理）
  const apiBaseUrl = computed(() => {
    const configUrl = config.value?.api_base_url;
    if (configUrl) {
      // 如果配置的URL是绝对路径且包含localhost，转换为相对路径
      // 这样可以通过nginx代理正确访问后端
      if (configUrl.includes('localhost') || configUrl.includes('127.0.0.1')) {
        return '/api/v1';
      }
      return configUrl;
    }
    return '/api/v1';
  });
  const pollingInterval = computed(() => config.value?.polling_interval || 5000);
  const chartRefreshInterval = computed(() => config.value?.chart_refresh_interval || 30000);
  const pageSize = computed(() => config.value?.page_size || 20);
  const maxPageSize = computed(() => config.value?.max_page_size || 100);

  // 加载配置
  async function loadConfig() {
    if (loading.value) return;
    
    loading.value = true;
    error.value = null;
    
    try {
      const response = await configApi.getConfig();
      config.value = response.data;
      
      // 缓存到localStorage
      localStorage.setItem('frontend_config', JSON.stringify(response.data));
    } catch (err: any) {
      error.value = err.message || '加载配置失败';
      console.error('Failed to load config:', err);
      
      // 尝试从localStorage加载缓存
      const cached = localStorage.getItem('frontend_config');
      if (cached) {
        try {
          config.value = JSON.parse(cached);
        } catch (e) {
          console.error('Failed to parse cached config:', e);
        }
      }
    } finally {
      loading.value = false;
    }
  }

  // 重新加载配置
  async function reloadConfig() {
    config.value = null;
    await loadConfig();
  }

  // 清除配置
  function clearConfig() {
    config.value = null;
    localStorage.removeItem('frontend_config');
  }

  return {
    // 状态
    config,
    loading,
    error,
    
    // 计算属性
    apiBaseUrl,
    pollingInterval,
    chartRefreshInterval,
    pageSize,
    maxPageSize,
    
    // 方法
    loadConfig,
    reloadConfig,
    clearConfig,
  };
});

