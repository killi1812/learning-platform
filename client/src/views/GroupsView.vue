<template>
  <v-container class="py-6" max-width="1400">
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <div>
        <h1 class="text-h4 font-weight-bold text-high-emphasis my-0">My Groups</h1>
        <p class="text-subtitle-1 text-medium-emphasis mb-0">
          Overview of study groups you have joined across your courses.
        </p>
      </div>
      <v-btn
        color="primary"
        prepend-icon="mdi-book-open-outline"
        size="large"
        to="/courses"
        variant="elevated"
      >
        My Courses
      </v-btn>
    </div>

    <!-- Partial Warning -->
    <v-alert
      v-if="partialWarning"
      class="mb-4"
      color="warning"
      icon="mdi-alert-outline"
      rounded="lg"
      variant="tonal"
    >
      {{ partialWarning }}
    </v-alert>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <!-- Error State -->
    <v-alert
      v-else-if="loadError"
      class="mb-4"
      color="error"
      icon="mdi-alert-circle"
      rounded="lg"
      variant="tonal"
    >
      <div class="d-flex flex-wrap align-center justify-space-between ga-3">
        <span>{{ loadError }}</span>
        <v-btn color="error" size="small" variant="outlined" @click="loadGroups"> Retry </v-btn>
      </div>
    </v-alert>

    <!-- Empty State -->
    <v-card
      v-else-if="userGroups.length === 0"
      class="pa-12 text-center bg-surface-variant"
      elevation="0"
      rounded="lg"
    >
      <v-icon class="mb-4 text-medium-emphasis" icon="mdi-account-group-outline" size="64" />
      <h3 class="text-h6 font-weight-medium text-high-emphasis">
        You are not a member of any groups
      </h3>
      <p class="text-body-2 text-medium-emphasis mb-6">
        Visit any of your enrolled courses to form or join a study group for exercises and hand-ins.
      </p>
      <v-btn color="primary" prepend-icon="mdi-book-open-outline" to="/courses" variant="elevated">
        Go to My Courses
      </v-btn>
    </v-card>

    <!-- Groups Grid -->
    <div v-else>
      <v-row>
        <v-col
          v-for="item in userGroups"
          :key="item.group.uuid"
          class="d-flex"
          cols="12"
          lg="4"
          md="6"
        >
          <v-card
            class="bg-surface w-100 d-flex flex-column justify-space-between"
            elevation="2"
            rounded="lg"
          >
            <div>
              <v-card-item class="pb-2">
                <template #title>
                  <div class="d-flex align-center justify-space-between ga-2">
                    <span class="text-h6 font-weight-bold text-high-emphasis text-truncate">
                      {{ item.group.name }}
                    </span>
                    <v-chip
                      :color="
                        item.group.members.length >= item.group.maxMembers ? 'warning' : 'primary'
                      "
                      size="small"
                      variant="tonal"
                    >
                      <v-icon icon="mdi-account-multiple" size="small" start />
                      {{ item.group.members.length }} / {{ item.group.maxMembers }}
                    </v-chip>
                  </div>
                </template>
                <template #subtitle>
                  <div class="text-caption text-medium-emphasis mt-1 text-truncate">
                    {{ item.course.title }} ({{ item.course.term }} {{ item.course.year }})
                  </div>
                </template>
              </v-card-item>

              <v-divider />

              <v-card-text class="pt-4">
                <div class="text-caption font-weight-bold text-medium-emphasis text-uppercase mb-2">
                  Members ({{ item.group.members.length }})
                </div>

                <div class="d-flex flex-column ga-2">
                  <div
                    v-for="member in item.group.members"
                    :key="member.uuid"
                    class="d-flex align-center justify-space-between py-1 px-2 rounded bg-surface-variant text-caption"
                  >
                    <div class="d-flex align-center ga-2 text-truncate">
                      <v-avatar color="primary" size="24" variant="tonal">
                        <span class="text-caption font-weight-bold">
                          {{ member.username.slice(0, 1).toUpperCase() }}
                        </span>
                      </v-avatar>
                      <span class="font-weight-medium text-high-emphasis text-truncate">
                        {{ member.username }}
                      </span>
                    </div>

                    <v-chip v-if="isMe(member)" color="primary" size="x-small" variant="elevated">
                      You
                    </v-chip>
                  </div>
                </div>
              </v-card-text>
            </div>

            <div>
              <v-divider />
              <v-card-actions
                class="pa-4 bg-surface-variant d-flex justify-space-between align-center"
              >
                <v-btn
                  color="error"
                  :loading="leavingUuid === item.group.uuid"
                  prepend-icon="mdi-account-remove-outline"
                  size="small"
                  variant="text"
                  @click="handleLeaveGroup(item.group)"
                >
                  Leave
                </v-btn>
                <v-btn
                  color="primary"
                  prepend-icon="mdi-open-in-new"
                  size="small"
                  :to="`/courses/${item.course.uuid}?tab=groups`"
                  variant="elevated"
                >
                  View Course
                </v-btn>
              </v-card-actions>
            </div>
          </v-card>
        </v-col>
      </v-row>
    </div>
  </v-container>
