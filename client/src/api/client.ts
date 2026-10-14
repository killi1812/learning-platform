import axios from 'axios'

export const API_BASE_URL = '/api'

/**
 * Public Axios instance for unauthenticated endpoints (login, registration, public info)
 */
export const publicApi = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

/**
 * Authorized Axios instance that automatically attaches the JWT bearer token
 */
export const authApi = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: automatically add Bearer token to requests
authApi.interceptors.request.use(
  config => {
    const token = localStorage.getItem('accessToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => Promise.reject(error),
)

// Response interceptor: handle 401 Unauthorized / 403 Forbidden globally
authApi.interceptors.response.use(
  response => response,
  error => {
    const status = error.response?.status
    if (status === 401 || status === 403) {
      // Clear expired credentials
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')

      const currentPath = window.location.pathname
      if (currentPath !== '/login' && currentPath !== '/unauthorized') {
        const redirectParam = encodeURIComponent(currentPath + window.location.search)
        window.location.href = `/unauthorized?redirect=${redirectParam}`
      }
    }
    return Promise.reject(error)
  },
)
