import { ref, computed } from 'vue'
import {
  loginUser,
  registerUser,
  logoutUser,
  logoutAllSessions,
  type LoginRequest,
  type RegisterRequest,
  type TokenResponse,
} from '@/api/auth'

export interface UserClaims {
  sub?: string
  username?: string
  email?: string
  role?: string
  sessionUuid?: string
  uuid?: string
  exp?: number
  iat?: number
}

const ACCESS_TOKEN_KEY = 'accessToken'
const REFRESH_TOKEN_KEY = 'refreshToken'
const SESSION_ID_KEY = 'sessionId'

function decodeJwtPayload(token: string): UserClaims | undefined {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) {
      return undefined
    }
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join(''),
    )
    return JSON.parse(jsonPayload) as UserClaims
  } catch {
    return undefined
  }
}

// Global reactive state (single source of truth)
const currentToken = ref<string | undefined>(localStorage.getItem(ACCESS_TOKEN_KEY) || undefined)
const currentUser = ref<UserClaims | undefined>(
  currentToken.value ? decodeJwtPayload(currentToken.value) : undefined,
)
const authError = ref<string | undefined>(undefined)
const isSubmitting = ref(false)

// Sync with global window auth events
if (typeof window !== 'undefined') {
  window.addEventListener('auth:logout', () => {
    currentToken.value = undefined
    currentUser.value = undefined
  })
  window.addEventListener('auth:token-refreshed', (event: Event) => {
    const customEvent = event as CustomEvent<{ accessToken: string }>
    if (customEvent.detail?.accessToken) {
      currentToken.value = customEvent.detail.accessToken
      currentUser.value = decodeJwtPayload(customEvent.detail.accessToken)
    }
  })
}

export function useAuth() {
  const isAuthenticated = computed(() => {
    if (!currentToken.value) {
      return false
    }
    if (currentUser.value?.exp) {
      return currentUser.value.exp * 1000 > Date.now()
    }
    return true
  })

  function setTokens(tokens: TokenResponse) {
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken)
    if (tokens.refreshToken) {
      localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken)
    }
    if (tokens.sessionId) {
      localStorage.setItem(SESSION_ID_KEY, tokens.sessionId)
    }
    currentToken.value = tokens.accessToken
    currentUser.value = decodeJwtPayload(tokens.accessToken)
    if (currentUser.value?.sessionUuid && !tokens.sessionId) {
      localStorage.setItem(SESSION_ID_KEY, currentUser.value.sessionUuid)
    }
    authError.value = undefined
  }

  function clearAuth() {
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    // NOTE: We intentionally preserve SESSION_ID_KEY so subsequent logins from this
    // device reuse the existing session rather than creating duplicate sessions.
    currentToken.value = undefined
    currentUser.value = undefined
  }

  async function login(credentials: LoginRequest): Promise<boolean> {
    isSubmitting.value = true
    authError.value = undefined
    try {
      const storedSessionId = localStorage.getItem(SESSION_ID_KEY) || undefined
      const payload: LoginRequest = {
        ...credentials,
        sessionId: credentials.sessionId || storedSessionId,
      }
      const data = await loginUser(payload)
      if (data?.accessToken) {
        setTokens(data)
        return true
      }
      authError.value = 'Invalid response from server'
      return false
    } catch (err: unknown) {
      clearAuth()
      const axiosError = err as { response?: { data?: string | { error?: string } } }
      const message =
        typeof axiosError.response?.data === 'string'
          ? axiosError.response.data
          : axiosError.response?.data?.error ||
            'Authentication failed. Please check your credentials.'
      authError.value = message
      return false
    } finally {
      isSubmitting.value = false
    }
  }

  async function register(credentials: RegisterRequest): Promise<boolean> {
    isSubmitting.value = true
    authError.value = undefined
    try {
      const data = await registerUser(credentials)
      if (data?.accessToken) {
        setTokens(data)
        return true
      }
      authError.value = 'Invalid response from server'
      return false
    } catch (err: unknown) {
      clearAuth()
      const axiosError = err as { response?: { data?: string | { error?: string } } }
      const message =
        typeof axiosError.response?.data === 'string'
          ? axiosError.response.data
          : axiosError.response?.data?.error ||
            'Registration failed. Please check your details and try again.'
      authError.value = message
      return false
    } finally {
      isSubmitting.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      if (currentToken.value) {
        await logoutUser()
      }
    } catch {
      // Ignore network errors during logout
    } finally {
      clearAuth()
    }
  }

  async function logoutAll(): Promise<void> {
    try {
      if (currentToken.value) {
        await logoutAllSessions()
      }
    } catch {
      // Ignore network errors during logout
    } finally {
      clearAuth()
    }
  }

  function hasRole(requiredRole: string): boolean {
    if (!currentUser.value?.role) {
      return false
    }
    return currentUser.value.role.toLowerCase() === requiredRole.toLowerCase()
  }

  return {
    token: currentToken,
    currentUser,
    isAuthenticated,
    authError,
    isSubmitting,
    login,
    register,
    logout,
    logoutAll,
    hasRole,
    clearAuth,
  }
}
