import type { MyCourse } from '@/types/course'
import { onBeforeUnmount, ref } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { getMyCourses } from '@/api/course'

export function useMyCourses() {
  const myCourses = ref<MyCourse[]>([])
  const loading = ref(false)
  const loadError = ref<string | undefined>(undefined)

  let isUnmounted = false
  let requestCounter = 0

  onBeforeUnmount(() => {
    isUnmounted = true
  })

  async function fetchMyCourses(): Promise<void> {
    const requestId = ++requestCounter
    loading.value = true
    loadError.value = undefined
    try {
      const courses = await getMyCourses()
      if (!isUnmounted && requestId === requestCounter) {
        myCourses.value = courses
      }
    } catch (error: unknown) {
      if (!isUnmounted && requestId === requestCounter) {
        loadError.value = getApiErrorMessage(error, 'Failed to load your courses.')
      }
    } finally {
      if (!isUnmounted && requestId === requestCounter) {
        loading.value = false
      }
    }
  }

  return {
    myCourses,
    loading,
    loadError,
    fetchMyCourses,
  }
}
