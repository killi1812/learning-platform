import { publicApi, authApi, executeRefreshToken } from './client'
import type {
  LoginRequest,
  RegisterRequest,
  TokenResponse,
  SessionInfo,
  ServerInfoResponse,
} from '@/types/auth'

export type {
  LoginRequest,
  RegisterRequest,
  TokenResponse,
  RefreshRequest,
  SessionInfo,
  ServerInfoResponse,
} from '@/types/auth'

/**
 * Authenticates user credentials without requiring authorization headers
 */
export async function loginUser(credentials: LoginRequest): Promise<TokenResponse> {
  const response = await publicApi.post<TokenResponse>('/auth/login', credentials)
  return response.data
}

/**
 * Registers a new user account without requiring authorization headers
 */
export async function registerUser(credentials: RegisterRequest): Promise<TokenResponse> {
  const response = await publicApi.post<TokenResponse>('/auth/register', credentials)
  return response.data
}

/**
 * Refreshes JWT tokens using the HTTP-only refresh token cookie (or optional fallback token)
 */
export async function refreshUserToken(refreshToken?: string): Promise<TokenResponse> {
  if (refreshToken) {
    const response = await publicApi.post<TokenResponse>('/auth/refresh', { refreshToken })
    return response.data
  }
  return executeRefreshToken()
}

/**
 * Logs out the current device session using authorized API instance
 */
export async function logoutUser(): Promise<void> {
  await authApi.post('/auth/logout')
}

/**
 * Logs out all active user sessions across all devices
 */
export async function logoutAllSessions(): Promise<void> {
  await authApi.post('/auth/logout-all')
}

/**
 * Lists all active user sessions
 */
export async function listUserSessions(): Promise<SessionInfo[]> {
  const response = await authApi.get<SessionInfo[]>('/auth/session')
  return response.data
}

/**
 * Revokes a specific session by ID
 */
export async function revokeUserSession(sessionId: string): Promise<void> {
  await authApi.delete(`/auth/session/${sessionId}`)
}

/**
 * Fetches public server info
 */
export async function getServerInfo(): Promise<ServerInfoResponse> {
  const response = await publicApi.get<ServerInfoResponse>('/info')
  return response.data
}
