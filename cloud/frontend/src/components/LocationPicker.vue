<template>
  <div class="location-picker">
    <el-input
      v-model="searchKeyword"
      placeholder="搜索地点(如: 北京市朝阳区XX路)"
      :prefix-icon="Search"
      clearable
      @input="handleSearch"
      @clear="clearSearch"
      style="width: 500px"
    >
      <template #append>
        <el-button :icon="Location" @click="getCurrentLocation">
          当前位置
        </el-button>
      </template>
    </el-input>

    <!-- 搜索结果下拉列表 -->
    <div v-if="searchResults.length > 0" class="search-results">
      <div
        v-for="result in searchResults"
        :key="result.id"
        class="result-item"
        @click="selectLocation(result)"
      >
        <div class="result-title">{{ result.title }}</div>
        <div class="result-address">{{ result.address }}</div>
      </div>
    </div>

    <!-- 已选位置显示 -->
    <div v-if="selectedLocation" class="selected-location">
      <el-tag type="success" closable @close="clearLocation" size="large">
        <el-icon><MapLocation /></el-icon>
        {{ selectedLocation.title }} - {{ selectedLocation.address }}
      </el-tag>
      <div class="coordinates">
        经度: {{ selectedLocation.longitude }}, 纬度: {{ selectedLocation.latitude }}
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-hint">
      <el-icon class="is-loading"><Loading /></el-icon>
      搜索中...
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="error-hint">
      <el-icon><Warning /></el-icon>
      {{ error }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Location, MapLocation, Loading, Warning } from '@element-plus/icons-vue'
import api from '@/api'

interface LocationResult {
  id: string
  title: string
  address: string
  latitude: number
  longitude: number
}

interface Props {
  modelValue?: string // 地址文本
  latitude?: number
  longitude?: number
}

interface Emits {
  (e: 'update:modelValue', value: string): void
  (e: 'update:latitude', value: number): void
  (e: 'update:longitude', value: number): void
  (e: 'change', location: { address: string; latitude: number; longitude: number }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const searchKeyword = ref(props.modelValue || '')
const searchResults = ref<LocationResult[]>([])
const selectedLocation = ref<LocationResult | null>(null)
const loading = ref(false)
const error = ref('')
let searchTimer: NodeJS.Timeout | null = null

// 如果有初始坐标,显示已选位置
watch(() => [props.latitude, props.longitude], ([lat, lng]) => {
  if (lat && lng && props.modelValue) {
    selectedLocation.value = {
      id: 'initial',
      title: props.modelValue,
      address: props.modelValue,
      latitude: lat,
      longitude: lng
    }
  }
}, { immediate: true })

// 搜索地点
const handleSearch = () => {
  if (!searchKeyword.value || searchKeyword.value.trim() === '') {
    searchResults.value = []
    return
  }

  // 防抖
  if (searchTimer) {
    clearTimeout(searchTimer)
  }

  searchTimer = setTimeout(async () => {
    await performSearch()
  }, 500)
}

// 执行搜索
const performSearch = async () => {
  const keyword = searchKeyword.value.trim()
  if (!keyword) return

  loading.value = true
  error.value = ''

  try {
    // 调用后端代理接口搜索地点
    const res = await api.map.searchLocation(keyword)
    if (res.data && res.data.results) {
      searchResults.value = res.data.results.map((item: any) => ({
        id: item.id || item.uid,
        title: item.title || item.name,
        address: item.address || item.location,
        latitude: item.location?.lat || item.latitude,
        longitude: item.location?.lng || item.longitude
      }))
    } else {
      searchResults.value = []
      ElMessage.warning('未找到相关地点')
    }
  } catch (err: any) {
    console.error('搜索地点失败:', err)
    error.value = err.message || '搜索失败,请重试'
    searchResults.value = []
  } finally {
    loading.value = false
  }
}

// 选择地点
const selectLocation = (location: LocationResult) => {
  selectedLocation.value = location
  searchKeyword.value = location.title
  searchResults.value = []

  // 触发更新
  emit('update:modelValue', location.address)
  emit('update:latitude', location.latitude)
  emit('update:longitude', location.longitude)
  emit('change', {
    address: location.address,
    latitude: location.latitude,
    longitude: location.longitude
  })

  ElMessage.success('位置已选择')
}

// 清除搜索结果
const clearSearch = () => {
  searchResults.value = []
  error.value = ''
}

// 清除已选位置
const clearLocation = () => {
  selectedLocation.value = null
  searchKeyword.value = ''
  emit('update:modelValue', '')
  emit('update:latitude', 0)
  emit('update:longitude', 0)
}

// 获取当前位置 (浏览器定位)
const getCurrentLocation = () => {
  if (!navigator.geolocation) {
    ElMessage.error('浏览器不支持定位功能')
    return
  }

  loading.value = true
  navigator.geolocation.getCurrentPosition(
    async (position) => {
      const { latitude, longitude } = position.coords

      try {
        // 逆地理编码:通过坐标获取地址
        const res = await api.map.reverseGeocode(latitude, longitude)
        if (res.data && res.data.address) {
          const location: LocationResult = {
            id: 'current',
            title: res.data.title || '当前位置',
            address: res.data.address,
            latitude,
            longitude
          }
          selectLocation(location)
        } else {
          ElMessage.warning('无法获取地址信息')
        }
      } catch (err: any) {
        console.error('逆地理编码失败:', err)
        ElMessage.error('获取地址信息失败')
      } finally {
        loading.value = false
      }
    },
    (err) => {
      loading.value = false
      console.error('获取位置失败:', err)
      ElMessage.error('获取当前位置失败,请检查定位权限')
    },
    {
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 0
    }
  )
}
</script>

<style scoped>
.location-picker {
  position: relative;
}

.search-results {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  max-width: 500px;
  max-height: 300px;
  overflow-y: auto;
  background: white;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  z-index: 1000;
  margin-top: 4px;
}

.result-item {
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.2s;
}

.result-item:last-child {
  border-bottom: none;
}

.result-item:hover {
  background-color: #f5f7fa;
}

.result-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.result-address {
  font-size: 13px;
  color: #909399;
}

.selected-location {
  margin-top: 12px;
}

.coordinates {
  margin-top: 8px;
  font-size: 13px;
  color: #909399;
}

.loading-hint,
.error-hint {
  margin-top: 8px;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.loading-hint {
  color: #409eff;
}

.error-hint {
  color: #f56c6c;
}

.is-loading {
  animation: rotating 2s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
