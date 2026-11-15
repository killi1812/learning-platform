<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-4">
      <h2 class="text-h5 font-weight-bold text-high-emphasis my-0">Course Modules</h2>
      <v-btn
        v-if="isTeacher"
        color="primary"
        prepend-icon="mdi-plus"
        variant="elevated"
        @click="emit('create-module')"
      >
        Add Module
      </v-btn>
    </div>

    <!-- Empty State -->
    <v-card
      v-if="modules.length === 0"
      class="pa-8 text-center bg-surface-variant"
      elevation="0"
      rounded="lg"
    >
      <v-icon class="mb-3 text-medium-emphasis" icon="mdi-folder-open-outline" size="48" />
      <h3 class="text-h6 font-weight-medium text-high-emphasis">No modules created yet</h3>
      <p class="text-body-2 text-medium-emphasis">
        {{
          isTeacher
            ? 'Click "Add Module" to begin designing your course structure.'
            : 'The teacher has not published any modules for this course yet.'
        }}
      </p>
    </v-card>

    <!-- Module List -->
    <div v-else class="d-flex flex-column ga-3">
      <v-card
        v-for="(mod, index) in modules"
        :key="mod.uuid"
        class="pa-4 bg-surface"
        elevation="1"
        rounded="lg"
      >
        <div class="d-flex flex-wrap align-center justify-space-between ga-2">
          <div class="d-flex align-center ga-3">
            <v-avatar color="primary" size="32" variant="tonal">
              <span class="text-caption font-weight-bold">{{ index + 1 }}</span>
            </v-avatar>

            <div class="d-flex flex-column justify-center">
              <h3 class="text-subtitle-1 font-weight-bold text-high-emphasis my-0">
                {{ mod.title }}
              </h3>
              <div v-if="mod.dependsOn" class="d-flex align-center ga-1 mt-1">
                <v-icon color="secondary" icon="mdi-link-variant" size="small" />
                <span class="text-caption text-secondary">
                  Prerequisite: {{ mod.dependsOn.title }}
                </span>
                <v-chip
                  v-if="lockedModules[mod.uuid]"
                  class="ml-1 font-weight-bold"
                  color="warning"
                  size="x-small"
                  variant="tonal"
                >
                  <v-icon icon="mdi-lock-outline" size="x-small" start />
                  Locked
                </v-chip>
              </div>
            </div>
          </div>

          <!-- Teacher Reorder and Manage Controls -->
          <div v-if="isTeacher" class="d-flex align-center ga-1">
            <v-btn
              aria-label="Move module up"
              density="comfortable"
              :disabled="reorderingModules || index === 0"
              icon="mdi-arrow-up"
              title="Move Up"
              variant="text"
              @click="emit('move-module', index, -1)"
            />
            <v-btn
              aria-label="Move module down"
              density="comfortable"
              :disabled="reorderingModules || index === modules.length - 1"
              icon="mdi-arrow-down"
              title="Move Down"
              variant="text"
              @click="emit('move-module', index, 1)"
            />
            <v-btn
              aria-label="Edit module"
              density="comfortable"
              icon="mdi-pencil-outline"
              title="Edit Module"
              variant="text"
              @click="emit('edit-module', mod)"
            />
            <v-btn
              aria-label="Delete module"
              color="error"
              density="comfortable"
              icon="mdi-delete-outline"
              title="Delete Module"
              variant="text"
              @click="emit('delete-module', mod.uuid)"
            />
          </div>
        </div>

        <!-- Course Materials Section -->
        <v-divider class="my-3" />
        <div class="d-flex align-center justify-space-between mb-2">
          <span class="text-caption font-weight-bold text-medium-emphasis">
            Materials ({{ (modulePages[mod.uuid] || []).length }})
          </span>
          <v-menu v-if="isTeacher">
            <template #activator="{ props: menuProps }">
              <v-btn
                v-bind="menuProps"
                append-icon="mdi-chevron-down"
                color="primary"
                prepend-icon="mdi-plus"
                size="small"
                variant="text"
              >
                Add
              </v-btn>
            </template>
            <v-list density="compact" elevation="2" rounded="md">
              <v-list-item
                prepend-icon="mdi-file-document-outline"
                title="Material"
                @click="emit('upload-material', mod.uuid)"
              />
              <v-list-item
                prepend-icon="mdi-clipboard-text-outline"
                title="Assignment"
                @click="emit('create-handin', mod.uuid)"
              />
              <v-list-item
                prepend-icon="mdi-help-circle-outline"
                title="Quiz"
                @click="emit('add-quiz', mod.uuid)"
              />
            </v-list>
          </v-menu>
        </div>

        <!-- Material Items -->
        <div
          v-if="lockedModules[mod.uuid]"
          class="d-flex align-center ga-2 py-2 text-medium-emphasis"
        >
          <v-icon color="warning" icon="mdi-lock-outline" size="small" />
          <span class="text-caption">
            This module is locked until prerequisite
            <template v-if="mod.dependsOn">({{ mod.dependsOn.title }})</template>
            is completed.
          </span>
        </div>
        <div v-else-if="(modulePages[mod.uuid] || []).length > 0" class="d-flex flex-column ga-2">
          <v-card
            v-for="page in modulePages[mod.uuid]"
            :key="page.uuid"
            class="pa-3 bg-surface-variant d-flex align-center justify-space-between"
            elevation="0"
            rounded="md"
          >
            <button
              class="material-trigger d-flex align-center ga-3 overflow-hidden flex-grow-1 text-left"
              type="button"
              @click="emit('view-material', mod.uuid, page.uuid)"
            >
              <v-avatar :color="getFileFormatColor(page)" size="32" variant="tonal">
                <v-icon :icon="getFileFormatIcon(page)" size="small" />
              </v-avatar>
              <div class="d-flex flex-column overflow-hidden">
                <span class="text-body-2 font-weight-bold text-high-emphasis text-truncate">
                  {{ page.title }}
                </span>
                <span class="text-caption text-medium-emphasis text-truncate">
                  <template v-if="page.fileSize">{{ formatBytes(page.fileSize) }} &bull; </template>
                  <template v-if="page.createdAt">{{ formatShortDate(page.createdAt) }}</template>
                </span>
              </div>
            </button>

            <div class="d-flex align-center ga-1 ml-2">
              <v-btn
                color="primary"
                prepend-icon="mdi-eye-outline"
                size="small"
                variant="tonal"
                @click="emit('view-material', mod.uuid, page.uuid)"
              >
                Preview
              </v-btn>
              <v-btn
                v-if="isTeacher"
                aria-label="Edit material"
                color="primary"
                icon="mdi-pencil-outline"
                size="small"
                title="Edit Material"
                variant="text"
                @click="emit('edit-material', mod.uuid, page)"
              />
              <v-btn
                v-if="isTeacher"
                aria-label="Delete material"
                color="error"
                icon="mdi-delete-outline"
                size="small"
                title="Delete Material"
                variant="text"
                @click="emit('delete-material', mod.uuid, page.uuid)"
              />
            </div>
          </v-card>
        </div>
        <div v-else class="text-caption text-medium-emphasis py-1 font-italic">
          No materials uploaded for this module yet.
        </div>

        <!-- Quizzes -->
        <div
          v-if="!lockedModules[mod.uuid] && quizzesIn(mod.uuid).length > 0"
          class="d-flex flex-column ga-2 mt-3"
        >
          <span class="text-caption font-weight-bold text-medium-emphasis">
            Quizzes ({{ quizzesIn(mod.uuid).length }})
          </span>
          <QuizListItem
            v-for="quiz in quizzesIn(mod.uuid)"
            :key="quiz.uuid"
            :assignment="quiz"
            :is-teacher="isTeacher"
            @delete="emit('delete-quiz', quiz)"
            @edit="emit('edit-quiz', quiz)"
            @results="emit('results-quiz', quiz)"
            @take="emit('take-quiz', quiz)"
          />
        </div>

        <!-- Hand-in Assignments -->
        <div
          v-if="!lockedModules[mod.uuid] && handinsIn(mod.uuid).length > 0"
          class="d-flex flex-column ga-2 mt-3"
        >
          <span class="text-caption font-weight-bold text-medium-emphasis">
            Assignments ({{ handinsIn(mod.uuid).length }})
          </span>
          <HandinAssignmentItem
            v-for="asgn in handinsIn(mod.uuid)"
            :key="asgn.uuid"
            :assignment="asgn"
            :is-teacher="isTeacher"
            @delete="emit('delete-handin', asgn)"
            @edit="emit('edit-handin', asgn)"
            @submissions="emit('submissions-handin', asgn)"
            @submit="emit('submit-handin', asgn)"
          />
        </div>
      </v-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FileUpload } from '@/types/content'