</template>

<script setup lang="ts">
import type { Course, Group, GroupMember } from '@/types/course'
import { onMounted, ref } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { getMyCourses, leaveGroup, listCourseGroups } from '@/api/course'
import { useAuth } from '@/composables/useAuth'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { findUserGroup, isCurrentUserGroupMember } from '@/util/group'

interface UserGroupItem {
  course: Course
  group: Group
}

const { currentUser } = useAuth()
const toast = useToast()
const { confirm } = useConfirm()

const loading = ref(false)
const loadError = ref<string | undefined>(undefined)
const partialWarning = ref<string | undefined>(undefined)
const userGroups = ref<UserGroupItem[]>([])
const leavingUuid = ref<string | undefined>(undefined)
let fetchCounter = 0

function isMe(member: GroupMember): boolean {
  return isCurrentUserGroupMember(member, currentUser.value)
}

async function loadGroups(): Promise<void> {
  const requestId = ++fetchCounter
  loading.value = true
  loadError.value = undefined
  partialWarning.value = undefined

  try {
    const enrollments = await getMyCourses()
    if (requestId !== fetchCounter) {
      return
    }
    const results: UserGroupItem[] = []
    const failedCourses: string[] = []

    await Promise.all(
      enrollments.map(async enrollment => {
        try {
          const groups = await listCourseGroups(enrollment.course.uuid)
          const myGroup = findUserGroup(groups, currentUser.value)
          if (myGroup) {
            results.push({
              course: enrollment.course,
              group: myGroup,
            })
          }
        } catch {
          failedCourses.push(enrollment.course.title)
        }
      }),
    )

    if (requestId !== fetchCounter) {
      return
    }
    userGroups.value = results
    if (failedCourses.length > 0) {
      partialWarning.value = `Could not load groups for: ${failedCourses.join(', ')}`
    }
  } catch (error: unknown) {
    if (requestId === fetchCounter) {
      userGroups.value = []
      loadError.value = getApiErrorMessage(error, 'Failed to load your groups.')
    }
  } finally {
    if (requestId === fetchCounter) {
      loading.value = false
    }
  }
}

async function handleLeaveGroup(group: Group): Promise<void> {
  if (
    !(await confirm({
      title: 'Leave Group',
      message: `Are you sure you want to leave "${group.name}"?`,
      confirmText: 'Leave Group',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  leavingUuid.value = group.uuid
  try {
    await leaveGroup(group.uuid)
    toast.success(`You have left ${group.name}.`)
    await loadGroups()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to leave group.'))
  } finally {
    leavingUuid.value = undefined
  }
}

onMounted(() => {
  void loadGroups()
})
</script>
