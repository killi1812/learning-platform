import axios, { type InternalAxiosRequestConfig } from 'axios'

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

interface QueuedPromise {
  resolve: (token: string) => void
  reject: (err: unknown) => void
}

let isRefreshing = false
let failedQueue: QueuedPromise[] = []

function processQueue(error: unknown, token: string | null = null) {
  failedQueue.forEach(prom => {
    if (error) {
      prom.reject(error)
    } else if (token) {
      prom.resolve(token)
    }
  })
  failedQueue = []
}

function handleAuthFailure() {
  localStorage.removeItem('accessToken')
  localStorage.removeItem('refreshToken')
  window.dispatchEvent(new Event('auth:logout'))

  const currentPath = window.location.pathname
  if (currentPath !== '/login' && currentPath !== '/unauthorized') {
    const redirectParam = encodeURIComponent(currentPath + window.location.search)
    window.location.href = `/login?redirect=${redirectParam}`
  }
}

interface CustomAxiosConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

// Response interceptor: handle 401 with silent token refresh, 403 with access denied
authApi.interceptors.response.use(
  response => response,
  async error => {
    const originalRequest = error.config as CustomAxiosConfig | undefined
    const status = error.response?.status

    if (status === 401 && originalRequest && !originalRequest._retry) {
      const refreshToken = localStorage.getItem('refreshToken')
      if (!refreshToken) {
        handleAuthFailure()
        return Promise.reject(error)
      }

      if (isRefreshing) {
        return new Promise<string>((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        })
          .then(token => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            return authApi(originalRequest)
          })
          .catch(queueErr => Promise.reject(queueErr))
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        const { data } = await publicApi.post<{ accessToken: string; refreshToken?: string }>(
          '/auth/refresh',
          { refreshToken },
        )

        localStorage.setItem('accessToken', data.accessToken)
        if (data.refreshToken) {
          localStorage.setItem('refreshToken', data.refreshToken)
        }

        authApi.defaults.headers.common.Authorization = `Bearer ${data.accessToken}`
        originalRequest.headers.Authorization = `Bearer ${data.accessToken}`

        window.dispatchEvent(
          new CustomEvent('auth:token-refreshed', { detail: { accessToken: data.accessToken } }),
        )

        processQueue(null, data.accessToken)
        return authApi(originalRequest)
      } catch (refreshErr) {
        processQueue(refreshErr, null)
        handleAuthFailure()
        return Promise.reject(refreshErr)
      } finally {
        isRefreshing = false
      }
    }

    if (status === 403) {
      const currentPath = window.location.pathname
      if (currentPath !== '/unauthorized') {
        const redirectParam = encodeURIComponent(currentPath + window.location.search)
        window.location.href = `/unauthorized?redirect=${redirectParam}`
      }
    }

    return Promise.reject(error)
  },
)
