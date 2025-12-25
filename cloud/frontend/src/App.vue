<template>
  <router-view />
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { useConfigStore } from '@/store/config';

const configStore = useConfigStore();

// 应用启动时加载配置
onMounted(() => {
  configStore.loadConfig();
});
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body,
#app {
  width: 100%;
  height: 100%;
  font-family: 'Times New Roman', Times, serif;
}

#app {
  background-color: #f5f7fa;
}
</style>
