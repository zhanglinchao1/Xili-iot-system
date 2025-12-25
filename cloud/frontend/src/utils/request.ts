/**
 * Axios请求封装
 * 统一处理请求/响应拦截、错误处理、认证Token
 */

import axios from 'axios';
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse, AxiosError } from 'axios';
import type { ErrorResponse } from '@/types/api';

// 创建axios实例
// 优先使用环境变量，如果没有则使用相对路径（通过nginx代理）
const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const service: AxiosInstance = axios.create({
  baseURL: apiBaseUrl,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
service.interceptors.request.use(
  (config) => {
    // 从localStorage获取token
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error: AxiosError) => {
    console.error('Request error:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    const data = response.data;
    
    // 成功响应，直接返回data
    if (data.success !== undefined) {
      return data;
    }
    
    // 兼容其他格式
    return data;
  },
  (error: AxiosError<ErrorResponse>) => {
    // 处理错误响应
    if (error.response) {
      const { status, data } = error.response;

      // 401未授权 - 跳转登录 (但不抛出错误，避免显示错误消息)
      if (status === 401) {
        console.warn('未授权，跳转到登录页面');
        localStorage.removeItem('token');
        // 延迟跳转，避免影响当前操作
        setTimeout(() => {
          window.location.href = '/login';
        }, 100);
        // 返回一个明确的401错误，让调用者知道需要登录
        return Promise.reject({
          code: 'UNAUTHORIZED',
          message: '未登录或登录已过期，请重新登录',
          status: 401,
        });
      }

      // 403禁止访问 - 权限不足（静默处理，不输出大量错误）
      if (status === 403) {
        // 使用 warn 级别而不是 error 级别，减少控制台噪音
        console.warn('权限不足：当前用户没有访问此资源的权限');
        return Promise.reject({
          code: 'FORBIDDEN',
          message: '权限不足，您没有访问此功能的权限',
          status: 403,
        });
      }

      // 提取错误信息
      const errorMessage = (data?.error as any)?.message || (data as any)?.message || '请求失败';
      const errorCode = (data?.error as any)?.code || (data as any)?.error || 'UNKNOWN_ERROR';

      console.error(`API Error [${errorCode}]:`, errorMessage, 'Status:', status);

      // 返回格式化的错误
      return Promise.reject({
        code: errorCode,
        message: errorMessage,
        details: data?.error?.details,
        status,
      });
    } else if (error.request) {
      // 请求已发送但没有收到响应
      console.error('Network error:', error.message);
      return Promise.reject({
        code: 'NETWORK_ERROR',
        message: '网络错误，请检查网络连接',
      });
    } else {
      // 其他错误
      console.error('Error:', error.message);
      return Promise.reject({
        code: 'REQUEST_ERROR',
        message: error.message || '请求失败',
      });
    }
  }
);

// 封装请求方法
export const request = {
  get<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    // 添加时间戳防止缓存
    const timestamp = new Date().getTime();
    const separator = url.includes('?') ? '&' : '?';
    const urlWithTimestamp = `${url}${separator}_t=${timestamp}`;
    return service.get(urlWithTimestamp, config);
  },
  
  post<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    return service.post(url, data, config);
  },
  
  put<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    return service.put(url, data, config);
  },
  
  delete<T = any>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return service.delete(url, config);
  },
  
  patch<T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
    return service.patch(url, data, config);
  },
};

export default service;

