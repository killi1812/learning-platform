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
  sessionId?: string
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
