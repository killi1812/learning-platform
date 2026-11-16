<template>
  <div>
    <div class="d-flex flex-wrap align-center justify-space-between ga-3 mb-4">
      <div>
        <h2 class="text-h5 font-weight-bold text-high-emphasis my-0">Course Groups</h2>
        <p class="text-body-2 text-medium-emphasis mb-0 mt-1">
          Collaborate with peers on course assignments and team hand-ins.
        </p>
      </div>
      <div v-if="isTeacher" class="d-flex align-center ga-2">
        <v-btn
          color="secondary"
          prepend-icon="mdi-card-multiple-outline"
          variant="tonal"
          @click="emit('bulk-create')"
        >
          Bulk Create Groups
        </v-btn>
        <v-btn
          color="primary"
          prepend-icon="mdi-plus"
          variant="elevated"
          @click="emit('create-group')"
        >
          Create Group
        </v-btn>
      </div>
    </div>

    <!-- Student Membership Alert -->
    <v-alert
      v-if="myGroup"
      class="mb-6"
      color="success"
      icon="mdi-account-check"
      rounded="lg"
      variant="tonal"
    >
      <div class="d-flex flex-wrap align-center justify-space-between ga-3">
        <div>
          <div class="text-subtitle-1 font-weight-bold">You are a member of {{ myGroup.name }}</div>
          <div class="text-body-2 text-medium-emphasis">
            {{ myGroup.members.length }} / {{ myGroup.maxMembers }} members enrolled
          </div>
        </div>
        <v-btn
          v-if="!isTeacher"
          color="error"
          :loading="Boolean(actionLoading[myGroup.uuid])"
          prepend-icon="mdi-exit-to-app"
          size="small"
          variant="tonal"
          @click="emit('leave-group', myGroup)"
        >
          Leave Group
        </v-btn>
      </div>
    </v-alert>
    <v-alert
      v-else-if="!isTeacher && groups.length > 0"
      class="mb-6"
      color="info"
      icon="mdi-information-outline"
      rounded="lg"
      variant="tonal"
    >
      <div class="text-subtitle-1 font-weight-bold">You have not joined a group yet</div>
      <div class="text-body-2 text-medium-emphasis">
        Select an open group below to join your teammates for group submissions.
      </div>
    </v-alert>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-10">
      <v-progress-circular color="primary" indeterminate size="48" />
    </div>

    <!-- Empty State -->
    <v-card
      v-else-if="groups.length === 0"
      class="pa-8 text-center bg-surface-variant mb-6"
      elevation="0"
      rounded="lg"
    >
      <v-icon class="mb-3 text-medium-emphasis" icon="mdi-account-group-outline" size="48" />
      <h3 class="text-h6 font-weight-medium text-high-emphasis">No groups created yet</h3>
      <p class="text-body-2 text-medium-emphasis mb-4">
        {{
          isTeacher
            ? 'Use "Bulk Create Groups" to quickly generate numbered groups, or create a single group.'
            : 'The instructor has not created any groups for this course yet.'
        }}
      </p>
      <div v-if="isTeacher" class="d-flex justify-center ga-3">
        <v-btn
          color="primary"
          prepend-icon="mdi-card-multiple-outline"
          variant="elevated"
          @click="emit('bulk-create')"
        >
          Bulk Create Groups
        </v-btn>
        <v-btn
          color="secondary"
          prepend-icon="mdi-plus"
          variant="tonal"
          @click="emit('create-group')"
        >
          Create Group
        </v-btn>
      </div>
    </v-card>

    <!-- Group Cards Grid -->
    <v-row v-else class="mb-6" dense>
      <v-col v-for="group in groups" :key="group.uuid" class="d-flex" cols="12" lg="4" sm="6">
        <v-card
          class="bg-surface w-100 d-flex flex-column justify-space-between"
          :class="{ 'group-card-selected': myGroup?.uuid === group.uuid }"
          :elevation="myGroup?.uuid === group.uuid ? 3 : 1"
          rounded="lg"
        >
          <div>
            <v-card-item class="pb-2">
              <div class="d-flex align-center justify-space-between mb-2">
                <h3 class="text-subtitle-1 font-weight-bold text-high-emphasis my-0 text-truncate">
                  {{ group.name }}
                </h3>
                <div class="d-flex align-center ga-1">
                  <v-chip
                    v-if="myGroup?.uuid === group.uuid"
                    color="success"
                    size="x-small"
                    variant="flat"
                  >
                    Your Group
                  </v-chip>
                  <v-chip
                    :color="group.members.length >= group.maxMembers ? 'error' : 'primary'"
                    size="x-small"
                    variant="tonal"
                  >
                    {{ group.members.length }} / {{ group.maxMembers }}
                  </v-chip>
                </div>
              </div>

              <div v-if="isTeacher" class="d-flex justify-end ga-1 mb-1">
                <v-btn
                  aria-label="Edit group"
                  density="comfortable"
                  :disabled="Boolean(actionLoading[group.uuid])"
                  icon="mdi-pencil-outline"
                  size="small"
                  title="Edit Group"
                  variant="text"
                  @click="emit('edit-group', group)"
                />
                <v-btn
                  aria-label="Delete group"
                  color="error"
                  density="comfortable"
                  :disabled="Boolean(actionLoading[group.uuid])"
                  icon="mdi-delete-outline"
                  :loading="Boolean(actionLoading[group.uuid])"
                  size="small"
                  title="Delete Group"
                  variant="text"
                  @click="emit('delete-group', group)"
                />
              </div>
            </v-card-item>

            <v-card-text class="py-0">
              <v-divider class="mb-3" />
              <div class="text-caption font-weight-bold text-medium-emphasis mb-2">
                Members ({{ group.members.length }})
              </div>

              <div v-if="group.members.length > 0" class="d-flex flex-column ga-1">
                <div
                  v-for="member in group.members"
                  :key="member.uuid"
                  class="d-flex align-center justify-space-between py-1 px-2 rounded bg-surface-variant text-caption"
                >
                  <div class="d-flex align-center ga-2 text-truncate">
                    <v-icon icon="mdi-account" size="small" />
                    <span class="text-truncate font-weight-medium">
                      {{ member.username || member.email }}
                    </span>
                  </div>
                  <v-chip
                    v-if="isCurrentUserGroupMember(member, currentUser)"
                    color="primary"
                    size="x-small"
                    variant="flat"
                  >
                    You
                  </v-chip>
                </div>
              </div>
              <div v-else class="text-caption text-medium-emphasis font-italic py-2">
                No members have joined yet.
              </div>
            </v-card-text>
          </div>

          <v-card-actions class="pa-4 bg-surface-variant mt-3">
            <template v-if="!isTeacher">
              <v-btn
                v-if="myGroup?.uuid === group.uuid"
                block
                color="error"
                :disabled="Boolean(actionLoading[group.uuid])"
                :loading="Boolean(actionLoading[group.uuid])"
                prepend-icon="mdi-exit-to-app"
                size="small"
                variant="tonal"
                @click="emit('leave-group', group)"
              >
                Leave Group
              </v-btn>
              <v-btn
                v-else
                block
                :color="!myGroup && group.members.length < group.maxMembers ? 'primary' : undefined"
                :disabled="
                  Boolean(myGroup) ||
                  group.members.length >= group.maxMembers ||
                  Boolean(actionLoading[group.uuid])
                "
                :loading="Boolean(actionLoading[group.uuid])"
                :prepend-icon="
                  group.members.length >= group.maxMembers
                    ? 'mdi-account-cancel'
                    : myGroup
                      ? 'mdi-account-check'
                      : 'mdi-account-plus'
                "
                size="small"
                :variant="
                  !myGroup && group.members.length < group.maxMembers ? 'elevated' : 'tonal'
                "
                @click="emit('join-group', group)"
              >
                {{
                  group.members.length >= group.maxMembers
                    ? 'Group Full'
                    : myGroup
                      ? 'In Another Group'
                      : 'Join Group'
                }}
              </v-btn>
            </template>
            <template v-else>
              <div class="d-flex justify-space-between align-center w-100">
                <span class="text-caption text-medium-emphasis">
                  {{
                    group.maxMembers - group.members.length > 0
                      ? `${group.maxMembers - group.members.length} spots remaining`
                      : 'Capacity reached'
                  }}
                </span>
                <v-btn
                  color="primary"
                  prepend-icon="mdi-pencil-outline"
                  size="small"
                  variant="tonal"
                  @click="emit('edit-group', group)"
                >
                  Manage
                </v-btn>
              </div>
            </template>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import type { UserClaims } from '@/types/auth'
import type { Group } from '@/types/course'
import { isCurrentUserGroupMember } from '@/util/group'

interface Props {
  groups: readonly Group[]
  myGroup?: Group
  currentUser?: UserClaims
  isTeacher: boolean
  loading: boolean
  actionLoading: Readonly<Record<string, boolean>>
}

defineProps<Props>()

const emit = defineEmits<{
  'bulk-create': []
  'create-group': []
  'edit-group': [group: Group]
  'delete-group': [group: Group]
  'join-group': [group: Group]
  'leave-group': [group: Group]
}>()
</script>

<style scoped>
.group-card-selected {
  border: 2px solid rgb(var(--v-theme-primary));
}
</style>
