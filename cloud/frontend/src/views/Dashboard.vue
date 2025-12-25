<template>
  <div class="dashboard">
    <!-- 页面标题 -->
    <div class="dashboard-header">
      <div class="header-content">
        <h1 class="title">监控大屏</h1>
        <p class="subtitle">实时监控储能柜运行状态与性能指标</p>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-icon primary">
              <el-icon :size="24"><Box /></el-icon>
            </div>
            <el-tag size="small" effect="plain" type="success">+12%</el-tag>
          </div>
          <h3 class="stat-value">{{ stats.totalCabinets }}</h3>
          <p class="stat-label">储能柜总数</p>
        </div>
      </el-col>

      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-icon success">
              <el-icon :size="24"><Connection /></el-icon>
            </div>
            <span class="stat-percentage">{{ onlinePercentage }}%</span>
          </div>
          <h3 class="stat-value">{{ stats.onlineCabinets }}</h3>
          <p class="stat-label">在线设备</p>
          <el-progress :percentage="onlinePercentage" :stroke-width="4" :show-text="false" />
        </div>
      </el-col>

      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-icon warning">
              <el-icon :size="24"><Warning /></el-icon>
            </div>
            <el-tag v-if="stats.activeAlerts > 0" size="small" effect="plain" type="danger">
              {{ stats.criticalAlerts }} 严重
            </el-tag>
          </div>
          <h3 class="stat-value">{{ stats.activeAlerts }}</h3>
          <p class="stat-label">活跃告警</p>
        </div>
      </el-col>

      <el-col :xs="12" :sm="12" :lg="6">
        <div class="stat-card">
          <div class="stat-header">
            <div class="stat-icon info">
              <el-icon :size="24"><SwitchButton /></el-icon>
            </div>
            <el-tag v-if="stats.offlineCabinets > 0" size="small" effect="plain" type="warning">
              需关注
            </el-tag>
            <el-tag v-else size="small" effect="plain" type="success">
              正常
            </el-tag>
          </div>
          <h3 class="stat-value">{{ stats.offlineCabinets }}</h3>
          <p class="stat-label">离线设备</p>
        </div>
      </el-col>

    </el-row>

    <!-- 地图展示 - 左右布局 -->
    <el-row :gutter="20" class="content-row">
      <el-col :xs="24" :lg="17">
        <el-card class="map-card" shadow="never">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon><Location /></el-icon>
                <span>储能柜位置分布</span>
              </div>
            </div>
          </template>
          <div ref="mapContainer" class="map-container"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="7">
        <el-card class="location-list-card" shadow="never">
          <template #header>
            <div class="card-header">
              <div class="card-title">
                <el-icon><Box /></el-icon>
                <span>储能柜列表</span>
              </div>
            </div>
          </template>
          <div class="location-list-container">
            <div v-if="cabinetLocations.length > 0" class="location-list-items">
              <div
                v-for="location in cabinetLocations"
                :key="location.cabinet_id"
                class="location-list-item"
                :class="{ active: selectedCabinetId === location.cabinet_id }"
                @click="focusOnCabinet(location)"
              >
                <div class="location-item-content">
                  <div class="location-item-header">
                    <h4 class="location-name">{{ location.name }}</h4>
                    <el-tag
                      :type="getStatusType(location.status)"
                      size="small"
                      effect="plain"
                    >
                      {{ getStatusText(location.status) }}
                    </el-tag>
              </div>
                  <p class="location-address">
                    <el-icon><Location /></el-icon>
                    {{ location.location || '未设置位置' }}
                  </p>
                  <div v-if="location.latitude && location.longitude" class="location-coords">
                    <small>
                      {{ location.latitude.toFixed(6) }}, {{ location.longitude.toFixed(6) }}
                    </small>
              </div>
                  <div v-else class="location-no-coords">
                    <small>未设置坐标</small>
            </div>
              </div>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import {
  Box, Connection, Warning,
  Location, SwitchButton
} from '@element-plus/icons-vue'
import { cabinetApi } from '@/api'
import type { CabinetLocation } from '@/types/api'
import { ElMessage } from 'element-plus'

