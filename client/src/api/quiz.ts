import type {
  Assignment,
  Attempt,
  EditQuiz,
  QuestionPayload,
  QuizResult,
  StudentResultRow,
  UpdateQuizPayload,
} from '@/types/quiz'
import { authApi, request } from './client'

const enc = encodeURIComponent
function quizPath(courseUuid: string, quizUuid: string) {
  return `/course/${enc(courseUuid)}/quiz/${enc(quizUuid)}`
}

// ---- teacher ----

export function createQuiz(
  courseUuid: string,
  moduleUuid: string,
  title: string,
): Promise<EditQuiz> {
  return request(
    authApi.post<EditQuiz>(`/course/${enc(courseUuid)}/module/${enc(moduleUuid)}/quiz`, { title }),
  )
}

export function getQuizForEdit(courseUuid: string, quizUuid: string): Promise<EditQuiz> {
  return request(authApi.get<EditQuiz>(quizPath(courseUuid, quizUuid)))
}

export function updateQuiz(
  courseUuid: string,
  quizUuid: string,
  payload: UpdateQuizPayload,
): Promise<EditQuiz> {
  return request(authApi.put<EditQuiz>(quizPath(courseUuid, quizUuid), payload))
}

export function replaceQuestions(
  courseUuid: string,
  quizUuid: string,
  questions: QuestionPayload[],
): Promise<EditQuiz> {
  return request(
    authApi.put<EditQuiz>(`${quizPath(courseUuid, quizUuid)}/questions`, { questions }),
  )
}

export function publishQuiz(courseUuid: string, quizUuid: string): Promise<EditQuiz> {
  return request(authApi.post<EditQuiz>(`${quizPath(courseUuid, quizUuid)}/publish`))
}

export function deleteQuiz(courseUuid: string, quizUuid: string): Promise<void> {
  return request(authApi.delete<void>(quizPath(courseUuid, quizUuid)))
}

export function getQuizResults(courseUuid: string, quizUuid: string): Promise<StudentResultRow[]> {
  return request(authApi.get<StudentResultRow[]>(`${quizPath(courseUuid, quizUuid)}/results`))
}

// ---- everyone enrolled ----

export function listAssignments(courseUuid: string): Promise<Assignment[]> {
  return request(authApi.get<Assignment[]>(`/course/${enc(courseUuid)}/assignment`))
}

// ---- student ----

export function startAttempt(courseUuid: string, quizUuid: string): Promise<Attempt> {
  return request(authApi.post<Attempt>(`${quizPath(courseUuid, quizUuid)}/attempt`))
}

export function saveAnswer(
  courseUuid: string,
  quizUuid: string,
  questionUuid: string,
  selected: number[],
): Promise<void> {
  return request(
    authApi.put<void>(`${quizPath(courseUuid, quizUuid)}/attempt/answers/${enc(questionUuid)}`, {
      selected,
    }),
  )
}

export function submitAttempt(courseUuid: string, quizUuid: string): Promise<QuizResult> {
  return request(authApi.post<QuizResult>(`${quizPath(courseUuid, quizUuid)}/attempt/submit`))
}
