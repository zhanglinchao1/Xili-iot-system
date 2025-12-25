/**
 * Vue Router配置
 */

import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '@/store/auth';

// 路由配置
const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/Register.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('@/components/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '监控大屏' },
      },
      {
        path: 'cabinets',
        name: 'CabinetList',
        component: () => import('@/views/CabinetList.vue'),
        meta: { title: '储能柜列表' },
      },
      {
        path: 'cabinets/create',
        name: 'CabinetCreate',
        component: () => import('@/views/CabinetCreate.vue'),
        meta: { title: '预注册储能柜' },
      },
      {
        path: 'cabinets/:id',
        name: 'CabinetDetail',
        component: () => import('@/views/CabinetDetail.vue'),
        meta: { title: '储能柜详情' },
      },
      {
        path: 'licenses',
        name: 'LicenseManage',
        component: () => import('@/views/LicenseManage.vue'),
        meta: { title: '许可控制' },
      },
      {
        path: 'vulnerability',
        name: 'Vulnerability',
        component: () => import('@/views/Vulnerability.vue'),
        meta: { title: '脆弱性评价' },
      },
      {
        path: 'vulnerability/:id',
        name: 'VulnerabilityDetail',
        component: () => import('@/views/VulnerabilityDetail.vue'),
        meta: { title: '脆弱性详情' },
      },
      {
        path: 'traffic',
        name: 'Traffic',
        component: () => import('@/views/Traffic.vue'),
        meta: { title: '流量检测' },
      },
      {
        path: 'traffic/:id',
        name: 'TrafficDetail',
        component: () => import('@/views/TrafficDetail.vue'),
        meta: { title: '流量详情' },
      },
      {
        path: 'alerts',
        name: 'AlertManage',
        component: () => import('@/views/AlertManage.vue'),
        meta: { title: '监控告警' },
      },
      {
        path: 'users',
        name: 'UserManage',
        component: () => import('@/views/UserManage.vue'),
        meta: { title: '用户管理', requiresAdmin: true },
      },
      {
        path: 'abac',
        name: 'ABACManage',
        component: () => import('@/views/ABACManage.vue'),
        meta: { title: '策略管理', requiresAdmin: true },
      },
      {
        path: 'abac/logs',
        name: 'ABACLogs',
        component: () => import('@/views/ABACLogs.vue'),
        meta: { title: '访问日志', requiresAdmin: true },
      },
      {
        path: 'abac/stats',
        name: 'ABACStats',
        component: () => import('@/views/ABACStats.vue'),
        meta: { title: '访问统计', requiresAdmin: true },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue'),
  },
];

// 创建路由实例
const router = createRouter({
  history: createWebHistory(),
  routes,
});

// 路由守卫
router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore();
  const requiresAuth = to.meta.requiresAuth !== false;
  const requiresAdmin = to.meta.requiresAdmin === true;

  // 设置页面标题
  if (to.meta.title) {
    document.title = `${to.meta.title} - Cloud端储能柜管理系统`;
  } else {
    document.title = 'Cloud端储能柜管理系统';
  }

  // 检查认证
  if (requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'Login', query: { redirect: to.fullPath } });
  } else if (to.name === 'Login' && authStore.isAuthenticated) {
    next({ name: 'Dashboard' });
  } else if (requiresAdmin && !authStore.isAdmin) {
    // 非管理员访问管理员页面，重定向到首页
    console.warn('非管理员用户尝试访问管理员页面，已重定向到首页');
    next({ name: 'Dashboard' });
  } else {
    next();
  }
});

export default router;