// 统计数据
const stats = ref({
  totalCabinets: 0,
  onlineCabinets: 0,
  activeAlerts: 0,
  criticalAlerts: 0,
  offlineCabinets: 0, // 离线设备数
})

const onlinePercentage = computed(() =>
  stats.value.totalCabinets > 0
    ? Math.round((stats.value.onlineCabinets / stats.value.totalCabinets) * 100)
    : 0
)

// 地图
const mapContainer = ref<HTMLDivElement>()
const cabinetLocations = ref<CabinetLocation[]>([])
const mapInitFailed = ref(false)
let map: any = null
let mapConfig: any = null
let isMapSDKLoaded = false
const selectedCabinetId = ref<string | null>(null)
const markerLayer: any = ref(null)
const isInitialMapLoad = ref(true) // 标记是否是初始加载

// 加载统计数据
const loadStatistics = async () => {
  try {
    const response = await cabinetApi.getStatistics()
    const data = response.data
    stats.value = {
      totalCabinets: data.total_cabinets,
      onlineCabinets: data.active_cabinets, // 使用active_cabinets表示在线设备数
      activeAlerts: stats.value.activeAlerts, // 告警数据暂时保持，后续可以从告警API获取
      criticalAlerts: stats.value.criticalAlerts,
      offlineCabinets: data.offline_cabinets,
    }
  } catch (error) {
    console.error('Failed to load statistics:', error)
    ElMessage.error('加载统计数据失败')
  }
}

// 获取状态类型
const getStatusType = (status: string) => {
  switch (status) {
    case 'online':
      return 'success'
    case 'offline':
      return 'info'
    case 'maintenance':
      return 'warning'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'active':
    case 'online':
      return '在线'
    case 'inactive':
    case 'offline':
      return '离线'
    case 'maintenance':
      return '维护'
    case 'pending':
      return '待激活'
    case 'error':
      return '故障'
    default:
      return '未知'
  }
}

// 聚焦到指定储能柜
const focusOnCabinet = (location: CabinetLocation) => {
  if (!location.latitude || !location.longitude) {
    ElMessage.warning('该储能柜未设置位置坐标')
    return
  }

  // 更新选中状态
  selectedCabinetId.value = location.cabinet_id
  
  // 标记为非初始加载，允许后续调整视野
  isInitialMapLoad.value = false

  if (!map || !isMapSDKLoaded) {
    return
  }

  const TMap = (window as any).TMap
  if (!TMap) return

  try {
    // 定位到该储能柜
    const position = new TMap.LatLng(location.latitude, location.longitude)
    map.setCenter(position)
    map.setZoom(15) // 设置合适的缩放级别

    // 高亮该标记（可以通过动画或改变样式）
    highlightMarker(location.cabinet_id)
  } catch (error) {
    console.error('定位失败:', error)
}
}

// 高亮指定标记
const highlightMarker = (_cabinetId: string) => {
  if (!markerLayer.value) return

  // 重新渲染标记以更新选中状态
  renderMapMarkers()
}

// 加载储能柜位置数据
const loadCabinetLocations = async () => {
  try {
    const response = await cabinetApi.getLocations()
    cabinetLocations.value = response.data
    console.log('Loaded cabinet locations:', cabinetLocations.value.length)
    
    // 如果地图已经初始化，重新渲染标记
    if (map && isMapSDKLoaded) {
      renderMapMarkers()
    } else if (mapInitFailed.value) {
      renderSimpleLocationList()
    }
  } catch (error) {
    console.error('Failed to load cabinet locations:', error)
    ElMessage.error('加载储能柜位置失败')
  }
}

// 加载地图配置
const loadMapConfig = async () => {
  try {
    const response = await fetch('/api/v1/config')
    const data = await response.json()
    mapConfig = data.map
    console.log('Map config loaded:', mapConfig)
  } catch (error) {
    console.error('Failed to load map config:', error)
    mapInitFailed.value = true
    renderSimpleLocationList()
  }
}

