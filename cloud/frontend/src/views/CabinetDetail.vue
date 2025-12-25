<template>
  <div class="cabinet-detail">
    <!-- 返回按钮 -->
    <el-page-header @back="goBack" title="返回列表" />

    <div v-if="cabinetStore.loading" class="loading">
      <el-skeleton :rows="8" animated />
    </div>

    <div v-else-if="cabinet" class="detail-content">
      <!-- 基本信息卡片 -->
      <el-card class="info-card">
        <template #header>
          <div class="card-header">
            <span>基本信息</span>
            <div class="header-status">
              <StatusBadge :status="cabinet.status" type="cabinet" />
            </div>
          </div>
        </template>

        <el-descriptions :column="2" border>
          <el-descriptions-item label="储能柜ID">
            {{ cabinet.cabinet_id }}
          </el-descriptions-item>
          <el-descriptions-item label="名称">
            {{ cabinet.name }}
          </el-descriptions-item>
          <el-descriptions-item label="位置">
            {{ cabinet.location || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="容量">
            {{ cabinet.capacity_kwh ? `${cabinet.capacity_kwh} kWh` : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="MAC地址">
            {{ cabinet.mac_address }}
          </el-descriptions-item>
          <el-descriptions-item label="最后同步时间">
            {{ cabinet.last_sync_at ? formatTime(cabinet.last_sync_at) : '从未同步' }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ formatTime(cabinet.created_at) }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- API Key 管理卡片 -->
      <el-card class="apikey-card" v-if="cabinet.activation_status === 'activated'">
        <template #header>
          <div class="card-header">
            <span>API Key 管理</span>
            <el-button size="small" @click="loadAPIKeyInfo" :loading="apikeyLoading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </template>

        <div v-if="apikeyLoading" class="apikey-loading">
          <el-skeleton :rows="2" animated />
        </div>
        <div v-else class="apikey-content">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="API Key">
              <template v-if="apikeyInfo.has_api_key">
                <el-text class="apikey-masked">{{ apikeyInfo.api_key_masked }}</el-text>
                <el-button
                  size="small"
                  type="primary"
                  plain
                  style="margin-left: 10px;"
                  @click="handleRegenerateAPIKey"
                  :loading="regenerating"
                >
                  <el-icon><RefreshRight /></el-icon>
                  重新生成
                </el-button>
                <el-button
                  size="small"
                  type="danger"
                  plain
                  @click="handleRevokeAPIKey"
                  :loading="revoking"
                >
                  <el-icon><Delete /></el-icon>
                  撤销
                </el-button>
              </template>
              <template v-else>
                <el-text type="info">未生成</el-text>
                <el-button
                  size="small"
                  type="primary"
                  style="margin-left: 10px;"
                  @click="handleRegenerateAPIKey"
                  :loading="regenerating"
                >
                  <el-icon><Plus /></el-icon>
                  生成 API Key
                </el-button>
              </template>
            </el-descriptions-item>
            <el-descriptions-item label="生成时间" v-if="apikeyInfo.has_api_key && apikeyInfo.generated_at">
              {{ formatTime(apikeyInfo.generated_at) }}
            </el-descriptions-item>
          </el-descriptions>

          <el-alert
            v-if="showCredentials"
            title="API 凭证已生成"
            type="success"
            :closable="false"
            style="margin-top: 16px;"
          >
            <template #default>
              <div class="credentials-display">
                <p><strong>请妥善保存API Key,仅显示一次:</strong></p>
                <el-descriptions :column="1" border size="small" style="margin-top: 10px;">
                  <el-descriptions-item label="API Key">
                    <el-text class="credential-text">{{ newCredentials.api_key }}</el-text>
                    <el-button
                      size="small"
                      @click="copyToClipboard(newCredentials.api_key)"
                      style="margin-left: 10px;"
                    >
                      <el-icon><CopyDocument /></el-icon>
                      复制
                    </el-button>
                  </el-descriptions-item>
                </el-descriptions>
                <el-button
                  type="primary"
                  size="small"
                  style="margin-top: 10px;"
                  @click="showCredentials = false"
                >
                  我已保存,关闭
                </el-button>
              </div>
            </template>
          </el-alert>
        </div>
      </el-card>

      <!-- 实时传感器数据卡片 -->
      <el-card class="realtime-sensors-card">
        <template #header>
          <div class="card-header">
            <span>传感器实时数据</span>
            <el-button size="small" @click="loadLatestSensorData" :loading="sensorLoading">
              <el-icon><Refresh /></el-icon>
              Refresh
            </el-button>
          </div>
        </template>
        
        <div v-if="sensorLoading" class="sensor-loading">
          <el-skeleton :rows="2" animated />
        </div>
        <div v-else class="sensor-grid">
          <!-- 如果有数据，显示实际数据 -->
          <el-row :gutter="16" v-if="latestSensorData.length > 0">
            <el-col 
              :xs="24" 
              :sm="12" 
              :md="8" 
              :lg="6"
              :xl="6"
              v-for="sensor in latestSensorData" 
              :key="sensor.device_id"
            >
              <SensorCard
                :sensor-type="sensor.sensor_type"
                :device-name="sensor.name"
                :device-id="sensor.device_id"
                :value="sensor.value"
                :unit="sensor.unit"
                :quality="sensor.quality"
                :status="sensor.status || (sensor.value === undefined || sensor.value === 0 ? 'offline' : 'normal')"
                :timestamp="sensor.timestamp"
                :threshold="getSensorThreshold(sensor.sensor_type)"
                :alarm-time="getLatestAlarmTime(sensor.device_id)"
                @view-detail="handleViewSensorDetail"
              />
            </el-col>
          </el-row>
          <!-- 如果没有数据但有储能柜，显示7种默认传感器卡片（离线状态） -->
          <el-row :gutter="16" v-else-if="cabinet">
            <el-col 
              :xs="24" 
              :sm="12" 
              :md="8" 
              :lg="6"
              :xl="6"
              v-for="sensorType in defaultSensorTypes" 
              :key="sensorType.type"
            >
              <SensorCard
                :sensor-type="sensorType.type"
                :device-name="sensorType.nameCN"
                :device-id="`${sensorType.type}_SENSOR`"
                :value="undefined"
                :unit="sensorType.unit"
                :quality="0"
                :status="'offline'"
                :timestamp="new Date().toISOString()"
                :threshold="getSensorThreshold(sensorType.type)"
                :alarm-time="undefined"
                @view-detail="handleViewSensorDetail"
              />
            </el-col>
          </el-row>
          <!-- 如果既没有数据也没有储能柜信息，显示空状态 -->
          <el-empty v-else description="No sensor data available" />
        </div>
      </el-card>

      <!-- 告警信息卡片 -->
      <el-card class="alert-card">
        <template #header>
          <div class="card-header">
            <span>最近告警</span>
            <el-button size="small" @click="loadAlerts">查看全部</el-button>
          </div>
        </template>

        <div v-if="alertsLoading" class="alerts-loading">
          <el-skeleton :rows="3" animated />
        </div>

        <el-table
          v-else-if="alerts.length > 0"
          :data="alerts"
          style="width: 100%"
        >
          <el-table-column prop="alert_type" label="告警类型" width="120" />
          <el-table-column prop="message" label="告警信息" />
          <el-table-column label="严重程度" width="100">
            <template #default="{ row }">
              <el-tag
                :type="getSeverityType(row.severity)"
                size="small"
              >
                {{ getSeverityLabel(row.severity) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag
                :type="row.resolved ? 'success' : 'danger'"
                size="small"
              >
                {{ row.resolved ? '已解决' : '未解决' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
        </el-table>

        <el-empty v-else description="暂无告警信息" />
      </el-card>

    </div>

    <el-empty v-else description="储能柜不存在" />
    
    <!-- 传感器详情对话框 -->
    <el-dialog
      v-model="sensorDetailDialogVisible"
      :title="`${selectedSensor?.deviceName || 'Sensor'} Details`"
      width="70%"
      :close-on-click-modal="false"
    >
      <div v-if="selectedSensor" class="sensor-detail-content">
        <el-tabs type="border-card">
          <!-- 实时数据标签页 -->
          <el-tab-pane label="Real-time Data">
            <div class="sensor-info-grid">
              <el-descriptions :column="2" border>
                <el-descriptions-item label="Device ID">
                  {{ selectedSensor.deviceId }}
                </el-descriptions-item>
                <el-descriptions-item label="Sensor Type">
                  {{ selectedSensor.sensorType }}
                </el-descriptions-item>
                <el-descriptions-item label="Device Name">
                  {{ selectedSensor.deviceName }}
                </el-descriptions-item>
                <el-descriptions-item label="Unit">
                  {{ selectedSensor.unit }}
                </el-descriptions-item>
              </el-descriptions>
              
              <div v-if="getCurrentSensorData" class="current-sensor-data">
                <h4>Current Reading</h4>
                <SensorCard
                  :sensor-type="getCurrentSensorData.sensor_type"
                  :device-name="getCurrentSensorData.name"
                  :device-id="getCurrentSensorData.device_id"
                  :value="getCurrentSensorData.value"
                  :unit="getCurrentSensorData.unit"
                  :quality="getCurrentSensorData.quality"
                  :status="getCurrentSensorData.status || (getCurrentSensorData.value === undefined ? 'offline' : 'normal')"
                  :timestamp="getCurrentSensorData.timestamp"
                  :threshold="getSensorThreshold(getCurrentSensorData.sensor_type)"
                  :alarm-time="getLatestAlarmTime(getCurrentSensorData.device_id)"
                />
              </div>
            </div>
          </el-tab-pane>
          
          <!-- 历史数据标签页 -->
          <el-tab-pane label="Historical Data">
            <div class="historical-data-placeholder">
              <el-empty description="Historical data chart will be available in the next version">
                <el-button type="primary" @click="sensorDetailDialogVisible = false">
                  Close
                </el-button>
              </el-empty>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
      
      <template #footer>
        <el-button @click="sensorDetailDialogVisible = false">Close</el-button>
        <el-button type="primary" @click="sensorDetailDialogVisible = false">
          OK
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, computed, ref, watch, nextTick } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Refresh, RefreshRight, Delete, Plus, CopyDocument } from '@element-plus/icons-vue';
import { useCabinetStore } from '@/store/cabinet';
import { cabinetApi } from '@/api';
import StatusBadge from '@/components/StatusBadge.vue';
import SensorCard from '@/components/SensorCard.vue';
import type { Alert, LatestSensorData } from '@/types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const WS_BASE_URL = (import.meta.env.VITE_WS_BASE_URL || '').trim();

/**
 * 构建WebSocket连接地址，优先使用显式配置，其次根据当前页面地址推导。
 */
const resolveWebSocketUrl = (): string => {
  try {
    // 如果配置了显式的 WebSocket 地址
    if (WS_BASE_URL) {
      const customUrl = new URL(WS_BASE_URL.startsWith('http') || WS_BASE_URL.startsWith('ws') ? WS_BASE_URL : `http://${WS_BASE_URL}`);
      if (customUrl.protocol === 'http:' || customUrl.protocol === 'https:') {
        customUrl.protocol = customUrl.protocol === 'https:' ? 'wss:' : 'ws:';
      }
      if (!customUrl.pathname || customUrl.pathname === '/') {
        customUrl.pathname = '/ws';
      }
      return customUrl.toString();
    }

    // 如果 API_BASE_URL 是相对路径，使用当前页面的协议和主机
    if (API_BASE_URL.startsWith('/')) {
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      return `${wsProtocol}//${window.location.host}/ws`;
    }

    // 如果 API_BASE_URL 是绝对路径，从中推导 WebSocket 地址
    const apiUrl = new URL(API_BASE_URL);
    const wsProtocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${wsProtocol}//${apiUrl.host}/ws`;
  } catch (error) {
    console.error('构建WebSocket地址失败，使用回退地址', error);
    const fallbackProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${fallbackProtocol}//${window.location.host}/ws`;
  }
};

const route = useRoute();
const router = useRouter();
const cabinetStore = useCabinetStore();

const cabinet = computed(() => cabinetStore.currentCabinet);

// 实时传感器数据
const latestSensorData = ref<LatestSensorData[]>([]);
const sensorLoading = ref(false);

// WebSocket连接
let ws: WebSocket | null = null;
const WS_RECONNECT_INTERVAL = 3000; // 3秒重连间隔
let wsReconnectTimer: number | null = null;

// 轮询定时器（作为WebSocket的fallback）
let pollingTimer: number | null = null;
const POLLING_INTERVAL = 30000; // 30秒轮询一次（作为fallback，WebSocket正常时不会使用）

// 传感器详情对话框
const sensorDetailDialogVisible = ref(false);
const selectedSensor = ref<{
  deviceId: string;
  deviceName: string;
  sensorType: string;
  unit: string;
} | null>(null);

// 告警数据
const alerts = ref<Alert[]>([]);
const alertsLoading = ref(false);

// API Key管理
const apikeyInfo = ref<{
  cabinet_id: string;
  activation_status: string;
  has_api_key: boolean;
  api_key_masked?: string;
  generated_at?: string;
}>({
  cabinet_id: '',
  activation_status: '',
  has_api_key: false,
});
const apikeyLoading = ref(false);
const regenerating = ref(false);
const revoking = ref(false);
const showCredentials = ref(false);
const newCredentials = ref<{
  cabinet_id: string;
  api_key: string;
}>({
  cabinet_id: '',
  api_key: '',
});

// 加载最新传感器数据
const loadLatestSensorData = async (silent = false) => {
  const id = route.params.id as string;
  if (!id) return;
  
  // 静默刷新时不显示加载状态
  if (!silent) {
    sensorLoading.value = true;
  }
  
  try {
    const response = await cabinetApi.getLatestSensorData(id);
    // 修正单位显示：将 CO 传感器的 MG/M3 统一显示为 ppm
    const sensors = (response.data || []).map((sensor: LatestSensorData) => {
      if (sensor.sensor_type === 'co' && sensor.unit === 'MG/M3') {
        return { ...sensor, unit: 'ppm' };
      }
      return sensor;
    });
    latestSensorData.value = sensors;
  } catch (error: any) {
    console.error('Failed to load latest sensor data:', error);
    // 静默刷新时不显示错误信息
    if (!silent) {
      ElMessage.error(error.message || 'Failed to load sensor data');
    }
  } finally {
    if (!silent) {
      sensorLoading.value = false;
    }
  }
};

// WebSocket消息类型
interface WebSocketMessage {
  type: string;
  data: any;
  timestamp: string;
}

// 初始化WebSocket连接
const initWebSocket = () => {
  const cabinetId = route.params.id as string;
  if (!cabinetId) return;

  const wsUrl = resolveWebSocketUrl();
  console.log('Connecting to WebSocket:', wsUrl);

  try {
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('WebSocket connected');
      ElMessage.success('实时数据连接已建立');
      // WebSocket连接成功后，停止轮询
      stopPolling();
    };

    ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        console.log('Received WebSocket message:', message.type, message.data);
        handleWebSocketMessage(message);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error, event.data);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      // WebSocket错误时，启动轮询作为fallback
      startPolling();
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      ws = null;
      // WebSocket断开时，启动轮询作为fallback
      startPolling();
      // 尝试重连
      scheduleReconnect();
    };
  } catch (error) {
    console.error('Failed to create WebSocket connection:', error);
    // WebSocket创建失败时，启动轮询作为fallback
    startPolling();
  }
};

// 处理WebSocket消息
const handleWebSocketMessage = async (message: WebSocketMessage) => {
  const cabinetId = route.params.id as string;
  console.log('Handling WebSocket message:', message.type, 'for cabinet:', cabinetId);
  
  if (message.type === 'sensor_data') {
    // 单个传感器数据更新
    const data = message.data as {
      cabinet_id: string;
      device_id: string;
      sensor_type: string;
      name: string;
      unit: string;
      value: number;
      quality: number;
      status: string;
      timestamp: string;
    };
    
    console.log('Processing sensor_data:', data.device_id, 'cabinet_id:', data.cabinet_id, 'current:', cabinetId);
    
    // 只处理当前储能柜的数据
    if (data.cabinet_id === cabinetId) {
      // 更新对应传感器的数据
      const index = latestSensorData.value.findIndex(
        (s: LatestSensorData) => s.device_id === data.device_id
      );
      
      // 修正单位显示：将 CO 传感器的 MG/M3 统一显示为 ppm
      const displayUnit = (data.sensor_type === 'co' && data.unit === 'MG/M3')
        ? 'ppm'
        : data.unit;
      
      const updatedSensor: LatestSensorData = {
        device_id: data.device_id,
        sensor_type: data.sensor_type,
        name: data.name,
        unit: displayUnit,
        value: data.value,
        quality: data.quality,
        status: data.status,
        timestamp: data.timestamp
      };

      if (index >= 0) {
        // 直接修改对象属性，触发响应式更新
        console.log('Updating existing sensor at index:', index, 'old value:', latestSensorData.value[index].value, 'new value:', updatedSensor.value);
        Object.assign(latestSensorData.value[index], updatedSensor);
        console.log('✅ Sensor updated, new value:', latestSensorData.value[index].value);
      } else {
        // 如果不存在，添加到列表
        console.log('Adding new sensor:', updatedSensor);
        latestSensorData.value.push(updatedSensor);
        console.log('✅ Sensor added, array length:', latestSensorData.value.length);
      }

      console.log('✅ Updated sensor data via WebSocket:', data.device_id, data.value, 'Total sensors:', latestSensorData.value.length);
    } else {
      console.log('⚠️ Ignoring sensor data for different cabinet:', data.cabinet_id, '!=', cabinetId);
    }
  } else if (message.type === 'latest_sensor_data') {
    // 批量更新（兼容旧格式）
    const data = message.data as { cabinet_id: string; sensors: LatestSensorData[] };
    
    if (data.cabinet_id === cabinetId && Array.isArray(data.sensors)) {
      console.log('✅ Batch updating sensors:', data.sensors.length);
      // 修正单位显示：将 CO 传感器的 MG/M3 统一显示为 ppm
      const sensors = data.sensors.map((sensor: LatestSensorData) => {
        if (sensor.sensor_type === 'co' && sensor.unit === 'MG/M3') {
          return { ...sensor, unit: 'ppm' };
        }
        return sensor;
      });
      latestSensorData.value = sensors;
      await nextTick();
      console.log('Updated sensor data via WebSocket:', data.sensors.length, 'sensors');
    }
  } else {
    console.log('⚠️ Unknown message type:', message.type);
  }
};

// 安排WebSocket重连
const scheduleReconnect = () => {
  if (wsReconnectTimer) {
    clearTimeout(wsReconnectTimer);
  }
  
  wsReconnectTimer = window.setTimeout(() => {
    console.log('Attempting to reconnect WebSocket...');
    initWebSocket();
  }, WS_RECONNECT_INTERVAL);
};

// 关闭WebSocket连接
const closeWebSocket = () => {
  if (ws) {
    ws.close();
    ws = null;
  }
  if (wsReconnectTimer) {
    clearTimeout(wsReconnectTimer);
    wsReconnectTimer = null;
  }
};

// 启动传感器数据轮询（作为WebSocket的fallback）
const startPolling = () => {
  stopPolling(); // 清除可能存在的旧定时器
  
  // 如果WebSocket已连接，不启动轮询
  if (ws && ws.readyState === WebSocket.OPEN) {
    return;
  }
  
  pollingTimer = window.setInterval(() => {
    // 静默刷新，不显示加载状态和错误信息
    loadLatestSensorData(true);
  }, POLLING_INTERVAL);
  
  console.log('Started polling as fallback (WebSocket not available)');
};

// 停止传感器数据轮询
const stopPolling = () => {
  if (pollingTimer) {
    clearInterval(pollingTimer);
    pollingTimer = null;
  }
};

// 处理查看传感器详情
const handleViewSensorDetail = (deviceId: string) => {
  const sensor = latestSensorData.value.find((s: LatestSensorData) => s.device_id === deviceId);
  if (sensor) {
    selectedSensor.value = {
      deviceId: sensor.device_id,
      deviceName: sensor.name,
      sensorType: sensor.sensor_type,
      unit: sensor.unit
    };
    sensorDetailDialogVisible.value = true;
  }
};

// 获取当前选中传感器的数据
const getCurrentSensorData = computed(() => {
  if (!selectedSensor.value) return null;
  return latestSensorData.value.find((s: LatestSensorData) => s.device_id === selectedSensor.value!.deviceId);
});

// 传感器默认阈值映射
const sensorThresholds: Record<string, { min?: number; max?: number }> = {
  'co2': { max: 5000 },
  'co': { max: 50 },
  'smoke': { max: 1000 },
  'liquid_level': { min: 0, max: 900 },
  'conductivity': { max: 10 },
  'temperature': { min: -10, max: 60 },
  'flow': { max: 100 }
};

// 7种默认传感器类型配置
const defaultSensorTypes = [
  { type: 'co2', nameCN: 'CO2传感器', unit: 'ppm' },
  { type: 'co', nameCN: 'CO传感器', unit: 'ppm' },
  { type: 'smoke', nameCN: '烟雾传感器', unit: 'ppm' },
  { type: 'liquid_level', nameCN: '液位传感器', unit: 'mm' },
  { type: 'conductivity', nameCN: '电导率传感器', unit: 'mS/cm' },
  { type: 'temperature', nameCN: '温度传感器', unit: '°C' },
  { type: 'flow', nameCN: '流速传感器', unit: 'L/min' }
];

// 获取传感器阈值
const getSensorThreshold = (sensorType: string) => {
  return sensorThresholds[sensorType] || { max: 0 };
};

// 获取设备的最新告警时间
const getLatestAlarmTime = (deviceId: string): string | undefined => {
  // 从告警列表中查找该设备的最新告警（未解决的）
  const deviceAlerts = alerts.value.filter(alert => {
    // 检查告警的details中是否包含device_id
    if (alert.details && alert.details.device_id === deviceId) {
      return !alert.resolved;
    }
    return false;
  });
  
  if (deviceAlerts.length > 0) {
    // 按创建时间排序，返回最新的
    deviceAlerts.sort((a, b) => {
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
    return deviceAlerts[0].created_at;
  }
  
  return undefined;
};

// 初始化加载
onMounted(() => {
  loadCabinet();
  loadLatestSensorData(); // 初始加载数据
  loadAlerts();
  loadAPIKeyInfo(); // 加载API Key信息

  // 初始化WebSocket连接（实时推送）
  initWebSocket();
});

// 监听告警数据变化，更新传感器卡片的告警时间
watch(() => alerts.value, () => {
  // 告警数据更新时，传感器卡片会自动重新渲染
}, { deep: true });

// 组件卸载时清理资源
onUnmounted(() => {
  closeWebSocket();
  stopPolling();
});

// 加载储能柜详情
async function loadCabinet() {
  const cabinetId = route.params.id as string;
  
  try {
    await cabinetStore.fetchCabinet(cabinetId);
  } catch (error: any) {
    ElMessage.error(error.message || '加载储能柜详情失败');
    // 如果加载失败，返回列表页
    setTimeout(() => {
      router.push('/cabinets');
    }, 2000);
  }
}

// 返回列表
function goBack() {
  router.push('/cabinets');
}

// 加载告警数据
async function loadAlerts() {
  const cabinetId = route.params.id as string;
  alertsLoading.value = true;

  try {
    const response = await cabinetApi.getAlerts(cabinetId, { page: 1, page_size: 5 });
    alerts.value = response.data || [];
  } catch (error: any) {
    console.error('加载告警数据失败:', error);
    alerts.value = [];
  } finally {
    alertsLoading.value = false;
  }
}

// ========== API Key管理方法 ==========
// 加载API Key信息
async function loadAPIKeyInfo() {
  const cabinetId = route.params.id as string;
  if (!cabinetId) return;

  apikeyLoading.value = true;
  try {
    const response = await cabinetApi.getAPIKeyInfo(cabinetId);
    apikeyInfo.value = response.data;
  } catch (error: any) {
    console.error('加载API Key信息失败:', error);
    ElMessage.error(error.message || '加载API Key信息失败');
  } finally {
    apikeyLoading.value = false;
  }
}

// 重新生成API Key
async function handleRegenerateAPIKey() {
  const cabinetId = route.params.id as string;
  if (!cabinetId) return;

  try {
    await ElMessageBox.confirm(
      '重新生成API Key将使旧的凭证失效,请确保已更新Edge端配置。是否继续?',
      '确认重新生成',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    regenerating.value = true;
    const response = await cabinetApi.regenerateAPIKey(cabinetId);
    newCredentials.value = response.data;
    showCredentials.value = true;

    ElMessage.success('API Key生成成功,请妥善保存');

    // 刷新API Key信息
    await loadAPIKeyInfo();
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('生成API Key失败:', error);
      ElMessage.error(error.message || '生成API Key失败');
    }
  } finally {
    regenerating.value = false;
  }
}

// 撤销API Key
async function handleRevokeAPIKey() {
  const cabinetId = route.params.id as string;
  if (!cabinetId) return;

  try {
    await ElMessageBox.confirm(
      '撤销API Key后,Edge端将无法同步数据,直到配置新的API Key。是否继续?',
      '确认撤销',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    revoking.value = true;
    await cabinetApi.revokeAPIKey(cabinetId);

    ElMessage.success('API Key已撤销');

    // 刷新API Key信息
    await loadAPIKeyInfo();
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('撤销API Key失败:', error);
      ElMessage.error(error.message || '撤销API Key失败');
    }
  } finally {
    revoking.value = false;
  }
}

// 复制到剪贴板
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    ElMessage.success('已复制到剪贴板');
  } catch (error) {
    console.error('复制失败:', error);
    ElMessage.error('复制失败,请手动复制');
  }
}

// 获取严重程度类型
function getSeverityType(severity: string): 'danger' | 'warning' | 'info' {
  const map: Record<string, 'danger' | 'warning' | 'info'> = {
    critical: 'danger',
    high: 'danger',
    medium: 'warning',
    low: 'info',
  };
  return map[severity] || 'info';
}

// 获取严重程度标签
function getSeverityLabel(severity: string): string {
  const map: Record<string, string> = {
    critical: '紧急',
    high: '高',
    medium: '中',
    low: '低',
  };
  return map[severity] || severity;
}

// 格式化时间
function formatTime(time: string) {
  return new Date(time).toLocaleString('zh-CN');
}
</script>

<style scoped>
.cabinet-detail {
  padding: 20px;
}

.loading {
  margin-top: 20px;
}

.detail-content {
  margin-top: 20px;
}

.info-card,
.sensor-card,
.alert-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-status {
  display: flex;
  align-items: center;
}

/* 传感器数据样式 */
.sensor-loading,
.alerts-loading {
  padding: 20px;
}

.sensor-data {
  padding: 12px 0;
}

.sensor-item {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 16px;
  text-align: center;
  transition: all 0.3s;
}

.sensor-item:hover {
  background: #e8ecf1;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.sensor-name {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}

.sensor-value {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 8px;
}

.sensor-time {
  font-size: 12px;
  color: #c0c4cc;
}

/* ============== 移动端响应式样式 ============== */
@media (max-width: 768px) {
  .cabinet-detail {
    padding: 12px;
  }

  .loading {
    margin-top: 12px;
  }

  .detail-content {
    margin-top: 12px;
  }

  .info-card,
  .sensor-card,
  .alert-card {
    margin-bottom: 12px;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .header-status {
    flex-wrap: wrap;
    gap: 8px;
  }

  .header-status .el-button {
    margin-left: 0 !important;
  }

  /* 传感器数据移动端优化 */
  .sensor-data {
    padding: 8px 0;
  }

  .sensor-item {
    padding: 12px;
    margin-bottom: 8px;
  }

  .sensor-name {
    font-size: 12px;
  }

  .sensor-value {
    font-size: 20px;
  }

  .sensor-time {
    font-size: 11px;
  }

  /* 描述列表优化 */
  :deep(.el-descriptions__cell) {
    padding: 8px !important;
  }

  :deep(.el-descriptions__label) {
    min-width: 70px !important;
    font-size: 13px;
  }

  :deep(.el-descriptions__content) {
    font-size: 13px;
  }

  /* 告警列表优化 */
  .alert-card :deep(.el-table) {
    font-size: 13px;
  }

  /* 对话框优化 */
  :deep(.el-dialog) {
    width: 92% !important;
    max-width: 92% !important;
    margin: 3vh auto !important;
  }

  :deep(.el-dialog__body) {
    padding: 16px !important;
    max-height: 60vh;
    overflow-y: auto;
  }

  /* API Key 信息优化 */
  :deep(.el-dialog) .el-input-group {
    flex-wrap: wrap;
  }

  :deep(.el-dialog) .el-input-group__append {
    margin-top: 8px;
    width: 100%;
    border-radius: 4px;
  }
}

@media (max-width: 480px) {
  .cabinet-detail {
    padding: 8px;
  }

  .sensor-item {
    padding: 10px;
  }

  .sensor-name {
    font-size: 11px;
  }

  .sensor-value {
    font-size: 18px;
  }

  :deep(.el-descriptions__label) {
    min-width: 60px !important;
    font-size: 12px;
  }

  :deep(.el-descriptions__content) {
    font-size: 12px;
  }

  :deep(.el-dialog) {
    width: 95% !important;
    max-width: 95% !important;
  }

  :deep(.el-dialog__body) {
    padding: 12px !important;
  }
}

</style>
