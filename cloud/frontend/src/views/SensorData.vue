<template>
  <div class="sensor-data-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>传感器数据监控</h2>
        </div>
      </template>

      <!-- 查询条件 -->
      <el-form :inline="true" :model="queryForm" class="query-form">
        <el-form-item label="储能柜">
          <el-select
            v-model="queryForm.cabinetId"
            placeholder="选择储能柜"
            clearable
            @change="onCabinetChange"
          >
            <el-option
              v-for="cabinet in cabinets"
              :key="cabinet.cabinet_id"
              :label="cabinet.name"
              :value="cabinet.cabinet_id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="传感器设备">
          <el-select
            v-model="queryForm.deviceId"
            placeholder="选择传感器设备"
            clearable
            :disabled="!queryForm.cabinetId"
            @change="handleDeviceChange"
          >
            <el-option
              v-for="device in sensorDevices"
              :key="device.device_id"
              :label="formatDeviceLabel(device)"
              :value="device.device_id"
              :title="deviceTooltip(device)"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="时间范围">
          <el-date-picker
            v-model="queryForm.timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DDTHH:mm:ss"
          />
        </el-form-item>

        <el-form-item label="聚合间隔">
          <el-select v-model="queryForm.interval" placeholder="选择间隔" clearable>
            <el-option label="原始数据" value="" />
            <el-option label="1分钟" value="1m" />
            <el-option label="5分钟" value="5m" />
            <el-option label="1小时" value="1h" />
            <el-option label="1天" value="1d" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="fetchHistoricalData">查询</el-button>
          <el-button @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 最新数据卡片 -->
    <el-card v-if="queryForm.cabinetId && latestData.length > 0" class="latest-data-card">
      <template #header>
        <h3>实时数据</h3>
      </template>
      <el-row :gutter="16">
        <el-col
          v-for="sensor in latestData"
          :key="sensor.device_id"
          :xs="24"
          :sm="12"
          :md="8"
          :lg="6"
        >
          <el-card shadow="hover" class="sensor-card">
            <div class="sensor-info">
              <div class="sensor-type">{{ sensor.sensor_type }}</div>
              <div class="sensor-value">
                {{ sensor.value.toFixed(2) }} <span class="unit">{{ sensor.unit }}</span>
              </div>
              <div class="sensor-meta">
                <el-tag :type="getStatusTagType(sensor.status)" size="small">
                  {{ sensor.status }}
                </el-tag>
                <span class="quality">质量: {{ sensor.quality }}%</span>
              </div>
              <div class="sensor-time">
                {{ formatTime(sensor.timestamp) }}
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </el-card>

    <!-- 历史数据图表 -->
    <el-card v-if="chartData.length > 0" class="chart-card">
      <template #header>
        <h3>历史趋势</h3>
      </template>
      <div ref="chartRef" style="width: 100%; height: 400px"></div>
    </el-card>

    <!-- 历史数据表格 -->
    <el-card v-if="tableData.length > 0" class="table-card">
      <template #header>
        <h3>历史数据</h3>
      </template>
      <el-table :data="tableData" border stripe>
        <el-table-column prop="timestamp" label="时间" width="180">
          <template #default="{ row }">
            {{ formatTime(row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column prop="value" label="数值" width="120">
          <template #default="{ row }">
            {{ row.value.toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="quality" label="质量" width="100">
          <template #default="{ row }">
            {{ row.quality }}%
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="fetchHistoricalData"
          @current-change="fetchHistoricalData"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, watch } from 'vue';
import { ElMessage } from 'element-plus';
import { sensorApi, cabinetApi } from '@/api';
import type { Cabinet, PaginatedResponse, SensorData, LatestSensorData, SensorDevice } from '@/types/api';
import * as echarts from 'echarts';

// 查询表单
const queryForm = ref({
  cabinetId: '',
  deviceId: '',
  timeRange: [] as string[],
  interval: '',
});

// 数据

interface HistoricalDataParams {
  device_id: string;
  start_time?: string;
  end_time?: string;
  aggregation?: string;
  page?: number;
  page_size?: number;
}

const cabinets = ref<Cabinet[]>([]);
const sensorDevices = ref<SensorDevice[]>([]);
const latestData = ref<LatestSensorData[]>([]);
const chartData = ref<any[]>([]);
const tableData = ref<SensorData[]>([]);

// 分页
const pagination = ref({
  page: 1,
  pageSize: 20,
  total: 0,
});

const DEVICE_PREF_KEY = 'sensor_device_pref'
const devicePreferences = ref<Record<string, string>>({})

const loadDevicePreferences = () => {
  try {
    const stored = localStorage.getItem(DEVICE_PREF_KEY)
    if (stored) {
      devicePreferences.value = JSON.parse(stored)
    }
  } catch {
    devicePreferences.value = {}
  }
}

const saveDevicePreference = (cabinetId: string, deviceId?: string) => {
  if (!cabinetId) return
  if (!deviceId) {
    delete devicePreferences.value[cabinetId]
  } else {
    devicePreferences.value[cabinetId] = deviceId
  }
  localStorage.setItem(DEVICE_PREF_KEY, JSON.stringify(devicePreferences.value))
}

// 图表实例
const chartRef = ref<HTMLElement | null>(null);
let chartInstance: echarts.ECharts | null = null;

// 获取储能柜列表
const fetchCabinets = async () => {
  try {
    const response: PaginatedResponse<Cabinet> = await cabinetApi.list({ page: 1, page_size: 100 });
    cabinets.value = response.data;
  } catch (error: any) {
    ElMessage.error(error.message || '获取储能柜列表失败');
  }
};

// 储能柜变化时，获取传感器设备和最新数据
const onCabinetChange = async () => {
  queryForm.value.deviceId = '';
  sensorDevices.value = [];
  latestData.value = [];

  if (!queryForm.value.cabinetId) return;

  try {
    const [deviceResp, latestResponse] = await Promise.all([
      cabinetApi.getDevices(queryForm.value.cabinetId),
      cabinetApi.getLatestSensorData(queryForm.value.cabinetId),
    ]);

    sensorDevices.value = deviceResp.data || [];
    latestData.value = latestResponse.data || [];

    const preferredDeviceId = devicePreferences.value[queryForm.value.cabinetId];
    const preferredExists = sensorDevices.value.find(device => device.device_id === preferredDeviceId);
    const fallbackDeviceId = sensorDevices.value.length > 0 ? sensorDevices.value[0].device_id : '';
    
    if (preferredExists) {
      queryForm.value.deviceId = preferredDeviceId!;
    } else if (fallbackDeviceId) {
      queryForm.value.deviceId = fallbackDeviceId;
    }

    if (queryForm.value.deviceId) {
      saveDevicePreference(queryForm.value.cabinetId, queryForm.value.deviceId);
      pagination.value.page = 1;
      await fetchHistoricalData();
    }
  } catch (error) {
    ElMessage.error('获取传感器数据失败');
  }
};

const handleDeviceChange = () => {
  if (!queryForm.value.deviceId) {
    if (queryForm.value.cabinetId) {
      saveDevicePreference(queryForm.value.cabinetId, undefined);
    }
    chartData.value = [];
    tableData.value = [];
    return;
  }
  if (queryForm.value.cabinetId) {
    saveDevicePreference(queryForm.value.cabinetId, queryForm.value.deviceId);
  }
  pagination.value.page = 1;
  fetchHistoricalData();
};

// 获取历史数据
const fetchHistoricalData = async () => {
  if (!queryForm.value.deviceId) {
    ElMessage.warning('请选择传感器设备');
    return;
  }

  try {
    const params: HistoricalDataParams = {
      device_id: queryForm.value.deviceId,
      page: pagination.value.page,
      page_size: pagination.value.pageSize,
    };

    if (queryForm.value.timeRange && queryForm.value.timeRange.length === 2) {
      params.start_time = queryForm.value.timeRange[0];
      params.end_time = queryForm.value.timeRange[1];
    }

    if (queryForm.value.interval) {
      params.aggregation = queryForm.value.interval;
    }

    const response = await sensorApi.getHistoricalData(params);

    if (queryForm.value.interval) {
      // 聚合数据
      chartData.value = response.data || [];
      tableData.value = [];
      renderChart();
    } else {
      // 原始数据
      tableData.value = response.data || [];
      pagination.value.total = response.total || 0;
      chartData.value = response.data || [];
      renderChart();
    }
  } catch (error) {
    ElMessage.error('获取历史数据失败');
  }
};

// 重置查询
const resetQuery = () => {
  queryForm.value = {
    cabinetId: '',
    deviceId: '',
    timeRange: [],
    interval: '',
  };
  sensorDevices.value = [];
  latestData.value = [];
  chartData.value = [];
  tableData.value = [];
};

// 渲染图表
const renderChart = async () => {
  await nextTick();
  if (!chartRef.value || chartData.value.length === 0) return;

  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value);
  }

  const xData = chartData.value.map((item: any) => {
    return formatTime(item.timestamp || item.time_bucket);
  });

  const yData = chartData.value.map((item: any) => {
    return item.value || item.avg_value;
  });

  const option = {
    title: {
      text: '传感器数据趋势',
      left: 'center',
    },
    tooltip: {
      trigger: 'axis',
    },
    xAxis: {
      type: 'category',
      data: xData,
      axisLabel: {
        rotate: 45,
      },
    },
    yAxis: {
      type: 'value',
    },
    series: [
      {
        name: '数值',
        type: 'line',
        data: yData,
        smooth: true,
      },
    ],
  };

  chartInstance.setOption(option);
};

// 格式化时间
const formatTime = (timestamp: string) => {
  return new Date(timestamp).toLocaleString('zh-CN');
};

// 获取状态标签类型
const getStatusTagType = (status: string) => {
  const map: Record<string, any> = {
    normal: 'success',
    warning: 'warning',
    error: 'danger',
  };
  return map[status] || 'info';
};

const formatDeviceLabel = (device: SensorDevice) => {
  const base = device.name || device.sensor_type;
  return device.unit ? `${base} (${device.unit})` : base;
};

const deviceTooltip = (device: SensorDevice) => {
  const statusMap: Record<string, string> = { active: '运行', inactive: '停用', error: '异常' };
  const lastTime = device.last_reading_at ? formatTime(device.last_reading_at) : '无';
  const lastValue = device.last_value !== undefined ? device.last_value : '—';
  return `状态: ${statusMap[device.status] || device.status}\n上次上报: ${lastTime}\n上次数值: ${lastValue}`;
};

onMounted(() => {
  loadDevicePreferences();
  fetchCabinets();
});

watch(
  () => queryForm.value.interval,
  () => {
    if (!queryForm.value.deviceId) return;
    pagination.value.page = 1;
    fetchHistoricalData();
  }
);

watch(
  () => (queryForm.value.timeRange || []).join(','),
  () => {
    if (!queryForm.value.deviceId) return;
    pagination.value.page = 1;
    fetchHistoricalData();
  }
);
</script>

<style scoped>
.sensor-data-page {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.query-form {
  margin-top: 20px;
}

.latest-data-card,
.chart-card,
.table-card {
  margin-top: 20px;
}

.sensor-card {
  margin-bottom: 16px;
}

.sensor-info {
  text-align: center;
}

.sensor-type {
  font-size: 14px;
  color: #606266;
  margin-bottom: 8px;
}

.sensor-value {
  font-size: 32px;
  font-weight: bold;
  color: #409eff;
  margin-bottom: 8px;
}

.unit {
  font-size: 16px;
  color: #909399;
  margin-left: 4px;
}

.sensor-meta {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.quality {
  font-size: 12px;
  color: #909399;
}

.sensor-time {
  font-size: 12px;
  color: #c0c4cc;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
