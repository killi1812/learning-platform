export interface LoginRequest {
  email: string
  password: string
  sessionId?: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface TokenResponse {
  accessToken: string
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
