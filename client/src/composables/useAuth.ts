import type { LoginRequest, RegisterRequest, TokenResponse, UserClaims } from '@/types/auth'
import { computed, readonly, ref } from 'vue'
import { loginUser, logoutAllSessions, logoutUser, registerUser } from '@/api/auth'
import {
  ACCESS_TOKEN_STORAGE_KEY,
  clearStoredAccessToken,
  executeRefreshToken,
  getApiErrorMessage,
  getStoredAccessToken,
  setStoredAccessToken,
  subscribeAuthEvents,
} from '@/api/client'
import { resetNotificationSync } from '@/composables/useNotifications'

const SESSION_ID_KEY = 'sessionId'
const MAX_TIMEOUT_MS = 2_147_483_647

function decodeJwtPayload(token: string): UserClaims | undefined {
  try {
    const parts = token.split('.')
    if (parts.length !== 3 || !parts[1]) {
      return undefined
    }
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')
    const binary = atob(padded)
    const bytes = Uint8Array.from(binary, char => char.codePointAt(0) ?? 0)
    const parsed: unknown = JSON.parse(new TextDecoder().decode(bytes))
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      return undefined
    }
    const record = parsed as Record<string, unknown>
    return {
      sub: typeof record.sub === 'string' ? record.sub : undefined,
      jti: typeof record.jti === 'string' ? record.jti : undefined,
      userUuid: typeof record.userUuid === 'string' ? record.userUuid : undefined,
      username: typeof record.username === 'string' ? record.username : undefined,
      email: typeof record.email === 'string' ? record.email : undefined,
      role: typeof record.role === 'string' ? record.role : undefined,
      sessionUuid: typeof record.sessionUuid === 'string' ? record.sessionUuid : undefined,
      exp: typeof record.exp === 'number' ? record.exp : undefined,
      iat: typeof record.iat === 'number' ? record.iat : undefined,
    }
  } catch {
    return undefined
  }
}

// Global reactive state (single source of truth)
const initialToken = getStoredAccessToken()
const currentToken = ref<string | undefined>(initialToken)
const currentUser = ref<UserClaims | undefined>(
  initialToken ? decodeJwtPayload(initialToken) : undefined,
)
const nowMs = ref(Date.now())
const authError = ref<string | undefined>(undefined)
const isSubmitting = ref(false)
let expiryTimer: number | undefined

function scheduleExpiryCheck(claims: UserClaims | undefined): void {
  if (typeof window === 'undefined') {
    return
  }
  if (expiryTimer !== undefined) {
    window.clearTimeout(expiryTimer)
    expiryTimer = undefined
  }
  if (!claims?.exp) {
    return
  }
  const delayMs = claims.exp * 1000 - Date.now()
  if (delayMs <= 0) {
    nowMs.value = Date.now()
    return
  }
  const boundedDelay = Math.min(delayMs, MAX_TIMEOUT_MS)
  expiryTimer = window.setTimeout(() => {
    nowMs.value = Date.now()
    if (currentUser.value && delayMs > MAX_TIMEOUT_MS) {
      scheduleExpiryCheck(currentUser.value)
    }
  }, boundedDelay)
}

function applyToken(token: string | undefined): void {
  currentToken.value = token
  currentUser.value = token ? decodeJwtPayload(token) : undefined
  nowMs.value = Date.now()
  scheduleExpiryCheck(currentUser.value)
  if (!token) {
    resetNotificationSync()
  }
}

if (currentUser.value) {
  scheduleExpiryCheck(currentUser.value)
}

subscribeAuthEvents({
  onLogout: () => applyToken(undefined),
  onTokenRefreshed: token => applyToken(token),
})

if (typeof window !== 'undefined') {
  window.addEventListener('storage', (event: StorageEvent) => {
    if (event.key === ACCESS_TOKEN_STORAGE_KEY) {
      applyToken(event.newValue || undefined)
    }
  })
}

