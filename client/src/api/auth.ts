import type {
  LoginRequest,
  RegisterRequest,
  ServerInfoResponse,
  SessionInfo,
  TokenResponse,
} from '@/types/auth'
import { authApi, publicApi, request } from './client'

/**
 * Authenticates user credentials without requiring authorization headers.
 */
export function loginUser(credentials: LoginRequest): Promise<TokenResponse> {
  return request(publicApi.post<TokenResponse>('/auth/login', credentials))
}

/**
 * Registers a new user account without requiring authorization headers.
 */
export function registerUser(credentials: RegisterRequest): Promise<TokenResponse> {
  return request(publicApi.post<TokenResponse>('/auth/register', credentials))
}

/**
 * Logs out the current device session using authorized API instance.
 */
export async function logoutUser(): Promise<void> {
  await request(authApi.post('/auth/logout'))
}

/**
 * Logs out all active user sessions across all devices.
 */
export async function logoutAllSessions(): Promise<void> {
  await request(authApi.post('/auth/logout-all'))
}

/**
 * Lists all active user sessions.
 */
export function listUserSessions(): Promise<SessionInfo[]> {
  return request(authApi.get<SessionInfo[]>('/auth/session'))
}

/**
 * Revokes a specific session by ID.
 */
export async function revokeUserSession(sessionId: string): Promise<void> {
  await request(authApi.delete(`/auth/session/${encodeURIComponent(sessionId)}`))
}

/**
 * Fetches public server info.
 */
export function getServerInfo(): Promise<ServerInfoResponse> {
  return request(publicApi.get<ServerInfoResponse>('/info'))
}
