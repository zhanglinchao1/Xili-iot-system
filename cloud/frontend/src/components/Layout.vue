<template>
  <el-container class="layout-container">
    <!-- 侧边栏 - 桌面端 -->
    <el-aside 
      v-if="!isMobile" 
      :width="isCollapse ? '64px' : '200px'" 
      class="layout-aside"
    >
      <div class="logo">
        <span v-if="!isCollapse">Cloud System</span>
        <span v-else>CS</span>
      </div>
      
      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        class="layout-menu"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Monitor /></el-icon>
          <template #title>监控大屏</template>
        </el-menu-item>

        <el-menu-item index="/cabinets">
          <el-icon><Box /></el-icon>
          <template #title>储能柜管理</template>
        </el-menu-item>

        <el-menu-item index="/licenses">
          <el-icon><Key /></el-icon>
          <template #title>许可证管理</template>
        </el-menu-item>

        <el-menu-item index="/vulnerability">
          <el-icon><CircleCheck /></el-icon>
          <template #title>脆弱性评价</template>
        </el-menu-item>

        <el-menu-item index="/traffic">
          <el-icon><Odometer /></el-icon>
          <template #title>流量检测</template>
        </el-menu-item>

        <el-menu-item index="/alerts">
          <el-icon><Bell /></el-icon>
          <template #title>监控告警</template>
        </el-menu-item>

        <el-sub-menu index="/abac" v-if="isAdmin">
          <template #title>
            <el-icon><Lock /></el-icon>
            <span>访问控制</span>
          </template>
          <el-menu-item index="/abac">
            <template #title>策略管理</template>
          </el-menu-item>
          <el-menu-item index="/abac/logs">
            <template #title>访问日志</template>
          </el-menu-item>
          <el-menu-item index="/abac/stats">
            <template #title>访问统计</template>
          </el-menu-item>
        </el-sub-menu>

        <el-menu-item index="/users" v-if="isAdmin">
          <el-icon><User /></el-icon>
          <template #title>用户管理</template>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 移动端抽屉式导航 -->
    <el-drawer
      v-model="mobileMenuVisible"
      direction="ltr"
      :size="260"
      :with-header="false"
      class="mobile-drawer"
    >
      <div class="mobile-menu-container">
        <div class="mobile-logo">
          <span>Cloud System</span>
        </div>
        
        <el-menu
          :default-active="activeMenu"
          router
          class="mobile-menu"
          @select="handleMobileMenuSelect"
        >
          <el-menu-item index="/dashboard">
            <el-icon><Monitor /></el-icon>
            <span>监控大屏</span>
          </el-menu-item>

          <el-menu-item index="/cabinets">
            <el-icon><Box /></el-icon>
            <span>储能柜管理</span>
          </el-menu-item>

          <el-menu-item index="/licenses">
            <el-icon><Key /></el-icon>
            <span>许可证管理</span>
          </el-menu-item>

          <el-menu-item index="/vulnerability">
            <el-icon><CircleCheck /></el-icon>
            <span>脆弱性评价</span>
          </el-menu-item>

          <el-menu-item index="/traffic">
            <el-icon><Odometer /></el-icon>
            <span>流量检测</span>
          </el-menu-item>

          <el-menu-item index="/alerts">
            <el-icon><Bell /></el-icon>
            <span>监控告警</span>
          </el-menu-item>

          <el-sub-menu index="/abac" v-if="isAdmin">
            <template #title>
              <el-icon><Lock /></el-icon>
              <span>访问控制</span>
            </template>
            <el-menu-item index="/abac">策略管理</el-menu-item>
            <el-menu-item index="/abac/logs">访问日志</el-menu-item>
            <el-menu-item index="/abac/stats">访问统计</el-menu-item>
          </el-sub-menu>

          <el-menu-item index="/users" v-if="isAdmin">
            <el-icon><User /></el-icon>
            <span>用户管理</span>
          </el-menu-item>
        </el-menu>
        
        <!-- 移动端用户信息 -->
        <div class="mobile-user-section">
          <div class="mobile-user-info">
            <el-icon><User /></el-icon>
            <span>{{ authStore.user?.username || 'User' }}</span>
          </div>
          <el-button type="danger" text @click="handleMobileLogout">
            <el-icon><SwitchButton /></el-icon>
            退出登录
          </el-button>
        </div>
      </div>
    </el-drawer>

    <!-- 主内容区 -->
    <el-container class="main-container">
      <!-- 顶部导航栏 -->
      <el-header class="layout-header">
        <div class="header-left">
          <!-- 移动端汉堡菜单按钮 -->
          <el-icon 
            v-if="isMobile" 
            class="mobile-menu-icon" 
            @click="toggleMobileMenu"
          >
            <Menu />
          </el-icon>
          
          <!-- 桌面端折叠按钮 -->
          <el-icon 
            v-else 
            class="collapse-icon" 
            @click="toggleCollapse"
          >
            <component :is="isCollapse ? 'Expand' : 'Fold'" />
          </el-icon>
          
          <!-- 面包屑 - 仅桌面端显示 -->
          <el-breadcrumb v-if="!isMobile" separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentRouteMeta.title">
              {{ currentRouteMeta.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
          
          <!-- 移动端页面标题 -->
          <span v-if="isMobile" class="mobile-page-title">
            {{ currentRouteMeta.title || 'Cloud System' }}
          </span>
        </div>
        
        <div class="header-right">
          <!-- 桌面端用户下拉菜单 -->
          <el-dropdown v-if="!isMobile" @command="handleCommand">
            <span class="user-info">
              <el-icon><User /></el-icon>
              <span class="username">{{ authStore.user?.username || 'User' }}</span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">个人信息</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          
          <!-- 移动端用户头像 -->
          <div v-else class="mobile-user-avatar" @click="toggleMobileMenu">
            <el-icon><User /></el-icon>
          </div>
        </div>
      </el-header>

      <!-- 主体内容 -->
      <el-main class="layout-main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/store/auth';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Menu, SwitchButton } from '@element-plus/icons-vue';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

// 侧边栏折叠状态
const isCollapse = ref(false);

// 移动端菜单可见性
const mobileMenuVisible = ref(false);

// 移动端检测断点
const MOBILE_BREAKPOINT = 768;

// 移动端检测 - 立即初始化以避免闪烁
const isMobile = ref(typeof window !== 'undefined' ? window.innerWidth < MOBILE_BREAKPOINT : false);

// 检测屏幕尺寸
function checkMobile() {
  isMobile.value = window.innerWidth < MOBILE_BREAKPOINT;
  // 桌面端时关闭移动端菜单
  if (!isMobile.value) {
    mobileMenuVisible.value = false;
  }
}

// 监听窗口大小变化
onMounted(() => {
  checkMobile();
  window.addEventListener('resize', checkMobile);
});

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile);
});

