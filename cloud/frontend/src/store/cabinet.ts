/**
 * 储能柜Store
 * 管理储能柜数据的状态和操作
 */

import { defineStore } from 'pinia';
import { ref } from 'vue';
import { cabinetApi } from '@/api';
import type { Cabinet, PaginatedResponse } from '@/types/api';

export const useCabinetStore = defineStore('cabinet', () => {
  const MAX_PAGE_SIZE = 100; // backend enforces this limit

  // 状态
  const cabinets = ref<Cabinet[]>([]);
  const currentCabinet = ref<Cabinet | null>(null);
  const loading = ref(false);
  const total = ref(0);
  const currentPage = ref(1);
  const pageSize = ref(20);

  // 获取储能柜列表
  async function fetchCabinets(params?: {
    status?: string;
    location?: string;
    page?: number;
    page_size?: number;
  }) {
    loading.value = true;
    try {
      const requestedPageSize = params?.page_size ?? pageSize.value ?? MAX_PAGE_SIZE;
      const response: PaginatedResponse<Cabinet> = await cabinetApi.list({
        ...params,
        page_size: Math.min(requestedPageSize, MAX_PAGE_SIZE),
      });
      cabinets.value = response.data;
      total.value = response.total;
      currentPage.value = response.page;
      pageSize.value = Math.min(response.page_size || MAX_PAGE_SIZE, MAX_PAGE_SIZE);
    } catch (error: any) {
      console.error('Failed to fetch cabinets:', error);
      throw error;
    } finally {
      loading.value = false;
    }
  }

  // 获取储能柜详情
  async function fetchCabinet(cabinetId: string) {
    loading.value = true;
    try {
      const response = await cabinetApi.get(cabinetId);
      currentCabinet.value = response.data;
      return response.data;
    } catch (error: any) {
      console.error('Failed to fetch cabinet:', error);
      throw error;
    } finally {
      loading.value = false;
    }
  }

  // 创建储能柜
  async function createCabinet(data: Partial<Cabinet>) {
    loading.value = true;
    try {
      const response = await cabinetApi.create(data);
      // 刷新列表
      await fetchCabinets({ page: currentPage.value, page_size: pageSize.value });
      return response.data;
    } catch (error: any) {
      console.error('Failed to create cabinet:', error);
      throw error;
    } finally {
      loading.value = false;
    }
  }

  // 更新储能柜
  async function updateCabinet(cabinetId: string, data: Partial<Cabinet>) {
    loading.value = true;
    try {
      await cabinetApi.update(cabinetId, data);
      // 刷新列表
      await fetchCabinets({ page: currentPage.value, page_size: pageSize.value });
      // 如果当前详情是这个柜子，刷新详情
      if (currentCabinet.value?.cabinet_id === cabinetId) {
        await fetchCabinet(cabinetId);
      }
    } catch (error: any) {
      console.error('Failed to update cabinet:', error);
      throw error;
    } finally {
      loading.value = false;
    }
  }

  // 删除储能柜
  async function deleteCabinet(cabinetId: string) {
    loading.value = true;
    try {
      await cabinetApi.delete(cabinetId);
      // 刷新列表
      await fetchCabinets({ page: currentPage.value, page_size: pageSize.value });
    } catch (error: any) {
      console.error('Failed to delete cabinet:', error);
      throw error;
    } finally {
      loading.value = false;
    }
  }

  // 清除当前储能柜
  function clearCurrentCabinet() {
    currentCabinet.value = null;
  }

  return {
    // 状态
    cabinets,
    currentCabinet,
    loading,
    total,
    currentPage,
    pageSize,

    // 方法
    fetchCabinets,
    fetchCabinet,
    createCabinet,
    updateCabinet,
    deleteCabinet,
    clearCurrentCabinet,
  };
});
