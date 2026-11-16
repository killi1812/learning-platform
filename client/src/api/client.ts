import type { TokenResponse } from '@/types/auth'
import axios, { type InternalAxiosRequestConfig, isAxiosError } from 'axios'

export const API_BASE_URL = '/api'
export const ACCESS_TOKEN_STORAGE_KEY = 'accessToken'

interface ApiErrorResponse {
  error: string
}

type AuthFailureHandler = () => void
type TokenRefreshListener = (accessToken: string) => void
type LogoutListener = () => void

let authFailureHandler: AuthFailureHandler | undefined
const tokenRefreshListeners = new Set<TokenRefreshListener>()
const logoutListeners = new Set<LogoutListener>()

export function getStoredAccessToken(): string | undefined {
  if (typeof localStorage === 'undefined') {
    return undefined
  }
  return localStorage.getItem(ACCESS_TOKEN_STORAGE_KEY) || undefined
}

export function setStoredAccessToken(token: string): void {
  if (typeof localStorage === 'undefined') {
    return
  }
  localStorage.setItem(ACCESS_TOKEN_STORAGE_KEY, token)
}

export function clearStoredAccessToken(): void {
  if (typeof localStorage === 'undefined') {
    return
  }
  localStorage.removeItem(ACCESS_TOKEN_STORAGE_KEY)
}

export function registerAuthFailureHandler(handler: AuthFailureHandler): void {
  authFailureHandler = handler
}

export function subscribeAuthEvents(handlers: {
  onLogout?: LogoutListener
  onTokenRefreshed?: TokenRefreshListener
}): () => void {
  if (handlers.onLogout) {
    logoutListeners.add(handlers.onLogout)
  }
  if (handlers.onTokenRefreshed) {
    tokenRefreshListeners.add(handlers.onTokenRefreshed)
  }
  return () => {
    if (handlers.onLogout) {
      logoutListeners.delete(handlers.onLogout)
    }
    if (handlers.onTokenRefreshed) {
      tokenRefreshListeners.delete(handlers.onTokenRefreshed)
    }
  }
}

export class ApiError extends Error {
  readonly status?: number
  readonly isFallback: boolean

  constructor(
    message: string,
    options: { status?: number; cause?: unknown; isFallback?: boolean } = {},
  ) {
    super(message, { cause: options.cause })
    this.name = 'ApiError'
    this.status = options.status
    this.isFallback = options.isFallback ?? false
  }
}

/**
 * Public Axios instance for unauthenticated endpoints (login, registration, public info).
 */
export const publicApi = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30_000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

/**
 * Authorized Axios instance that automatically attaches the JWT bearer token.
 */
export const authApi = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30_000,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

// Request interceptor: automatically add Bearer token to requests
authApi.interceptors.request.use(
  config => {
    const token = getStoredAccessToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    } else {
      delete config.headers.Authorization
    }
    return config
  },
  (error: unknown) => Promise.reject(error),
)

let activeRefreshPromise: Promise<TokenResponse> | undefined

function extractAxiosErrorString(data: unknown): string | undefined {
  if (typeof data === 'string') {
    const trimmed = data.trim()
    if (!trimmed || trimmed.startsWith('<')) {
      return undefined
    }
    return trimmed
  }
  if (
    data &&
    typeof data === 'object' &&
    'error' in data &&
    typeof (data as ApiErrorResponse).error === 'string'
  ) {
    const trimmed = (data as ApiErrorResponse).error.trim()
    return trimmed.length > 0 ? trimmed : undefined
  }
  return undefined
}

async function extractBlobErrorMessage(blob: Blob): Promise<string | undefined> {
  if (blob.size > 65_536) {
    return undefined
  }
  if (blob.type && !blob.type.includes('json') && !blob.type.includes('text')) {
    return undefined
  }
  try {
    const text = await blob.text()
    const parsed: unknown = JSON.parse(text)
    return extractAxiosErrorString(parsed)
  } catch {
    return undefined
  }
}

/**
 * Safely extracts a user-friendly error message from an API error or network failure.
 */
export function getApiErrorMessage(
  err: unknown,
  fallback = 'An unexpected error occurred. Please try again.',
): string {
  if (err instanceof ApiError) {
    return err.isFallback ? fallback : err.message
  }
  if (isAxiosError(err)) {
    const serverMessage = extractAxiosErrorString(err.response?.data)
    return serverMessage ?? fallback
  }
  if (err instanceof Error && err.message) {
    return err.message
  }
  return fallback
}

/**
 * Unwraps an Axios request and normalizes errors into `ApiError` (preserving status & Blob messages).
 */
export async function request<T>(
  promise: Promise<{ data: T }>,
  fallback = 'An unexpected error occurred. Please try again.',
): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (error: unknown) {
    if (error instanceof ApiError) {
      throw error
    }
    if (isAxiosError(error)) {
      const status = error.response?.status
      let serverMessage = extractAxiosErrorString(error.response?.data)
      if (!serverMessage && error.response?.data instanceof Blob) {
        serverMessage = await extractBlobErrorMessage(error.response.data)
      }
      if (serverMessage) {
        throw new ApiError(serverMessage, { status, cause: error, isFallback: false })
      }
      throw new ApiError(fallback, { status, cause: error, isFallback: true })
    }
    if (error instanceof Error && error.message) {
      throw new ApiError(error.message, { cause: error, isFallback: false })
    }
    throw new ApiError(fallback, { cause: error, isFallback: true })
  }
}

/**
 * Deduplicated token refresh executor. Guarantees that only a single request
 * to /auth/refresh is in flight at any given time.
 */
export async function executeRefreshToken(): Promise<TokenResponse> {
  if (activeRefreshPromise) {
    return activeRefreshPromise
  }

  activeRefreshPromise = (async () => {
    try {
      const data = await request(publicApi.post<TokenResponse>('/auth/refresh'))
      setStoredAccessToken(data.accessToken)
      for (const listener of tokenRefreshListeners) {
        listener(data.accessToken)
      }
      return data
    } finally {
      activeRefreshPromise = undefined
    }
  })()

  return activeRefreshPromise
}

function handleAuthFailure(): void {
  clearStoredAccessToken()
  for (const listener of logoutListeners) {
    listener()
  }

  if (authFailureHandler) {
    authFailureHandler()
    return
  }

  if (typeof window !== 'undefined') {
    const currentPath = window.location.pathname
    if (currentPath !== '/login' && currentPath !== '/unauthorized') {
      const redirectParam = encodeURIComponent(currentPath + window.location.search)
      window.location.href = `/login?redirect=${redirectParam}`
    }
  }
}

interface CustomAxiosConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

// Response interceptor: handle 401 with silent token refresh
authApi.interceptors.response.use(
  response => response,
  async (error: unknown) => {
    if (!isAxiosError(error)) {
      throw error
    }

    const originalRequest = error.config as CustomAxiosConfig | undefined
    const status = error.response?.status

    if (status === 401 && originalRequest && !originalRequest._retry) {
      originalRequest._retry = true
      try {
        const data = await executeRefreshToken()
        originalRequest.headers.Authorization = `Bearer ${data.accessToken}`
        return authApi(originalRequest)
      } catch (refreshError: unknown) {
        handleAuthFailure()
        throw refreshError
      }
    }

    throw error
  },
)
