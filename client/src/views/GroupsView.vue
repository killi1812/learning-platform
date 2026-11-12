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
        variant="elevated"
        to="/courses"
      >
        My Courses
      </v-btn>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

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
      <v-btn color="primary" prepend-icon="mdi-book-open-outline" variant="elevated" to="/courses">
        Go to My Courses
      </v-btn>
    </v-card>

    <!-- Groups Grid -->
    <div v-else>
      <v-row>
        <v-col
          v-for="item in userGroups"
          :key="item.group.uuid"
          cols="12"
          md="6"
          lg="4"
          class="d-flex"
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
                      <v-icon icon="mdi-account-multiple" start size="small" />
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
                  size="small"
                  variant="text"
                  prepend-icon="mdi-account-remove-outline"
                  :loading="leavingUuid === item.group.uuid"
                  @click="handleLeaveGroup(item.group)"
                >
                  Leave
                </v-btn>
                <v-btn
                  color="primary"
                  size="small"
                  variant="elevated"
                  prepend-icon="mdi-open-in-new"
                  :to="`/courses/${item.course.uuid}?tab=groups`"
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
import { ref, onMounted } from 'vue'
import { getMyCourses, listCourseGroups, leaveGroup } from '@/api/course'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import type { Course, Group, GroupMember } from '@/types/course'

interface UserGroupItem {
  course: Course
  group: Group
}

const { currentUser } = useAuth()
const toast = useToast()
const { confirm } = useConfirm()

const loading = ref(false)
const userGroups = ref<UserGroupItem[]>([])
const leavingUuid = ref<string | null>(null)

function isMe(member: GroupMember): boolean {
  if (!currentUser.value) return false
  const userUuid = currentUser.value.userUuid
  const userEmail = currentUser.value.email
  if (userUuid && member.uuid === userUuid) return true
  if (userEmail && member.email.toLowerCase() === userEmail.toLowerCase()) return true
  return false
}

async function loadGroups() {
  loading.value = true
  try {
    const enrollments = await getMyCourses()
    const results: UserGroupItem[] = []
    const userUuid = currentUser.value?.userUuid
    const userEmail = currentUser.value?.email

    // Fetch groups for each enrolled course in parallel
    const groupPromises = enrollments.map(async enrollment => {
      try {
        const groups = await listCourseGroups(enrollment.course.uuid)
        const myGroup = groups.find(g =>
          g.members.some(m => {
            if (userUuid && m.uuid === userUuid) return true
            if (userEmail && m.email.toLowerCase() === userEmail.toLowerCase()) return true
            return false
          }),
        )
        if (myGroup) {
          results.push({
            course: enrollment.course,
            group: myGroup,
          })
        }
      } catch {
        // If groups fail to load for one course, keep other courses intact
      }
    })

    await Promise.all(groupPromises)
    userGroups.value = results
  } catch (err: unknown) {
    userGroups.value = []
    toast.error(err instanceof Error ? err.message : 'Failed to load your groups.')
  } finally {
    loading.value = false
  }
}

async function handleLeaveGroup(group: Group) {
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
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to leave group.')
  } finally {
    leavingUuid.value = null
  }
}

onMounted(() => {
  loadGroups()
})
</script>