export function useAuth() {
  function isTokenExpired(): boolean {
    if (!currentToken.value || !currentUser.value) {
      return true
    }
    if (currentUser.value.exp !== undefined) {
      return currentUser.value.exp * 1000 <= Date.now()
    }
    return false
  }

  const isAuthenticated = computed(() => {
    if (!currentToken.value || !currentUser.value) {
      return false
    }
    if (currentUser.value.exp !== undefined) {
      return currentUser.value.exp * 1000 > nowMs.value
    }
    return true
  })

  function clearAuthError(): void {
    authError.value = undefined
  }

  function setTokens(tokens: TokenResponse): void {
    setStoredAccessToken(tokens.accessToken)
    applyToken(tokens.accessToken)
    if (currentUser.value?.sessionUuid && typeof localStorage !== 'undefined') {
      localStorage.setItem(SESSION_ID_KEY, currentUser.value.sessionUuid)
    }
    authError.value = undefined
  }

  function clearAuth(): void {
    clearStoredAccessToken()
    // NOTE: SESSION_ID_KEY is intentionally preserved so subsequent logins from this
    // device reuse the existing session rather than creating duplicate sessions.
    applyToken(undefined)
  }

  async function login(credentials: LoginRequest): Promise<boolean> {
    if (isSubmitting.value) {
      return false
    }
    isSubmitting.value = true
    authError.value = undefined
    try {
      const storedSessionId =
        typeof localStorage === 'undefined'
          ? undefined
          : localStorage.getItem(SESSION_ID_KEY) || undefined
      const payload: LoginRequest = {
        email: credentials.email,
        password: credentials.password,
        sessionId: credentials.sessionId || storedSessionId,
      }
      const data = await loginUser(payload)
      if (data?.accessToken) {
        setTokens(data)
        return true
      }
      authError.value = 'Invalid response from server'
      return false
    } catch (error: unknown) {
      clearAuth()
      authError.value = getApiErrorMessage(
        error,
        'Authentication failed. Please check your credentials.',
      )
      return false
    } finally {
      isSubmitting.value = false
    }
  }

  async function register(credentials: RegisterRequest): Promise<boolean> {
    if (isSubmitting.value) {
      return false
    }
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
    } catch (error: unknown) {
      clearAuth()
      authError.value = getApiErrorMessage(
        error,
        'Registration failed. Please check your details and try again.',
      )
      return false
    } finally {
      isSubmitting.value = false
    }
  }

  async function logout(): Promise<void> {
    try {
      if (currentToken.value && !isTokenExpired()) {
        await logoutUser()
      }
    } catch {
      // Best-effort server session revocation; always clear local state afterward.
    } finally {
      clearAuth()
    }
  }

  async function logoutAll(): Promise<void> {
    if (currentToken.value) {
      await logoutAllSessions()
    }
    clearAuth()
  }

  async function refresh(): Promise<boolean> {
    try {
      const data = await executeRefreshToken()
      if (data?.accessToken) {
        setTokens(data)
        return true
      }
      clearAuth()
      return false
    } catch {
      clearAuth()
      return false
    }
  }

  function hasRole(requiredRole: string): boolean {
    const role = currentUser.value?.role
    if (typeof role !== 'string') {
      return false
    }
    return role.toLowerCase() === requiredRole.toLowerCase()
  }

  function hasAnyRole(roles: readonly string[]): boolean {
    return roles.some(role => hasRole(role))
  }

  const isAdminOrSuperadmin = computed(() => hasAnyRole(['admin', 'superadmin']))

  return {
    token: readonly(currentToken),
    currentUser: readonly(currentUser),
    isAuthenticated,
    isAdminOrSuperadmin,
    authError: readonly(authError),
    isSubmitting: readonly(isSubmitting),
    clearAuthError,
    isTokenExpired,
    login,
    register,
    refresh,
    logout,
    logoutAll,
    hasRole,
    hasAnyRole,
    clearAuth,
  }
}