import type { Module } from '@/types/course'
import type { Assignment } from '@/types/quiz'
import HandinAssignmentItem from '@/components/HandinAssignmentItem.vue'
import QuizListItem from '@/components/QuizListItem.vue'
import { formatShortDate } from '@/util/dateTime'
import { formatBytes, getFileFormatColor, getFileFormatIcon } from '@/util/file'

interface Props {
  modules: readonly Module[]
  modulePages: Readonly<Record<string, FileUpload[]>>
  lockedModules: Readonly<Record<string, boolean>>
  assignments: readonly Assignment[]
  isTeacher: boolean
  reorderingModules: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'create-module': []
  'edit-module': [mod: Module]
  'delete-module': [moduleUuid: string]
  'move-module': [index: number, direction: -1 | 1]
  'upload-material': [moduleUuid: string]
  'edit-material': [moduleUuid: string, page: FileUpload]
  'delete-material': [moduleUuid: string, pageUuid: string]
  'view-material': [moduleUuid: string, pageUuid: string]
  'create-handin': [moduleUuid: string]
  'edit-handin': [asgn: Assignment]
  'delete-handin': [asgn: Assignment]
  'submit-handin': [asgn: Assignment]
  'submissions-handin': [asgn: Assignment]
  'add-quiz': [moduleUuid: string]
  'edit-quiz': [quiz: Assignment]
  'results-quiz': [quiz: Assignment]
  'delete-quiz': [quiz: Assignment]
  'take-quiz': [quiz: Assignment]
}>()

function quizzesIn(moduleUuid: string): Assignment[] {
  return props.assignments.filter(a => a.moduleUuid === moduleUuid && a.type !== 'HANDIN')
}

function handinsIn(moduleUuid: string): Assignment[] {
  return props.assignments.filter(a => a.moduleUuid === moduleUuid && a.type === 'HANDIN')
}
</script>

<style scoped>
.material-trigger {
  background: transparent;
  border: none;
  padding: 0;
  cursor: pointer;
}
</style>