// 当前激活的菜单
const activeMenu = computed(() => route.path);

// 当前路由元信息
const currentRouteMeta = computed(() => route.meta);

// 是否为管理员
const isAdmin = computed(() => authStore.user?.role === 'admin');

// 切换侧边栏折叠
function toggleCollapse() {
  isCollapse.value = !isCollapse.value;
}

// 切换移动端菜单
function toggleMobileMenu() {
  mobileMenuVisible.value = !mobileMenuVisible.value;
}

// 移动端菜单选择后关闭抽屉
function handleMobileMenuSelect() {
  mobileMenuVisible.value = false;
}

// 移动端退出登录
async function handleMobileLogout() {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    });
    
    mobileMenuVisible.value = false;
    await authStore.logout();
    router.push('/login');
    ElMessage.success('已退出登录');
  } catch (error) {
    // 用户取消
  }
}

// 处理下拉菜单命令
async function handleCommand(command: string) {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      });
      
      await authStore.logout();
      router.push('/login');
      ElMessage.success('已退出登录');
    } catch (error) {
      // 用户取消
    }
  } else if (command === 'profile') {
    router.push('/users');
  }
}
</script>

<style scoped>
.layout-container {
  height: 100%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

/* 侧边栏 - 鲜艳渐变 */
.layout-aside {
  background: linear-gradient(180deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
  box-shadow: 4px 0 20px rgba(102, 126, 234, 0.3);
  transition: width 0.3s;
  position: relative;
  overflow: hidden;
}

.layout-aside::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, transparent 100%);
  pointer-events: none;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  font-weight: 700;
  background: rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(10px);
  letter-spacing: 1px;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  position: relative;
  z-index: 1;
}

/* 菜单样式 - 白色文字 */
.layout-menu {
  border-right: none;
  background: transparent !important;
  position: relative;
  z-index: 1;
}

.layout-menu:not(.el-menu--collapse) {
  width: 200px;
}

