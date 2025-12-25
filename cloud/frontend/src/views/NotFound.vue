<template>
  <div class="not-found">
    <el-result icon="warning" title="404" sub-title="Page not found">
      <template #extra>
        <el-button type="primary" @click="goHome">Back to Home</el-button>
      </template>
    </el-result>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';

const router = useRouter();

function goHome() {
  router.push('/');
}
</script>

<style scoped>
.not-found {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}
</style>

