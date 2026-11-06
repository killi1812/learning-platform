export type PayloadType = 'URL' | 'FILE'

export interface UploadTicket {
  ticketUuid: string
  submittedAt: string
  expiresAt: string
}

export interface SubmissionAccepted {
  uuid: string
  submittedAt: string
  status: string
}

export interface Submission {
  uuid: string
  attempt: number
  isLate: boolean
  payloadType: PayloadType
  payloadUrl?: string
  fileName?: string
  fileSize?: number
  contentType?: string
  submittedAt: string
  userEmail?: string
}

export interface CreateHandinPayload {
  title: string
  deadline?: string
  acceptsLateSubmissions: boolean
}

export interface UpdateHandinPayload {
  title: string
  deadline?: string
  acceptsLateSubmissions: boolean
}

export interface HandinAssignment {
  uuid: string
  moduleUuid?: string
  title: string
  position: number
  deadline?: string
  acceptsLateSubmissions: boolean
  createdAt: string
  updatedAt: string
}
