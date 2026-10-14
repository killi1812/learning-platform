import { publicApi, authApi } from './client'

export interface LoginRequest {
  username: string
  password: string
  email?: string
}

export interface TokenResponse {
  accessToken: string
  refreshToken?: string
}

export interface RefreshRequest {
  refreshToken: string
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
 * Refreshes JWT tokens using a refresh token
 */
export async function refreshUserToken(refreshToken: string): Promise<TokenResponse> {
  const response = await publicApi.post<TokenResponse>('/auth/refresh', {
    refreshToken,
  })
  return response.data
}

/**
 * Logs out the current session using authorized API instance
 */
export async function logoutUser(): Promise<void> {
  await authApi.post('/auth/logout')
}

/**
 * Fetches public server info
 */
export async function getServerInfo(): Promise<ServerInfoResponse> {
  const response = await publicApi.get<ServerInfoResponse>('/info')
  return response.data
}