// 初始化地图
const initMap = async () => {
  if (!mapContainer.value) return

  // 如果没有加载配置，先加载
  if (!mapConfig) {
    await loadMapConfig()
  }

  // 检查地图功能是否启用
  if (!mapConfig || mapConfig.enabled === false) {
    console.log('地图功能已禁用，使用简单列表展示')
    mapInitFailed.value = true
    renderSimpleLocationList()
    return
  }

  // 检查是否有腾讯地图Key
  if (!mapConfig.tencent_map_key) {
    console.warn('腾讯地图Key未配置，使用简单列表展示')
    mapInitFailed.value = true
    renderSimpleLocationList()
    return
  }

  // 加载腾讯地图SDK
  if (!isMapSDKLoaded) {
    try {
      await loadTencentMapSDK()
    } catch (error) {
      console.warn('腾讯地图SDK加载失败，使用简单列表展示:', error)
      mapInitFailed.value = true
      renderSimpleLocationList()
      return
    }
  }

  // 初始化腾讯地图
  if (isMapSDKLoaded && (window as any).TMap) {
    initTencentMap()
  } else {
    // 如果SDK加载失败，使用简单列表展示
    console.warn('腾讯地图SDK未就绪，使用简单列表展示')
    mapInitFailed.value = true
    renderSimpleLocationList()
  }
}

// 加载腾讯地图SDK
const loadTencentMapSDK = (): Promise<void> => {
  return new Promise((resolve, reject) => {
    if (isMapSDKLoaded || (window as any).TMap) {
      isMapSDKLoaded = true
      resolve()
      return
    }

    // 使用官方CDN直接加载腾讯地图SDK
    const scriptSrc = `https://map.qq.com/api/gljs?v=1.exp&key=${mapConfig.tencent_map_key}`

    console.log('开始加载腾讯地图SDK:', scriptSrc)
    console.log('当前页面URL:', window.location.href)
    console.log('当前域名:', window.location.hostname)
    console.log('当前端口:', window.location.port)

    const script = document.createElement('script')
    script.src = scriptSrc
    script.async = true
    script.defer = false
    script.type = 'text/javascript'
    // 移除crossOrigin属性，避免CORS检查（腾讯云已配置域名白名单）
    
    let timeoutId: ReturnType<typeof setTimeout> | null = null
    
    script.onload = () => {
      if (timeoutId) clearTimeout(timeoutId)
      
      // 等待一小段时间确保TMap对象已初始化
      setTimeout(() => {
        // 检查TMap是否真的加载成功
        if ((window as any).TMap) {
          isMapSDKLoaded = true
          console.log('腾讯地图SDK加载成功')
          resolve()
        } else {
          console.error('腾讯地图SDK加载完成但TMap对象不存在，等待更长时间...')
          // 再等待一段时间，有些SDK可能需要更长时间初始化
          setTimeout(() => {
            if ((window as any).TMap) {
              isMapSDKLoaded = true
              console.log('腾讯地图SDK加载成功（延迟初始化）')
              resolve()
            } else {
              console.error('腾讯地图SDK加载完成但TMap对象仍然不存在')
              reject(new Error('腾讯地图SDK加载失败：TMap对象未定义'))
            }
          }, 1000)
        }
      }, 100)
    }
    
    script.onerror = (error) => {
      if (timeoutId) clearTimeout(timeoutId)
      script.remove()
      console.error('腾讯地图SDK加载失败:', error)
      console.error('脚本URL:', script.src)
      console.error('当前域名:', window.location.hostname)
      console.error('当前协议:', window.location.protocol)
      console.warn('可能的原因：')
      console.warn('1. 网络连接问题 - 请检查网络连接')
      console.warn('2. API Key域名白名单未配置 - 请在腾讯云控制台配置域名白名单')
      console.warn('3. API Key无效 - 请检查API Key是否正确')
      console.warn('4. 域名白名单需要包含以下域名：')
      console.warn('   - localhost')
      console.warn('   - 127.0.0.1')
      console.warn('   - localhost:8002')
      console.warn('   - 127.0.0.1:8002')
      console.warn('   - ' + window.location.hostname)
      console.warn('   - ' + window.location.host)
      reject(new Error('腾讯地图SDK加载失败，请检查网络连接和API Key配置'))
    }
    
    // 设置超时（30秒）
    timeoutId = setTimeout(() => {
      script.remove()
      console.error('腾讯地图SDK加载超时')
      reject(new Error('腾讯地图SDK加载超时，请检查网络连接或API Key配置'))
    }, 30000)
    
    document.head.appendChild(script)
  })
}

