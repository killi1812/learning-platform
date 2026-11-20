<template>
  <v-container class="py-6" max-width="1400">
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <DashboardGreeting
        :now="now"
        :pending-count="isTeacher ? pending.data.value.length : undefined"
        :upcoming-count="weekCount"
        :username="currentUser?.username"
      />
      <QuickActions :is-teacher="isTeacher" />
    </div>

    <v-row v-if="myCourses.length > 0">
      <template v-if="studentCourses.length > 0">
        <v-col cols="12" lg="7">
          <DashboardCard
            :empty="studentUpcoming.length === 0 ? 'No upcoming deadlines.' : undefined"
            :error="assignments.error.value"
            icon="mdi-clock-alert-outline"
            :loading="assignments.loading.value"
            title="Upcoming deadlines"
          >
            <DeadlineList
              :items="studentUpcoming"
              :now="now"
              :submitted-uuids="submittedUuids.data.value"
            />
          </DashboardCard>
        </v-col>
        <v-col cols="12" lg="5">
          <DashboardCard
            :empty="progress.data.value.length === 0 ? 'No graded work yet.' : undefined"
            :error="progress.error.value"
            icon="mdi-chart-line"
            :loading="progress.loading.value"
            title="Course progress"
          >
            <CourseProgressList :courses="progress.data.value" />
          </DashboardCard>
        </v-col>
        <v-col cols="12" md="6">
          <DashboardCard
            :empty="quizzes.length === 0 ? 'No quizzes in your courses.' : undefined"
            :error="assignments.error.value"
            icon="mdi-help-circle-outline"
            :loading="assignments.loading.value"
            title="Quiz status"
          >
            <QuizStatusList :quizzes="quizzes" />
          </DashboardCard>
        </v-col>
        <v-col cols="12" md="6">
          <DashboardCard
            :empty="groups.data.value.length === 0 ? 'You are not in any group.' : undefined"
            :error="groups.error.value"
            icon="mdi-account-group-outline"
            :loading="groups.loading.value"
            title="My groups"
          >
            <GroupList :groups="groups.data.value" />
          </DashboardCard>
        </v-col>
      </template>

      <template v-if="teacherCourses.length > 0">
        <v-col cols="12" lg="4" md="6">
          <DashboardCard
            :error="pending.error.value"
            icon="mdi-clipboard-check-outline"
            :loading="pending.loading.value"
            title="Pending grading"
          >
            <PendingGradingList :items="pending.data.value" />
          </DashboardCard>
        </v-col>
        <v-col cols="12" lg="4" md="6">
          <DashboardCard
            :empty="teacherUpcoming.length === 0 ? 'No upcoming deadlines.' : undefined"
            :error="assignments.error.value"
            icon="mdi-calendar-clock-outline"
            :loading="assignments.loading.value"
            title="My deadlines"
          >
            <DeadlineList :items="teacherUpcoming" :now="now" />
          </DashboardCard>
        </v-col>
        <v-col cols="12" lg="4">
          <DashboardCard
            :empty="activity.data.value.length === 0 ? 'No hand-ins due around now.' : undefined"
            :error="activity.error.value"
            icon="mdi-upload-multiple"
            :loading="activity.loading.value"
            title="Submission activity"
          >
            <SubmissionActivityList :rows="activity.data.value" />
          </DashboardCard>
        </v-col>
      </template>
    </v-row>

    <h2 class="text-h6 font-weight-bold text-high-emphasis mt-8 mb-4">Your courses</h2>
    <EnrolledCoursesGrid
      :courses="myCourses"
      :error="loadError"
      :loading="loading"
      @retry="refresh"
    />
  </v-container>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import CourseProgressList from '@/components/dashboard/CourseProgressList.vue'
import DashboardCard from '@/components/dashboard/DashboardCard.vue'
import DashboardGreeting from '@/components/dashboard/DashboardGreeting.vue'
import DeadlineList from '@/components/dashboard/DeadlineList.vue'
import GroupList from '@/components/dashboard/GroupList.vue'
import PendingGradingList from '@/components/dashboard/PendingGradingList.vue'
import QuickActions from '@/components/dashboard/QuickActions.vue'
import QuizStatusList from '@/components/dashboard/QuizStatusList.vue'
import SubmissionActivityList from '@/components/dashboard/SubmissionActivityList.vue'
import EnrolledCoursesGrid from '@/components/EnrolledCoursesGrid.vue'
import { useAuth } from '@/composables/useAuth'
import { useDashboardData } from '@/composables/useDashboardData'
import { useMyCourses } from '@/composables/useMyCourses'

const WEEK_MS = 7 * 24 * 3_600_000

const { currentUser, isAdminOrSuperadmin } = useAuth()
const { myCourses, loading, loadError, fetchMyCourses } = useMyCourses()
const {
  assignments,
  progress,
  groups,
  pending,
  submittedUuids,
  activity,
  studentCourses,
  teacherCourses,
  load,
} = useDashboardData(myCourses)

const now = ref(new Date())

const isTeacher = computed(() => teacherCourses.value.length > 0 || isAdminOrSuperadmin.value)

const upcoming = computed(() =>
  assignments.data.value.filter(item => Date.parse(item.deadline) >= now.value.getTime()),
)
const studentUpcoming = computed(() =>
  upcoming.value.filter(item => item.role === 'STUDENT' && !item.quizFinished),
)
const teacherUpcoming = computed(() => upcoming.value.filter(item => item.role === 'TEACHER'))
const quizzes = computed(() =>
  assignments.data.value.filter(item => item.role === 'STUDENT' && item.type === 'QUIZ'),
)
const weekCount = computed(
  () =>
    upcoming.value.filter(item => Date.parse(item.deadline) - now.value.getTime() <= WEEK_MS)
      .length,
)

async function refresh(): Promise<void> {
  now.value = new Date()
  await fetchMyCourses()
  await load()
}

onMounted(() => {
  void refresh()
})
</script>
