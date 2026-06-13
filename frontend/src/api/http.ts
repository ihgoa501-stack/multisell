import axios from 'axios'

// 创建 axios 实例
const http = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器 - 自动带token
http.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
http.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== undefined && res.code !== 200) {
      const msg = res.message || '请求失败'
      console.error('[API Error]', msg)
      return Promise.reject(new Error(msg))
    }
    return res
  },
  (error) => {
    // 401 → 跳转登录页
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
      return Promise.reject(new Error('登录已过期，请重新登录'))
    }
    const msg = error.response?.data?.message || error.message || '网络错误'
    console.error('[HTTP Error]', msg)
    return Promise.reject(new Error(msg))
  }
)

export default http
