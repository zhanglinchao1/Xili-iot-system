<template>
  <el-tag :type="statusType" :size="size">
    {{ statusText }}
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  status: string;
  type?: 'cabinet' | 'device' | 'license' | 'command' | 'alert';
  size?: 'large' | 'default' | 'small';
}

const props = withDefaults(defineProps<Props>(), {
  type: 'cabinet',
  size: 'default',
});

// 状态类型映射
// 统一使用数据库定义的状态: pending, active, inactive, offline, maintenance
const statusTypeMap: Record<string, Record<string, 'success' | 'info' | 'warning' | 'danger'>> = {
  cabinet: {
    pending: 'info',        // 待激活
    active: 'success',      // 在线同步中
    inactive: 'danger',     // 已停用
    offline: 'warning',     // 离线
    maintenance: 'warning', // 维护中
  },
  device: {
    active: 'success',
    inactive: 'info',
    error: 'danger',
  },
  license: {
    active: 'success',
    expired: 'danger',
    revoked: 'warning',
  },
  command: {
    pending: 'info',
    sent: 'warning',
    completed: 'success',
    failed: 'danger',
    timeout: 'danger',
  },
  alert: {
    info: 'info',
    warning: 'warning',
    error: 'danger',
    critical: 'danger',
  },
};

// 状态文本映射
const statusTextMap: Record<string, Record<string, string>> = {
  cabinet: {
    pending: '待激活',      // 待激活
    active: '在线同步中',   // 激活且同步中
    inactive: '已停用',     // 已停用
    offline: '离线',        // 离线
    maintenance: '维护中',  // 维护中
  },
  device: {
    active: '活跃',
    inactive: '未激活',
    error: '故障',
  },
  license: {
    active: '有效',
    expired: '已过期',
    revoked: '已吊销',
  },
  command: {
    pending: '等待中',
    sent: '已发送',
    completed: '已完成',
    failed: '失败',
    timeout: '超时',
  },
  alert: {
    info: '信息',
    warning: '警告',
    error: '错误',
    critical: '严重',
  },
};

const statusType = computed(() => {
  return statusTypeMap[props.type]?.[props.status] || 'info';
});

const statusText = computed(() => {
  return statusTextMap[props.type]?.[props.status] || props.status;
});
</script>
