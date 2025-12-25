<template>
  <div class="health-score">
    <el-progress
      :percentage="score"
      :color="scoreColor"
      :stroke-width="strokeWidth"
      :type="type"
    >
      <template #default="{ percentage }">
        <span class="score-text">{{ percentage }}%</span>
      </template>
    </el-progress>
    <div v-if="showLevel" class="score-level" :style="{ color: scoreColor }">
      {{ scoreLevel }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  score: number;
  type?: 'line' | 'circle' | 'dashboard';
  strokeWidth?: number;
  showLevel?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: 'circle',
  strokeWidth: 6,
  showLevel: true,
});

// 健康评分颜色
const scoreColor = computed(() => {
  const score = props.score;
  if (score >= 90) return '#67c23a'; // 优秀 - 绿色
  if (score >= 75) return '#409eff'; // 良好 - 蓝色
  if (score >= 60) return '#e6a23c'; // 一般 - 橙色
  return '#f56c6c'; // 差 - 红色
});

// 健康评分等级
const scoreLevel = computed(() => {
  const score = props.score;
  if (score >= 90) return 'Excellent';
  if (score >= 75) return 'Good';
  if (score >= 60) return 'Fair';
  return 'Poor';
});
</script>

<style scoped>
.health-score {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.score-text {
  font-size: 16px;
  font-weight: bold;
}

.score-level {
  font-size: 14px;
  font-weight: 500;
}
</style>