// 初始化腾讯地图实例
const initTencentMap = () => {
  if (!mapContainer.value || !isMapSDKLoaded) return

  const TMap = (window as any).TMap
  if (!TMap) {
    console.error('TMap SDK未加载')
    renderSimpleLocationList()
    return
  }

  try {
    // 创建地图实例
    const center = mapConfig.default_center || { latitude: 30.287459, longitude: 120.153576 }
    map = new TMap.Map(mapContainer.value, {
      center: new TMap.LatLng(center.latitude, center.longitude),
      zoom: mapConfig.default_zoom || 12,
      viewMode: '2D'
    })

    console.log('腾讯地图实例创建成功')

    // 渲染标记
    renderMapMarkers()
  } catch (error) {
    console.error('初始化腾讯地图失败:', error)
    mapInitFailed.value = true
    renderSimpleLocationList()
  }
}

// 渲染地图标记
const renderMapMarkers = () => {
  if (!map || !isMapSDKLoaded) return

  const TMap = (window as any).TMap
  if (!TMap) return

  // 清除旧标记
  if (markerLayer.value) {
    markerLayer.value.setMap(null)
  }

  // 过滤有效的位置数据
  const validLocations = cabinetLocations.value.filter(
    loc => loc.latitude != null && loc.longitude != null
  )

  if (validLocations.length === 0) {
    console.warn('没有有效的储能柜位置数据')
    return
  }

  // 创建标记数据
  const geometries = validLocations.map(loc => ({
    id: loc.cabinet_id,
    position: new TMap.LatLng(loc.latitude!, loc.longitude!),
    properties: {
      title: loc.name,
      status: loc.status,
      location: loc.location || '未设置位置',
      cabinetId: loc.cabinet_id
    }
  }))

  // 创建红旗图标SVG
  const createFlagIcon = (color: string, isSelected: boolean = false) => {
    const borderColor = isSelected ? '#667eea' : 'transparent'
    const borderWidth = isSelected ? 3 : 0
    const svg = `
      <svg xmlns="http://www.w3.org/2000/svg" width="30" height="40" viewBox="0 0 30 40">
        <rect x="0" y="0" width="30" height="30" fill="${color}" stroke="${borderColor}" stroke-width="${borderWidth}"/>
        <polygon points="0,0 30,0 15,15" fill="#ffffff"/>
        <circle cx="15" cy="7.5" r="2" fill="${color}"/>
        <rect x="14" y="30" width="2" height="10" fill="#8b5cf6"/>
      </svg>
    `
    return 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
  }

  // 定义标记样式 - 使用红旗图标
  const styles: any = {
    online: new TMap.MarkerStyle({
      width: 30,
      height: 40,
      anchor: { x: 15, y: 40 },
      src: createFlagIcon('#dc2626', false)
    }),
    active: new TMap.MarkerStyle({
      width: 30,
      height: 40,
      anchor: { x: 15, y: 40 },
      src: createFlagIcon('#10b981', false)  // 绿色表示正常
    }),
    offline: new TMap.MarkerStyle({
      width: 30,
      height: 40,
      anchor: { x: 15, y: 40 },
      src: createFlagIcon('#94a3b8', false)
    }),
    maintenance: new TMap.MarkerStyle({
      width: 30,
      height: 40,
      anchor: { x: 15, y: 40 },
      src: createFlagIcon('#f59e0b', false)
    }),
    error: new TMap.MarkerStyle({
      width: 30,
      height: 40,
      anchor: { x: 15, y: 40 },
      src: createFlagIcon('#ef4444', false)
    }),
    // 选中状态的样式
    'online-selected': new TMap.MarkerStyle({
      width: 36,
      height: 48,
      anchor: { x: 18, y: 48 },
      src: createFlagIcon('#dc2626', true)
    }),
    'active-selected': new TMap.MarkerStyle({
      width: 36,
      height: 48,
      anchor: { x: 18, y: 48 },
      src: createFlagIcon('#10b981', true)
    }),
    'offline-selected': new TMap.MarkerStyle({
      width: 36,
      height: 48,
      anchor: { x: 18, y: 48 },
      src: createFlagIcon('#94a3b8', true)
    }),
    'maintenance-selected': new TMap.MarkerStyle({
      width: 36,
      height: 48,
      anchor: { x: 18, y: 48 },
      src: createFlagIcon('#f59e0b', true)
    }),
    'error-selected': new TMap.MarkerStyle({
      width: 36,
      height: 48,
      anchor: { x: 18, y: 48 },
      src: createFlagIcon('#ef4444', true)
    })
  }

  // 创建MultiMarker
  const layer = new TMap.MultiMarker({
    map,
    styles,
    geometries: geometries.map(g => ({
      ...g,
      styleId: selectedCabinetId.value === g.properties.cabinetId
        ? `${g.properties.status}-selected`
        : g.properties.status
    }))
  })

  // 添加点击事件
  layer.on('click', (evt: any) => {
    const { title, location, status, cabinetId } = evt.geometry.properties
    selectedCabinetId.value = cabinetId
    
    const statusText =
      status === 'active' ? '在线同步中' :
      status === 'offline' ? '离线' :
      status === 'pending' ? '待激活' :
      status === 'inactive' ? '已停用' :
      status === 'maintenance' ? '维护中' : '未知'
    
    const infoWindow = new TMap.InfoWindow({
      map,
      position: evt.geometry.position,
      content: `
        <div style="padding: 12px; min-width: 200px;">
          <h4 style="margin: 0 0 8px 0; font-size: 15px; font-weight: 600;">${title}</h4>
          <p style="margin: 4px 0; font-size: 13px; color: #64748b;">
            <strong>状态:</strong> <span style="color: ${
              status === 'active' ? '#10b981' :
              status === 'offline' ? '#94a3b8' :
              status === 'pending' ? '#f59e0b' :
              status === 'inactive' ? '#64748b' :
              status === 'maintenance' ? '#3b82f6' : '#ef4444'
            }">${statusText}</span>
          </p>
          <p style="margin: 4px 0; font-size: 13px; color: #64748b;">
            <strong>位置:</strong> ${location}
          </p>
        </div>
      `
    })
    infoWindow.open()
    
    // 重新渲染以更新选中状态
    renderMapMarkers()
  })

  // 保存引用
  markerLayer.value = layer

  // 只在非初始加载时调整视野以包含所有标记
  // 初始加载时使用配置的 default_zoom，保持省级视图
  if (!isInitialMapLoad.value && validLocations.length > 0) {
    const bounds = new TMap.LatLngBounds()
    validLocations.forEach(loc => {
      bounds.extend(new TMap.LatLng(loc.latitude!, loc.longitude!))
    })
    map.fitBounds(bounds, {
      padding: 50
    })
  }

  console.log(`已渲染 ${validLocations.length} 个储能柜标记`)
}

