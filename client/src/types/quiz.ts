export type QuestionType = 'MULTIPLE_CHOICE' | 'TRUE_FALSE'

/** One question as the teacher edits it. `correct` holds the marked option indices. */
export interface EditQuestion {
  uuid?: string
  type: QuestionType
  prompt: string
  options: string[]
  correct: number[]
}

export interface EditQuiz {
  uuid: string
  moduleUuid: string
  title: string
  published: boolean
  visibleFrom?: string
  visibleUntil?: string
  passPercent: number
  maxAttempts: number
  questions: EditQuestion[]
}

export interface UpdateQuizPayload {
  title: string
  visibleFrom?: string
  visibleUntil?: string
  passPercent: number
  maxAttempts: number
}

export type QuestionPayload = Omit<EditQuestion, 'uuid'>

/** Student view of a question. The server never sends the answer key. */
export interface StudentQuestion {
  uuid: string
  type: QuestionType
  prompt: string
  options: string[]
}

export interface Attempt {
  questions: StudentQuestion[]
  answers: Record<string, number[]>
  startedAt: string
}

/** The only thing a student learns about a graded quiz. */
export interface QuizResult {
  percent: number
  passed: boolean
  attemptsUsed: number
  maxAttempts: number
}

export type AssignmentType = 'HANDIN' | 'QUIZ'

export interface Assignment {
  uuid: string
  type: AssignmentType
  title: string
  moduleUuid: string
  moduleTitle: string
  published: boolean
  visibleFrom?: string
  visibleUntil?: string
  deadline?: string
  acceptsLateSubmissions?: boolean
  allowsGroupSubmission?: boolean
  maxPoints?: number
  passPercent: number
  maxAttempts: number
  attemptsUsed: number
  myResult?: QuizResult
}

export interface StudentResultRow {
  username: string
  email: string
  attempts: number
  percent?: number
  passed?: boolean
}
