import { publicApi, authApi } from './client'

export interface LoginRequest {
  username: string
  password: string
  email?: string
  sessionId?: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface TokenResponse {
  accessToken: string
  refreshToken?: string
  sessionId?: string
}

export interface RefreshRequest {
  refreshToken: string
}

export interface SessionInfo {
  id: string
  deviceName: string
  createdAt: string
  expiresAt: string
  isCurrent: boolean
}

export interface ServerInfoResponse {
  build: string
  version: string
  commitHash: string
  buildTimestamp: string
}

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
 * Refreshes JWT tokens using a refresh token
 */
export async function refreshUserToken(refreshToken: string): Promise<TokenResponse> {
  const response = await publicApi.post<TokenResponse>('/auth/refresh', {
    refreshToken,
  })
  return response.data
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
  const response = await authApi.get<SessionInfo[]>('/auth/sessions')
  return response.data
}

/**
 * Revokes a specific session by ID
 */
export async function revokeUserSession(sessionId: string): Promise<void> {
  await authApi.delete(`/auth/sessions/${sessionId}`)
}

/**
 * Fetches public server info
 */
export async function getServerInfo(): Promise<ServerInfoResponse> {
  const response = await publicApi.get<ServerInfoResponse>('/info')
  return response.data
}
