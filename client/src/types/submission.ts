import type { GradeDto } from './grade'

export type PayloadType = 'URL' | 'FILE'
export type SubmissionStatus = 'QUEUED'

export interface UploadTicket {
  ticketUuid: string
  submittedAt: string
  expiresAt: string
}

export interface SubmissionAccepted {
  uuid: string
  submittedAt: string
  status: SubmissionStatus
}

export type SubmissionGrade = Omit<GradeDto, 'maxPoints' | 'gradedByEmail'>

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
  isGroup?: boolean
  groupName?: string
  memberEmails?: string[]
  grade?: SubmissionGrade
}

export interface CreateHandinPayload {
  title: string
  deadline?: string
  acceptsLateSubmissions: boolean
  allowsGroupSubmission?: boolean
  maxPoints?: number
}

export type UpdateHandinPayload = CreateHandinPayload

export interface HandinAssignment {
  uuid: string
  moduleUuid?: string
  title: string
  position: number
  deadline?: string
  acceptsLateSubmissions: boolean
  allowsGroupSubmission?: boolean
  maxPoints?: number
  createdAt: string
  updatedAt: string
}