/* Element Plus 菜单项覆盖 - 关键修复 */
:deep(.el-menu-item) {
  color: #ffffff !important;
  background: transparent !important;
  border-left: 3px solid transparent;
  transition: all 0.3s;
  font-weight: 500;
  font-size: 16px;
}

:deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.15) !important;
  color: #fff !important;
  border-left-color: #fbbf24;
}

:deep(.el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.25) !important;
  color: #fff !important;
  border-left-color: #fbbf24;
  font-weight: 600;
  box-shadow: inset 0 0 20px rgba(255, 255, 255, 0.1);
}

:deep(.el-menu-item .el-icon) {
  color: #fff !important;
  font-size: 20px;
}

/* 折叠菜单样式 */
:deep(.el-menu--collapse .el-menu-item) {
  padding: 0 !important;
  display: flex;
  justify-content: center;
  align-items: center;
}

/* 子菜单样式 - 与其他菜单一致 */
:deep(.el-sub-menu__title) {
  color: #ffffff !important;
  background: transparent !important;
  border-left: 3px solid transparent;
  transition: all 0.3s;
  font-weight: 500;
  font-size: 16px;
}

:deep(.el-sub-menu__title:hover) {
  background: rgba(255, 255, 255, 0.15) !important;
  color: #fff !important;
  border-left-color: #fbbf24;
}

:deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  color: #fff !important;
  border-left-color: #fbbf24;
}

:deep(.el-sub-menu__title .el-icon) {
  color: #fff !important;
  font-size: 20px;
}

:deep(.el-sub-menu__title .el-sub-menu__icon-arrow) {
  color: #fff !important;
}

/* 子菜单弹出层样式 */
:deep(.el-menu--popup) {
  background: linear-gradient(180deg, #667eea 0%, #764ba2 100%) !important;
  border: none !important;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

:deep(.el-menu--popup .el-menu-item) {
  color: #ffffff !important;
  background: transparent !important;
  font-size: 14px;
}

:deep(.el-menu--popup .el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.2) !important;
}

:deep(.el-menu--popup .el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.25) !important;
  color: #fff !important;
}

/* 内联子菜单样式 - 修复背景色问题 */
:deep(.el-menu--inline) {
  background: transparent !important;
}

:deep(.el-menu--inline .el-menu-item) {
  color: #ffffff !important;
  background: transparent !important;
  font-size: 14px;
  padding-left: 56px !important;
  border-left: 3px solid transparent;
}

:deep(.el-menu--inline .el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.15) !important;
  border-left-color: #fbbf24;
}

:deep(.el-menu--inline .el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.25) !important;
  color: #fff !important;
  border-left-color: #fbbf24;
  font-weight: 600;
}

/* 主容器 */
.main-container {
  flex: 1;
  overflow: hidden;
}

/* 顶部导航栏 - 鲜艳渐变 */
.layout-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: linear-gradient(90deg, #06b6d4 0%, #3b82f6 50%, #8b5cf6 100%);
  border-bottom: none;
  padding: 0 24px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  position: relative;
  overflow: hidden;
  height: 64px;
}

.layout-header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, transparent 100%);
  pointer-events: none;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 24px;
  position: relative;
  z-index: 1;
}

.collapse-icon {
  font-size: 24px;
  cursor: pointer;
  transition: all 0.3s;
  color: #fff;
  padding: 8px;
  border-radius: 8px;
}

.mobile-menu-icon {
  font-size: 32px;
  cursor: pointer;
  transition: all 0.3s;
  color: #fff;
  padding: 10px;
  border-radius: 8px;
  min-width: 44px;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.collapse-icon:hover,
.mobile-menu-icon:hover {
  background: rgba(255, 255, 255, 0.2);
  transform: scale(1.1);
}

/* 移动端页面标题 */
.mobile-page-title {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

/* 面包屑 */
:deep(.el-breadcrumb) {
  font-size: 15px;
}

:deep(.el-breadcrumb__item) {
  color: rgba(255, 255, 255, 0.9);
}

:deep(.el-breadcrumb__item .el-breadcrumb__inner) {
  color: rgba(255, 255, 255, 0.9);
  font-weight: 500;
  transition: all 0.3s;
}

:deep(.el-breadcrumb__item .el-breadcrumb__inner:hover) {
  color: #fff;
}

:deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
  color: #fff;
  font-weight: 600;
}

:deep(.el-breadcrumb__separator) {
  color: rgba(255, 255, 255, 0.6);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 20px;
  position: relative;
  z-index: 1;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 8px 16px;
  border-radius: 10px;
  transition: all 0.3s;
  color: #fff;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.user-info:hover {
  background: rgba(255, 255, 255, 0.25);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.user-info .el-icon {
  font-size: 20px;
}

.username {
  font-size: 15px;
  font-weight: 600;
}

/* 移动端用户头像 */
.mobile-user-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  cursor: pointer;
  transition: all 0.3s;
}

.mobile-user-avatar:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: scale(1.1);
}