// 简单的位置列表渲染（替代地图）
const renderSimpleLocationList = () => {
  if (!mapContainer.value) return
  
  const locationList = document.createElement('div')
  locationList.className = 'location-list'
  locationList.innerHTML = `
    <div class="location-list-header">
    </div>
    <div class="location-list-body">
      ${cabinetLocations.value.length > 0 ? 
        cabinetLocations.value.map(loc => `
          <div class="location-item">
            <div class="location-info">
              <h4>${loc.name}</h4>
              <p>${loc.location || '未设置位置'}</p>
              ${loc.latitude && loc.longitude ? 
                `<small>坐标: ${loc.latitude.toFixed(6)}, ${loc.longitude.toFixed(6)}</small>` : 
                '<small>未设置坐标</small>'}
            </div>
            <div class="location-status status-${loc.status}">
              ${loc.status === 'active' ? '在线同步中' :
                loc.status === 'offline' ? '离线' :
                loc.status === 'pending' ? '待激活' :
                loc.status === 'inactive' ? '已停用' :
                loc.status === 'maintenance' ? '维护中' : '未知'}
            </div>
          </div>
        `).join('') :
        ''
      }
    </div>
  `
  mapContainer.value.innerHTML = ''
  mapContainer.value.appendChild(locationList)
}


