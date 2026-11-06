import { authApi, getApiErrorMessage } from './client'
import type {
  Assignment,
  Attempt,
  EditQuiz,
  QuestionPayload,
  QuizResult,
  StudentResultRow,
  UpdateQuizPayload,
} from '@/types/quiz'

async function wrapRequest<T>(promise: Promise<{ data: T }>): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

const quizPath = (courseUuid: string, quizUuid: string) => `/course/${courseUuid}/quiz/${quizUuid}`

// ---- teacher ----

export function createQuiz(
  courseUuid: string,
  moduleUuid: string,
  title: string,
): Promise<EditQuiz> {
  return wrapRequest(
    authApi.post<EditQuiz>(`/course/${courseUuid}/module/${moduleUuid}/quiz`, { title }),
  )
}

export function getQuizForEdit(courseUuid: string, quizUuid: string): Promise<EditQuiz> {
  return wrapRequest(authApi.get<EditQuiz>(quizPath(courseUuid, quizUuid)))
}

export function updateQuiz(
  courseUuid: string,
  quizUuid: string,
  payload: UpdateQuizPayload,
): Promise<EditQuiz> {
  return wrapRequest(authApi.put<EditQuiz>(quizPath(courseUuid, quizUuid), payload))
}

export function replaceQuestions(
  courseUuid: string,
  quizUuid: string,
  questions: QuestionPayload[],
): Promise<EditQuiz> {
  return wrapRequest(
    authApi.put<EditQuiz>(`${quizPath(courseUuid, quizUuid)}/questions`, { questions }),
  )
}

export function publishQuiz(courseUuid: string, quizUuid: string): Promise<EditQuiz> {
  return wrapRequest(authApi.post<EditQuiz>(`${quizPath(courseUuid, quizUuid)}/publish`))
}

export function deleteQuiz(courseUuid: string, quizUuid: string): Promise<void> {
  return wrapRequest(authApi.delete<void>(quizPath(courseUuid, quizUuid)))
}

export function getQuizResults(courseUuid: string, quizUuid: string): Promise<StudentResultRow[]> {
  return wrapRequest(authApi.get<StudentResultRow[]>(`${quizPath(courseUuid, quizUuid)}/results`))
}

// ---- everyone enrolled ----

export function listAssignments(courseUuid: string): Promise<Assignment[]> {
  return wrapRequest(authApi.get<Assignment[]>(`/course/${courseUuid}/assignment`))
}

// ---- student ----

export function startAttempt(courseUuid: string, quizUuid: string): Promise<Attempt> {
  return wrapRequest(authApi.post<Attempt>(`${quizPath(courseUuid, quizUuid)}/attempt`))
}

export function saveAnswer(
  courseUuid: string,
  quizUuid: string,
  questionUuid: string,
  selected: number[],
): Promise<void> {
  return wrapRequest(
    authApi.put<void>(`${quizPath(courseUuid, quizUuid)}/attempt/answers/${questionUuid}`, {
      selected,
    }),
  )
}

export function submitAttempt(courseUuid: string, quizUuid: string): Promise<QuizResult> {
  return wrapRequest(authApi.post<QuizResult>(`${quizPath(courseUuid, quizUuid)}/attempt/submit`))
}