/* 主内容区 - 浅色背景 */
.layout-main {
  background: #f8fafc;
  padding: 0;
  overflow-y: auto;
  height: calc(100vh - 64px);
}

/* 路由过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s, transform 0.3s;
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

/* 下拉菜单样式 */
:deep(.el-dropdown-menu__item) {
  padding: 10px 20px;
  font-size: 15px;
}

:deep(.el-dropdown-menu__item:hover) {
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  color: #fff;
}

/* ============== 移动端抽屉式导航样式 ============== */
.mobile-menu-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: linear-gradient(180deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
}

.mobile-logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
  font-weight: 700;
  background: rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(10px);
  letter-spacing: 1px;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

.mobile-menu {
  flex: 1;
  border-right: none;
  background: transparent !important;
  overflow-y: auto;
}

/* 移动端菜单项样式 */
.mobile-menu :deep(.el-menu-item) {
  color: #ffffff !important;
  background: transparent !important;
  height: 50px;
  line-height: 50px;
  font-size: 15px;
  font-weight: 500;
  border-left: 3px solid transparent;
  transition: all 0.3s;
}

.mobile-menu :deep(.el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.15) !important;
  border-left-color: #fbbf24;
}

.mobile-menu :deep(.el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.25) !important;
  color: #fff !important;
  border-left-color: #fbbf24;
  font-weight: 600;
}

.mobile-menu :deep(.el-menu-item .el-icon) {
  color: #fff !important;
  font-size: 18px;
  margin-right: 12px;
}

.mobile-menu :deep(.el-sub-menu__title) {
  color: #ffffff !important;
  background: transparent !important;
  height: 50px;
  line-height: 50px;
  font-size: 15px;
  font-weight: 500;
  border-left: 3px solid transparent;
}

.mobile-menu :deep(.el-sub-menu__title:hover) {
  background: rgba(255, 255, 255, 0.15) !important;
  border-left-color: #fbbf24;
}

.mobile-menu :deep(.el-sub-menu__title .el-icon) {
  color: #fff !important;
  font-size: 18px;
  margin-right: 12px;
}

.mobile-menu :deep(.el-sub-menu__title .el-sub-menu__icon-arrow) {
  color: #fff !important;
}

.mobile-menu :deep(.el-menu--inline) {
  background: transparent !important;
}

.mobile-menu :deep(.el-menu--inline .el-menu-item) {
  padding-left: 50px !important;
  font-size: 14px;
  height: 44px;
  line-height: 44px;
}

/* 移动端用户信息区域 */
.mobile-user-section {
  padding: 16px;
  background: rgba(0, 0, 0, 0.2);
  backdrop-filter: blur(10px);
}

.mobile-user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #fff;
  font-size: 15px;
  font-weight: 500;
  padding: 12px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  margin-bottom: 12px;
}

.mobile-user-info .el-icon {
  font-size: 20px;
}

.mobile-user-section .el-button {
  width: 100%;
  justify-content: center;
  color: #fff !important;
  font-size: 14px;
}

.mobile-user-section .el-button:hover {
  background: rgba(255, 255, 255, 0.1);
}

/* 移动端响应式 */
@media (max-width: 768px) {
  .layout-header {
    padding: 0 12px;
  }

  .header-left {
    gap: 12px;
  }

  .layout-main {
    padding: 0;
  }
}

/* 小屏幕额外优化 */
@media (max-width: 480px) {
  .mobile-page-title {
    font-size: 16px;
    max-width: 150px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>

<!-- 全局样式 - 移动端抽屉覆盖 -->
<style>
/* 移动端抽屉全局样式 */
.mobile-drawer .el-drawer__body {
  padding: 0 !important;
}

.mobile-drawer .el-drawer {
  background: transparent !important;
}
</style>