onMounted(async () => {
  // 注意：腾讯地图SDK在运行时会产生一些CORS错误（浏览器控制台会显示），
  // 这些错误是正常的，不影响地图基本功能。这些错误来自：
  // 1. API Key验证请求（apikey.map.qq.com）
  // 2. 地图瓦片加载（rt*.map.gtimg.com）
  // 3. 图标和样式资源（vectorsdk.map.qq.com）
  // 4. 其他腾讯地图内部资源
  // 如果地图正常显示，可以忽略这些CORS警告。
  
  initMap()
  await Promise.all([loadStatistics(), loadCabinetLocations()])
})

onBeforeUnmount(() => {
  // 清理地图实例
  if (map && map.destroy) {
    try {
      map.destroy()
    } catch (error) {
      console.error('Error destroying map:', error)
    }
  }
})
</script>

<style scoped>
.dashboard {
  padding: 24px;
  background: #f8fafc;
  min-height: calc(100vh - 60px);
}

/* 页面标题 */
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

/* 统计卡片 */
.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  padding: 20px;
  background: white;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  transition: all 0.3s;
  margin-bottom: 16px;
}

.stat-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
  transform: translateY(-2px);
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.stat-icon.primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-icon.success {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.stat-icon.warning {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.stat-icon.info {
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
}

.stat-percentage {
  font-size: 15px;
  font-weight: 600;
  color: #64748b;
}

.stat-value {
  margin: 0 0 4px 0;
  font-size: 32px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -1px;
}

.stat-label {
  margin: 0 0 12px 0;
  font-size: 15px;
  color: #64748b;
  font-weight: 500;
}

/* 卡片 */
.content-row {
  margin-bottom: 20px;
}

:deep(.el-card) {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  margin-bottom: 16px;
}

:deep(.el-card__header) {
  border-bottom: 1px solid #e2e8f0;
  padding: 16px 20px;
  background: #fafbfc;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 17px;
  font-weight: 600;
  color: #0f172a;
}

/* 地图/位置列表 - 左右布局 */
.map-card {
  height: 100%;
}

.map-container {
  height: 600px;
  width: 100%;
}

.location-list-card {
  height: 100%;
}

.location-list-container {
  height: 600px;
  display: flex;
  flex-direction: column;
}

.empty-locations {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.location-list-items {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.location-list-item {
  padding: 16px;
  margin-bottom: 8px;
  background: #f8fafc;
  border: 2px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.location-list-item:hover {
  background: #f1f5f9;
  transform: translateX(4px);
  border-color: #e2e8f0;
}

.location-list-item.active {
  background: #eff6ff;
  border-color: #667eea;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.15);
}

.location-item-content {
  width: 100%;
}

.location-item-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
}

.location-name {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #0f172a;
  flex: 1;
}

.location-address {
  margin: 8px 0 4px 0;
  font-size: 13px;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 4px;
}

.location-coords {
  margin-top: 4px;
}

.location-coords small {
  font-size: 11px;
  color: #94a3b8;
  font-family: monospace;
}

.location-no-coords {
  margin-top: 4px;
}

.location-no-coords small {
  font-size: 11px;
  color: #cbd5e1;
  font-style: italic;
}

/* 图表 */
.chart-container {
  height: 320px;
}

/* 告警列表 */
.alert-list {
  max-height: 320px;
  overflow-y: auto;
}

.alert-item {
  padding: 12px;
  border-bottom: 1px solid #f1f5f9;
  transition: background 0.2s;
  cursor: pointer;
}

.alert-item:hover {
  background: #f8fafc;
}

.alert-item:last-child {
  border-bottom: none;
}

.alert-content {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.alert-message {
  font-size: 15px;
  color: #475569;
  flex: 1;
}

.alert-time {
  font-size: 13px;
  color: #94a3b8;
}

/* 状态列表 */
.status-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.status-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.status-badge {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.status-name {
  font-size: 15px;
  color: #475569;
  font-weight: 500;
  flex: 1;
}

.status-count {
  font-size: 15px;
  color: #0f172a;
  font-weight: 600;
}

/* 快捷操作列表 */
.action-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.action-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
  cursor: pointer;
  transition: all 0.3s;
}

.action-item:hover {
  background: white;
  border-color: #667eea;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.1);
}

.action-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.action-icon.primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.action-icon.success {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.action-icon.info {
  background: linear-gradient(135deg, #06b6d4 0%, #0891b2 100%);
}

.action-icon.warning {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.action-info {
  flex: 1;
}

.action-info h4 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
}

.action-info p {
  margin: 0;
  font-size: 14px;
  color: #64748b;
}

/* 响应式 - 平板 */
@media (max-width: 992px) {
  .dashboard-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
  
  .header-actions {
    width: 100%;
  }
  
  .header-actions .el-button {
    width: 100%;
  }
}

/* 响应式 - 移动端 */
@media (max-width: 768px) {
  .dashboard {
    padding: 12px;
  }

  .dashboard-header {
    padding: 16px;
    margin-bottom: 16px;
  }

  .header-content .title {
    font-size: 20px;
  }

  .header-content .subtitle {
    font-size: 13px;
  }

  .stat-card {
    padding: 16px;
  }

  .stat-value {
    font-size: 22px;
  }

  .stat-label {
    font-size: 13px;
  }

  .stats-row {
    margin-bottom: 16px;
  }

  .content-row {
    margin-bottom: 16px;
  }

  /* 地图高度调整 */
  .map-container {
    height: 280px;
  }

  /* 储能柜列表高度 */
  .location-list-card :deep(.el-card__body) {
    max-height: 300px;
  }

  /* 卡片间距 */
  .map-card,
  .location-list-card {
    margin-bottom: 12px;
  }

  /* 位置列表项 */
  .location-list-item {
    padding: 10px;
  }

  .location-name {
    font-size: 14px;
  }

  .location-address {
    font-size: 12px;
  }

  /* 告警列表 */
  .alert-list {
    max-height: 250px;
  }

  .alert-item {
    padding: 10px;
  }

  .alert-message {
    font-size: 14px;
  }

  /* 状态列表 */
  .status-item {
    gap: 6px;
  }

  .status-name,
  .status-count {
    font-size: 14px;
  }

  /* 快捷操作 */
  .action-item {
    padding: 12px;
    gap: 12px;
  }

  .action-icon {
    width: 40px;
    height: 40px;
  }

  .action-info h4 {
    font-size: 14px;
  }

  .action-info p {
    font-size: 12px;
  }
}

/* 响应式 - 超小屏幕 */
@media (max-width: 480px) {
  .dashboard {
    padding: 8px;
  }

  .dashboard-header {
    padding: 12px;
  }

  .header-content .title {
    font-size: 18px;
  }

  .stat-card {
    padding: 12px;
  }

  .stat-value {
    font-size: 20px;
  }

  .stat-icon {
    width: 36px;
    height: 36px;
  }

  .map-container {
    height: 220px;
  }

  .chart-container {
    height: 240px;
  }
}
</style>
