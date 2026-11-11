<template>
  <v-container class="pt-2 pb-6" max-width="1200">
    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <div v-else-if="course">
      <!-- Course Title -->
      <h1 class="text-h4 font-weight-bold text-high-emphasis mt-0 mb-4">
        {{ course.title }}
      </h1>

      <!-- Tabs Navigation -->
      <v-tabs v-model="activeTab" class="mb-6" color="primary">
        <v-tab value="overview">
          <v-icon start icon="mdi-information-outline" />
          Overview
        </v-tab>
        <v-tab v-if="isEnrolled" value="modules">
          <v-icon start icon="mdi-view-module" />
          Course Modules ({{ modules.length }})
        </v-tab>
        <v-tab v-if="isTeacher" value="participants">
          <v-icon start icon="mdi-account-group" />
          Participants ({{ enrollments.length }})
        </v-tab>
        <v-tab v-if="isEnrolled" value="assignments">
          <v-icon start icon="mdi-clipboard-text-outline" />
          Assignments
        </v-tab>
        <v-tab v-if="isEnrolled" value="groups">
          <v-icon start icon="mdi-account-multiple" />
          Groups ({{ groups.length }})
        </v-tab>
        <v-tab v-if="isTeacher" value="settings">
          <v-icon start icon="mdi-cog" />
          Course Settings
        </v-tab>
      </v-tabs>

      <!-- TAB: OVERVIEW -->
      <div v-if="activeTab === 'overview'">
        <!-- Enrollment Protection Alert -->
        <v-alert
          v-if="!isEnrolled"
          class="mb-6"
          color="info"
          icon="mdi-shield-lock-outline"
          variant="tonal"
        >
          <div class="d-flex flex-wrap align-center justify-space-between ga-3">
            <div>
              <div class="text-subtitle-1 font-weight-bold">
                You are not enrolled in this course
              </div>
              <div class="text-body-2 text-medium-emphasis">
                Enroll as a participant to access modules, materials, assignments, and groups.
              </div>
            </div>
            <v-btn
              color="primary"
              :loading="enrolling"
              prepend-icon="mdi-account-plus"
              variant="elevated"
              @click="handleSelfEnroll"
            >
              Enroll in Course
            </v-btn>
          </div>
        </v-alert>

        <v-card class="pa-6 bg-surface mb-6" elevation="1" rounded="lg">
          <div class="d-flex flex-wrap align-center ga-3 mb-4">
            <v-chip
              v-if="userRole"
              :color="userRole === 'TEACHER' ? 'primary' : 'secondary'"
              size="default"
              variant="flat"
            >
              {{ userRole }}
            </v-chip>
            <v-chip color="secondary" size="default" variant="tonal">
              {{ course.term }} {{ course.year }}
            </v-chip>
          </div>

          <div
            v-if="course.teachers && course.teachers.length > 0"
            class="d-flex align-center ga-2 mb-4 text-body-1 text-medium-emphasis"
          >
            <v-icon icon="mdi-account-tie-outline" size="small" />
            <span>
              {{ course.teachers.length === 1 ? 'Professor' : 'Professors' }}:
              <strong class="text-high-emphasis">{{ course.teachers.join(', ') }}</strong>
            </span>
          </div>

          <v-divider class="mb-4" />

          <h2 class="text-h6 font-weight-bold text-high-emphasis mb-2">Description</h2>
          <p class="text-body-1 text-medium-emphasis mb-0" style="white-space: pre-line">
            {{ course.description || 'No course description provided.' }}
          </p>
        </v-card>
      </div>

      <!-- TAB: MODULES -->
      <div v-if="activeTab === 'modules' && isEnrolled">
        <div class="d-flex align-center justify-space-between mb-4">
          <h2 class="text-h5 font-weight-bold text-high-emphasis my-0">Course Modules</h2>
          <v-btn
            v-if="isTeacher"
            color="primary"
            prepend-icon="mdi-plus"
            variant="elevated"
            @click="openCreateModuleDialog"
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
                      <v-icon start icon="mdi-lock-outline" size="x-small" />
                      Locked
                    </v-chip>
                  </div>
                </div>
              </div>

              <!-- Teacher Reorder and Manage Controls -->
              <div v-if="isTeacher" class="d-flex align-center ga-1">
                <v-btn
                  density="comfortable"
                  :disabled="index === 0"
                  icon="mdi-arrow-up"
                  title="Move Up"
                  variant="text"
                  @click="moveModule(index, -1)"
                />
                <v-btn
                  density="comfortable"
                  :disabled="index === modules.length - 1"
                  icon="mdi-arrow-down"
                  title="Move Down"
                  variant="text"
                  @click="moveModule(index, 1)"
                />
                <v-btn
                  density="comfortable"
                  icon="mdi-pencil-outline"
                  title="Edit Module"
                  variant="text"
                  @click="openEditModuleDialog(mod)"
                />
                <v-btn
                  color="error"
                  density="comfortable"
                  icon="mdi-delete-outline"
                  title="Delete Module"
                  variant="text"
                  @click="handleDeleteModule(mod.uuid)"
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
                    size="small"
                    color="primary"
                    variant="text"
                    prepend-icon="mdi-plus"
                    append-icon="mdi-chevron-down"
                  >
                    Add
                  </v-btn>
                </template>
                <v-list density="compact" elevation="2" rounded="md">
                  <v-list-item
                    prepend-icon="mdi-file-document-outline"
                    title="Material"
                    @click="openUploadDialog(mod.uuid)"
                  />
                  <v-list-item
                    prepend-icon="mdi-clipboard-text-outline"
                    title="Assignment"
                    @click="openCreateHandinDialog(mod.uuid)"
                  />
                  <v-list-item
                    prepend-icon="mdi-help-circle-outline"
                    title="Quiz"
                    @click="handleAddQuiz(mod.uuid)"
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
            <div
              v-else-if="(modulePages[mod.uuid] || []).length > 0"
              class="d-flex flex-column ga-2"
            >
              <v-card
                v-for="page in modulePages[mod.uuid]"
                :key="page.uuid"
                class="pa-3 bg-surface-variant d-flex align-center justify-space-between"
                elevation="0"
                rounded="md"
              >
                <div
                  class="d-flex align-center ga-3 overflow-hidden cursor-pointer flex-grow-1"
                  @click="viewMaterial(mod.uuid, page.uuid)"
                >
                  <v-avatar size="32" :color="getPageFormatColor(page)" variant="tonal">
                    <v-icon :icon="getPageFormatIcon(page)" size="small" />
                  </v-avatar>
                  <div class="d-flex flex-column overflow-hidden">
                    <span class="text-body-2 font-weight-bold text-high-emphasis text-truncate">
                      {{ page.title }}
                    </span>
                    <span class="text-caption text-medium-emphasis text-truncate">
                      <template v-if="page.fileSize">{{ formatBytes(page.fileSize) }} • </template>
                      <template v-if="page.createdAt">{{ formatDate(page.createdAt) }}</template>
                    </span>
                  </div>
                </div>

                <div class="d-flex align-center ga-1 ml-2">
                  <v-btn
                    size="small"
                    color="primary"
                    variant="tonal"
                    prepend-icon="mdi-eye-outline"
                    @click="viewMaterial(mod.uuid, page.uuid)"
                  >
                    Preview
                  </v-btn>
                  <v-btn
                    v-if="isTeacher"
                    size="small"
                    color="primary"
                    icon="mdi-pencil-outline"
                    variant="text"
                    title="Edit Material"
                    @click="openEditMaterialDialog(mod.uuid, page)"
                  />
                  <v-btn
                    v-if="isTeacher"
                    size="small"
                    color="error"
                    icon="mdi-delete-outline"
                    variant="text"
                    title="Delete Material"
                    @click="handleDeleteMaterial(mod.uuid, page.uuid)"
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
                @edit="editQuiz(quiz)"
                @results="openQuizResults(quiz)"
                @delete="handleDeleteQuiz(quiz)"
                @take="takeQuiz(quiz)"
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
                @edit="openEditHandinDialog(asgn)"
                @submissions="openTeacherSubmissions(asgn)"
                @delete="handleDeleteHandin(asgn)"
                @submit="openStudentSubmit(asgn)"
              />
            </div>
          </v-card>
        </div>
      </div>

      <!-- TAB 2: PARTICIPANTS (Teachers only) -->
      <div v-if="activeTab === 'participants' && isTeacher">
        <div class="d-flex align-center justify-space-between mb-4">
          <h2 class="text-h5 font-weight-bold text-high-emphasis">Course Participants</h2>
          <v-btn
            color="primary"
            prepend-icon="mdi-account-plus"
            variant="elevated"
            @click="addEnrollmentDialogOpen = true"
          >
            Add Participant
          </v-btn>
        </div>

        <v-card class="bg-surface" elevation="1" rounded="lg">
          <v-table>
            <thead>
              <tr>
                <th class="text-left font-weight-bold">User Email</th>
                <th class="text-left font-weight-bold">Course Role</th>
                <th class="text-left font-weight-bold">Enrolled At</th>
                <th class="text-right font-weight-bold">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in enrollments" :key="e.userEmail">
                <td class="font-weight-medium">{{ e.userEmail }}</td>
                <td>
                  <v-chip
                    :color="e.role === 'TEACHER' ? 'primary' : 'secondary'"
                    size="small"
                    variant="flat"
                  >
                    {{ e.role }}
                  </v-chip>
                </td>
                <td class="text-medium-emphasis">
                  {{ new Date(e.enrolledAt).toLocaleDateString() }}
                </td>
                <td class="text-right">
                  <v-btn
                    v-if="e.role === 'STUDENT'"
                    color="error"
                    :disabled="!e.userUuid"
                    icon="mdi-delete-outline"
                    :loading="deletingEnrollmentUserUuid === e.userUuid"
                    size="small"
                    variant="text"
                    @click="handleRemoveEnrollment(e)"
                  />
                </td>
              </tr>
            </tbody>
          </v-table>
        </v-card>
      </div>

      <!-- TAB: ASSIGNMENTS -->
      <div v-if="activeTab === 'assignments' && isEnrolled">
        <h2 class="text-h5 font-weight-bold text-high-emphasis mb-4">Assignments</h2>
        <div v-if="assignments.length > 0" class="d-flex flex-column ga-2">
          <template v-for="asgn in assignments" :key="asgn.uuid">
            <HandinAssignmentItem
              v-if="asgn.type === 'HANDIN'"
              :assignment="asgn"
              :is-teacher="isTeacher"
              show-module
              @edit="openEditHandinDialog(asgn)"
              @submissions="openTeacherSubmissions(asgn)"
              @delete="handleDeleteHandin(asgn)"
              @submit="openStudentSubmit(asgn)"
            />
            <QuizListItem
              v-else
              :assignment="asgn"
              :is-teacher="isTeacher"
              show-module
              @edit="editQuiz(asgn)"
              @results="openQuizResults(asgn)"
              @delete="handleDeleteQuiz(asgn)"
              @take="takeQuiz(asgn)"
            />
          </template>
        </div>
        <v-card v-else class="pa-6 bg-surface mb-6 text-center" elevation="1" rounded="lg">
          <v-icon class="mb-3 text-medium-emphasis" icon="mdi-clipboard-text-outline" size="48" />
          <p class="text-body-1 text-medium-emphasis mb-0">
            Assignments will be available here once published.
          </p>
        </v-card>
      </div>

      <!-- TAB: GROUPS -->
      <div v-if="activeTab === 'groups' && isEnrolled">
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
              @click="openBulkDialog"
            >
              Bulk Create Groups
            </v-btn>
            <v-btn
              color="primary"
              prepend-icon="mdi-plus"
              variant="elevated"
              @click="openCreateGroupDialog"
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
              <div class="text-subtitle-1 font-weight-bold">
                You are a member of {{ myGroup.name }}
              </div>
              <div class="text-body-2 text-medium-emphasis">
                {{ myGroup.members.length }} / {{ myGroup.maxMembers }} members enrolled
              </div>
            </div>
            <v-btn
              v-if="!isTeacher"
              color="error"
              :loading="!!actionLoading[myGroup.uuid]"
              prepend-icon="mdi-exit-to-app"
              size="small"
              variant="tonal"
              @click="handleLeaveGroup(myGroup)"
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
        <div v-if="loadingGroups" class="d-flex justify-center py-10">
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
              @click="openBulkDialog"
            >
              Bulk Create Groups
            </v-btn>
            <v-btn
              color="secondary"
              prepend-icon="mdi-plus"
              variant="tonal"
              @click="openCreateGroupDialog"
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
              :elevation="myGroup?.uuid === group.uuid ? 3 : 1"
              rounded="lg"
              :style="
                myGroup?.uuid === group.uuid ? 'border: 2px solid rgb(var(--v-theme-primary));' : ''
              "
            >
              <div>
                <v-card-item class="pb-2">
                  <div class="d-flex align-center justify-space-between mb-2">
                    <h3
                      class="text-subtitle-1 font-weight-bold text-high-emphasis my-0 text-truncate"
                    >
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

                  <!-- Teacher action buttons in header -->
                  <div v-if="isTeacher" class="d-flex justify-end ga-1 mb-1">
                    <v-btn
                      density="comfortable"
                      icon="mdi-pencil-outline"
                      size="small"
                      title="Edit Group"
                      variant="text"
                      @click="openEditGroupDialog(group)"
                    />
                    <v-btn
                      color="error"
                      density="comfortable"
                      icon="mdi-delete-outline"
                      :loading="!!actionLoading[group.uuid]"
                      size="small"
                      title="Delete Group"
                      variant="text"
                      @click="handleDeleteGroup(group)"
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
                        v-if="
                          (currentUser?.userUuid && currentUser.userUuid === member.uuid) ||
                          (currentUser?.email &&
                            currentUser.email.toLowerCase() === member.email.toLowerCase())
                        "
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

              <!-- Card Actions -->
              <v-card-actions class="pa-4 bg-surface-variant mt-3">
                <template v-if="!isTeacher">
                  <v-btn
                    v-if="myGroup?.uuid === group.uuid"
                    block
                    color="error"
                    :loading="!!actionLoading[group.uuid]"
                    prepend-icon="mdi-exit-to-app"
                    size="small"
                    variant="tonal"
                    @click="handleLeaveGroup(group)"
                  >
                    Leave Group
                  </v-btn>
                  <v-btn
                    v-else
                    block
                    color="primary"
                    :disabled="!!myGroup || group.members.length >= group.maxMembers"
                    :loading="!!actionLoading[group.uuid]"
                    prepend-icon="mdi-account-plus"
                    size="small"
                    variant="elevated"
                    @click="handleJoinGroup(group)"
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
                      @click="openEditGroupDialog(group)"
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

      <!-- TAB 3: SETTINGS (Teachers only) -->
      <div v-if="activeTab === 'settings' && isTeacher">
        <v-card class="pa-6 bg-surface mb-6" elevation="1" rounded="lg">
          <h2 class="text-h5 font-weight-bold text-high-emphasis mb-4">Course Details</h2>

          <v-form @submit.prevent="handleUpdateCourse">
            <v-text-field
              v-model="editCourseData.title"
              class="mb-3"
              label="Course Title"
              required
              variant="outlined"
            />
            <v-textarea
              v-model="editCourseData.description"
              class="mb-3"
              label="Course Description"
              rows="3"
              variant="outlined"
            />
            <v-row dense>
              <v-col cols="6">
                <v-select
                  v-model="editCourseData.term"
                  :items="['AUTUMN', 'SPRING']"
                  label="Term"
                  variant="outlined"
                />
              </v-col>
              <v-col cols="6">
                <v-select
                  v-model="editCourseData.year"
                  :items="[2025, 2026, 2027, 2028]"
                  label="Year"
                  variant="outlined"
                />
              </v-col>
            </v-row>

            <v-btn
              class="mt-2"
              color="primary"
              :loading="savingCourse"
              size="large"
              type="submit"
              variant="elevated"
            >
              Save Changes
            </v-btn>
          </v-form>
        </v-card>

        <!-- Danger Zone -->
        <v-card class="pa-6 bg-surface border-error" elevation="0" rounded="lg">
          <h3 class="text-h6 font-weight-bold text-error mb-2">Danger Zone</h3>
          <p class="text-body-2 text-medium-emphasis mb-4">
            Deleting this course will permanently remove all modules, enrollments, and student
            submissions.
          </p>
          <v-btn
            color="error"
            :loading="deletingCourse"
            prepend-icon="mdi-delete"
            variant="outlined"
            @click="handleDeleteCourse"
          >
            Delete Course
          </v-btn>
        </v-card>
      </div>
    </div>

    <!-- Create/Edit Module Dialog -->
    <v-dialog v-model="moduleDialogOpen" max-width="500" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          {{ isEditingModule ? 'Edit Module' : 'Create Module' }}
        </v-card-title>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="moduleForm.title"
            class="mb-3"
            label="Module Title"
            placeholder="e.g. Week 1: Architecture & Patterns"
            required
            variant="outlined"
          />

          <v-select
            v-model="moduleForm.dependsOnUuid"
            clearable
            :items="availablePrerequisites"
            item-title="title"
            item-value="uuid"
            label="Prerequisite Module (Optional)"
            placeholder="None"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="moduleDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :loading="savingModule"
            variant="elevated"
            @click="handleSaveModule"
          >
            {{ isEditingModule ? 'Save' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Add Participant Dialog -->
    <v-dialog v-model="addEnrollmentDialogOpen" max-width="500" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          Add Participant
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          Enroll an existing user into this course by their registered email.
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="newEnrollmentEmail"
            class="mb-3"
            label="User Email"
            placeholder="student@hvl.no"
            required
            type="email"
            variant="outlined"
          />

          <v-select
            v-model="newEnrollmentRole"
            :items="['STUDENT', 'TEACHER']"
            label="Role"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="addEnrollmentDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :loading="addingEnrollment"
            variant="elevated"
            @click="handleAddEnrollment"
          >
            Add Participant
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Upload Material Dialog -->
    <v-dialog v-model="uploadDialogOpen" max-width="540" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          Upload Course Material
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          Upload a file (.pdf, .pptx, .md, .txt) or enter an external URL for embedded view.
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="uploadTitle"
            class="mb-3"
            label="Title (Optional for files, required for URLs)"
            placeholder="e.g. Lecture 1 Slides"
            variant="outlined"
          />

          <v-textarea
            v-model="uploadDescription"
            class="mb-3"
            label="Description / Notes (Optional)"
            placeholder="Additional context or notes"
            rows="2"
            variant="outlined"
          />

          <v-file-input
            v-model="uploadFileInput"
            class="mb-3"
            clearable
            label="Select File"
            placeholder="Choose .pdf, .docx, .pptx, image, .md, .txt"
            prepend-icon="mdi-paperclip"
            show-size
            variant="outlined"
          />

          <v-text-field
            v-model="uploadSourceUrl"
            class="mb-3"
            label="External / Iframe URL (Optional if file uploaded)"
            placeholder="https://example.com/embed"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="uploadDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :disabled="
              (!uploadFileInput && !uploadSourceUrl.trim()) ||
              (!uploadFileInput && !uploadTitle.trim())
            "
            :loading="uploading"
            variant="elevated"
            @click="handleUploadMaterial"
          >
            Upload
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Edit Material Dialog -->
    <v-dialog v-model="editMaterialDialogOpen" max-width="540" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          Edit Material Details
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          Update the title and description for this course material.
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="editMaterialTitle"
            class="mb-3"
            label="Title"
            required
            variant="outlined"
          />

          <v-textarea
            v-model="editMaterialDescription"
            class="mb-3"
            label="Description / Notes (Optional)"
            rows="3"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="editMaterialDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :disabled="!editMaterialTitle.trim()"
            :loading="savingMaterial"
            variant="elevated"
            @click="handleSaveMaterial"
          >
            Save Changes
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Create/Edit Hand-in Assignment Dialog -->
    <v-dialog v-model="handinDialogOpen" max-width="500" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          {{ isEditingHandin ? 'Edit Assignment' : 'Create Assignment' }}
        </v-card-title>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="handinTitle"
            class="mb-3"
            label="Assignment Title"
            placeholder="e.g. Lab 1: Storage & Queues"
            required
            variant="outlined"
          />

          <v-text-field
            v-model="handinDeadline"
            class="mb-3"
            label="Deadline (Optional)"
            type="datetime-local"
            variant="outlined"
          />

          <v-switch v-model="handinLateAllowed" color="primary" label="Accept late submissions" />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="handinDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :disabled="!handinTitle.trim()"
            :loading="savingHandin"
            variant="elevated"
            @click="handleSaveHandin"
          >
            {{ isEditingHandin ? 'Save' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Student Submission Dialog -->
    <HandinSubmissionDialog
      v-model="studentSubmitDialogOpen"
      :course-uuid="resolvedCourseUuid"
      :assignment="selectedHandinForSubmit"
      @submitted="loadAssignments"
    />

    <!-- Teacher Submissions Dialog -->
    <TeacherSubmissionsDialog
      v-model="teacherSubmissionsDialogOpen"
      :course-uuid="resolvedCourseUuid"
      :assignment="selectedHandinForTeacher"
    />

    <!-- Create/Edit Group Dialog -->
    <v-dialog v-model="groupDialogOpen" max-width="500" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          {{ isEditingGroup ? 'Edit Group' : 'Create Group' }}
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          {{
            isEditingGroup
              ? 'Update the group name and maximum member capacity.'
              : 'Add a new group for students to collaborate and submit assignments.'
          }}
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="groupForm.name"
            class="mb-3"
            label="Group Name"
            placeholder="e.g. Group 1 or Team Alpha"
            required
            variant="outlined"
          />

          <v-text-field
            v-model.number="groupForm.maxMembers"
            class="mb-3"
            hint="Maximum number of students permitted in this group (at least 1)"
            label="Maximum Members"
            max="100"
            min="1"
            persistent-hint
            required
            type="number"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="groupDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn color="primary" :loading="savingGroup" variant="elevated" @click="handleSaveGroup">
            {{ isEditingGroup ? 'Save Changes' : 'Create Group' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Bulk Create Groups Dialog -->
    <v-dialog v-model="bulkDialogOpen" max-width="540" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          Bulk Create Groups
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          Instantly generate multiple numbered groups with minimal clicks.
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="bulkForm.prefix"
            class="mb-3"
            hint="Prefix for group names (e.g. 'Group' produces 'Group 1', 'Group 2', ...)"
            label="Group Prefix"
            persistent-hint
            placeholder="Group"
            variant="outlined"
          />

          <div class="mb-3">
            <label class="text-caption font-weight-bold text-medium-emphasis mb-1 d-block">
              Quick Count Preset:
            </label>
            <div class="d-flex ga-2">
              <v-btn
                v-for="cnt in [5, 10, 15, 20]"
                :key="cnt"
                :color="bulkForm.count === cnt ? 'primary' : 'secondary'"
                size="small"
                :variant="bulkForm.count === cnt ? 'elevated' : 'tonal'"
                @click="bulkForm.count = cnt"
              >
                {{ cnt }} Groups
              </v-btn>
            </div>
          </div>

          <v-row class="mb-2" dense>
            <v-col cols="6">
              <v-text-field
                v-model.number="bulkForm.count"
                label="Number of Groups"
                max="50"
                min="1"
                required
                type="number"
                variant="outlined"
              />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="bulkForm.maxMembers"
                label="Capacity per Group"
                max="100"
                min="1"
                required
                type="number"
                variant="outlined"
              />
            </v-col>
          </v-row>

          <v-alert
            class="text-caption"
            color="info"
            density="compact"
            icon="mdi-information-outline"
            variant="tonal"
          >
            <div>
              <strong>Preview:</strong>
              {{ bulkForm.prefix || 'Group' }} 1, {{ bulkForm.prefix || 'Group' }} 2
              <template v-if="bulkForm.count > 3">
                ... {{ bulkForm.prefix || 'Group' }} {{ bulkForm.count }}
              </template>
              <template v-else-if="bulkForm.count === 3">
                , {{ bulkForm.prefix || 'Group' }} 3
              </template>
              ({{ bulkForm.maxMembers }} students each)
            </div>
            <div class="mt-1 text-medium-emphasis">
              Numbers that already exist will be automatically skipped.
            </div>
          </v-alert>
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="bulkDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :loading="savingBulkGroups"
            prepend-icon="mdi-plus-box-multiple"
            variant="elevated"
            @click="handleBulkCreateGroups"
          >
            Create {{ bulkForm.count }} Groups
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script lang="ts" setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import {
  getCourse,
  getMyCourses,
  selfEnroll,
  listModules,
  getCourseEnrollments,
  updateModule,
  createModule,
  reorderModules,
  deleteModule,
  addEnrollment,
  removeEnrollment,
  updateCourse,
  deleteCourse,
  listCourseGroups,
  createGroup,
  bulkCreateGroups,
  updateGroup,
  deleteGroup,
  joinGroup,
  leaveGroup,
} from '@/api/course'
import { listFiles, uploadFile, updateFile, deleteFile } from '@/api/content'
import { createQuiz, deleteQuiz, listAssignments } from '@/api/quiz'
import { createHandin, updateHandin, deleteHandin } from '@/api/submission'
import QuizListItem from '@/components/QuizListItem.vue'
import HandinAssignmentItem from '@/components/HandinAssignmentItem.vue'
import HandinSubmissionDialog from '@/components/HandinSubmissionDialog.vue'
import TeacherSubmissionsDialog from '@/components/TeacherSubmissionsDialog.vue'
import { isAxiosError } from 'axios'
import { getApiErrorMessage } from '@/api/client'
import type { Course, Module, Enrollment, UpdateCoursePayload, Group } from '@/types/course'
import type { FileUpload } from '@/types/content'
import type { Assignment } from '@/types/quiz'

const props = defineProps<{
  courseUuid?: string
}>()

const emit = defineEmits<{
  (e: 'course-deleted'): void
}>()

const route = useRoute()
const router = useRouter()
const { hasRole, currentUser } = useAuth()
const toast = useToast()

const resolvedCourseUuid = computed(() => props.courseUuid || String(route.params.uuid || ''))

const activeTab = ref('modules')
const loading = ref(false)
const course = ref<Course | undefined>(undefined)
const modules = ref<Module[]>([])
const enrollments = ref<Enrollment[]>([])
const userRole = ref<'STUDENT' | 'TEACHER' | undefined>(undefined)

const isTeacher = computed(
  () => userRole.value === 'TEACHER' || hasRole('admin') || hasRole('superadmin'),
)
const isEnrolled = computed(() => !!userRole.value || hasRole('admin') || hasRole('superadmin'))
const enrolling = ref(false)

watch(activeTab, newTab => {
  if (newTab !== 'overview' && !isEnrolled.value) {
    activeTab.value = 'overview'
    toast.info('You must be enrolled in this course to access course content.')
  }
})

// Settings edit form
const editCourseData = ref<UpdateCoursePayload>({
  title: '',
  description: '',
  term: 'AUTUMN',
  year: 2026,
})
const savingCourse = ref(false)
const deletingCourse = ref(false)

// Module dialog form
const moduleDialogOpen = ref(false)
const isEditingModule = ref(false)
const editingModuleUuid = ref<string | undefined>(undefined)
const moduleForm = ref<{ title: string; position?: number; dependsOnUuid?: string }>({
  title: '',
  position: undefined,
  dependsOnUuid: undefined,
})
const savingModule = ref(false)

// Participant dialog
const addEnrollmentDialogOpen = ref(false)
const newEnrollmentEmail = ref('')
const newEnrollmentRole = ref<'STUDENT' | 'TEACHER'>('STUDENT')
const addingEnrollment = ref(false)
const deletingEnrollmentUserUuid = ref<string | undefined>(undefined)

// Group management state
const groups = ref<Group[]>([])
const loadingGroups = ref(false)
const groupDialogOpen = ref(false)
const isEditingGroup = ref(false)
const editingGroupUuid = ref<string | undefined>(undefined)
const groupForm = ref<{ name: string; maxMembers: number }>({
  name: '',
  maxMembers: 4,
})
const savingGroup = ref(false)

const bulkDialogOpen = ref(false)
const bulkForm = ref<{ prefix: string; count: number; maxMembers: number }>({
  prefix: 'Group',
  count: 10,
  maxMembers: 4,
})
const savingBulkGroups = ref(false)
const actionLoading = ref<Record<string, boolean>>({})

const myGroup = computed(() => {
  if (!currentUser.value) return undefined
  return groups.value.find(g =>
    g.members?.some(
      m =>
        (currentUser.value?.userUuid && m.uuid === currentUser.value.userUuid) ||
        (currentUser.value?.email &&
          m.email.toLowerCase() === currentUser.value.email.toLowerCase()),
    ),
  )
})

// Course materials state
const modulePages = ref<Record<string, FileUpload[]>>({})
const lockedModules = ref<Record<string, boolean>>({})
const assignments = ref<Assignment[]>([])
const uploadDialogOpen = ref(false)
const targetModuleUuid = ref<string>('')
const uploadFileInput = ref<File | File[] | undefined>(undefined)
const uploadTitle = ref('')
const uploadDescription = ref('')
const uploadSourceUrl = ref('')
const uploading = ref(false)

// Edit material state
const editMaterialDialogOpen = ref(false)
const editMaterialTargetModUuid = ref('')
const editMaterialTargetPageUuid = ref('')
const editMaterialTitle = ref('')
const editMaterialDescription = ref('')
const savingMaterial = ref(false)

// Hand-in assignment create/edit state
const handinDialogOpen = ref(false)
const isEditingHandin = ref(false)
const targetHandinModUuid = ref('')
const targetHandinUuid = ref('')
const handinTitle = ref('')
const handinDeadline = ref('')
const handinLateAllowed = ref(false)
const savingHandin = ref(false)

// Student submit dialog state
const studentSubmitDialogOpen = ref(false)
const selectedHandinForSubmit = ref<Assignment | undefined>(undefined)

// Teacher submissions dialog state
const teacherSubmissionsDialogOpen = ref(false)
const selectedHandinForTeacher = ref<Assignment | undefined>(undefined)

const availablePrerequisites = computed(() => {
  const disallowed = new Set<string>()
  if (isEditingModule.value && editingModuleUuid.value) {
    disallowed.add(editingModuleUuid.value)
    let added = true
    while (added) {
      added = false
      for (const m of modules.value) {
        if (m.dependsOnUuid && disallowed.has(m.dependsOnUuid) && !disallowed.has(m.uuid)) {
          disallowed.add(m.uuid)
          added = true
        }
      }
    }
  }
  return modules.value.filter(m => !disallowed.has(m.uuid))
})

async function loadAssignments() {
  try {
    assignments.value = await listAssignments(resolvedCourseUuid.value)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load assignments.')
  }
}

function quizzesIn(moduleUuid: string): Assignment[] {
  return assignments.value.filter(a => a.moduleUuid === moduleUuid && a.type !== 'HANDIN')
}

function handinsIn(moduleUuid: string): Assignment[] {
  return assignments.value.filter(a => a.moduleUuid === moduleUuid && a.type === 'HANDIN')
}

async function handleAddQuiz(moduleUuid: string) {
  try {
    const quiz = await createQuiz(resolvedCourseUuid.value, moduleUuid, 'New quiz')
    editQuiz({ uuid: quiz.uuid, moduleUuid })
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to create quiz.')
  }
}

function quizRoute(quiz: Pick<Assignment, 'uuid' | 'moduleUuid'>, page: string): string {
  return `/courses/${resolvedCourseUuid.value}/modules/${quiz.moduleUuid}/quizzes/${quiz.uuid}/${page}`
}

function editQuiz(quiz: Pick<Assignment, 'uuid' | 'moduleUuid'>) {
  router.push(quizRoute(quiz, 'edit'))
}

function openQuizResults(quiz: Assignment) {
  router.push(quizRoute(quiz, 'results'))
}

function takeQuiz(quiz: Assignment) {
  router.push(quizRoute(quiz, 'take'))
}

async function handleDeleteQuiz(quiz: Assignment) {
  if (!confirm(`Delete quiz "${quiz.title}" and all student attempts?`)) return
  try {
    await deleteQuiz(resolvedCourseUuid.value, quiz.uuid)
    toast.success('Quiz deleted.')
    await loadAssignments()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete quiz.')
  }
}

function openCreateHandinDialog(moduleUuid: string) {
  isEditingHandin.value = false
  targetHandinModUuid.value = moduleUuid
  targetHandinUuid.value = ''
  handinTitle.value = ''
  handinDeadline.value = ''
  handinLateAllowed.value = false
  handinDialogOpen.value = true
}

function openEditHandinDialog(asgn: Assignment) {
  isEditingHandin.value = true
  targetHandinModUuid.value = asgn.moduleUuid
  targetHandinUuid.value = asgn.uuid
  handinTitle.value = asgn.title
  handinDeadline.value = asgn.deadline ? asgn.deadline.slice(0, 16) : ''
  handinLateAllowed.value = asgn.acceptsLateSubmissions || false
  handinDialogOpen.value = true
}

async function handleSaveHandin() {
  if (!handinTitle.value.trim() || !resolvedCourseUuid.value) return
  savingHandin.value = true
  try {
    const deadlineIso = handinDeadline.value
      ? new Date(handinDeadline.value).toISOString()
      : undefined
    if (isEditingHandin.value && targetHandinUuid.value) {
      await updateHandin(
        resolvedCourseUuid.value,
        targetHandinModUuid.value,
        targetHandinUuid.value,
        {
          title: handinTitle.value.trim(),
          deadline: deadlineIso,
          acceptsLateSubmissions: handinLateAllowed.value,
        },
      )
      toast.success('Assignment updated successfully.')
    } else {
      await createHandin(resolvedCourseUuid.value, targetHandinModUuid.value, {
        title: handinTitle.value.trim(),
        deadline: deadlineIso,
        acceptsLateSubmissions: handinLateAllowed.value,
      })
      toast.success('Assignment created successfully.')
    }
    handinDialogOpen.value = false
    await loadAssignments()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to save assignment.')
  } finally {
    savingHandin.value = false
  }
}

async function handleDeleteHandin(asgn: Assignment) {
  if (!confirm(`Delete assignment "${asgn.title}"?`)) return
  try {
    await deleteHandin(resolvedCourseUuid.value, asgn.moduleUuid, asgn.uuid)
    toast.success('Assignment deleted.')
    await loadAssignments()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete assignment.')
  }
}

function openStudentSubmit(asgn: Assignment) {
  selectedHandinForSubmit.value = asgn
  studentSubmitDialogOpen.value = true
}

function openTeacherSubmissions(asgn: Assignment) {
  selectedHandinForTeacher.value = asgn
  teacherSubmissionsDialogOpen.value = true
}

async function loadAllModulePages() {
  if (!resolvedCourseUuid.value || modules.value.length === 0) return
  await loadAssignments()
  const pagesMap: Record<string, FileUpload[]> = {}
  const lockedMap: Record<string, boolean> = {}
  let hasError = false
  await Promise.all(
    modules.value.map(async mod => {
      try {
        pagesMap[mod.uuid] = await listFiles(resolvedCourseUuid.value, mod.uuid)
      } catch (err: unknown) {
        pagesMap[mod.uuid] = []
        const errMsg = getApiErrorMessage(err, '')
        const is403 = isAxiosError(err) && err.response?.status === 403
        if (
          (is403 && (errMsg.toLowerCase().includes('prerequisite') || !!mod.dependsOn)) ||
          errMsg.toLowerCase().includes('prerequisite')
        ) {
          lockedMap[mod.uuid] = true
        } else {
          hasError = true
        }
      }
    }),
  )
  modulePages.value = pagesMap
  lockedModules.value = lockedMap
  if (hasError) {
    toast.error('Failed to load materials for one or more modules.')
  }
}

function openUploadDialog(modUuid: string) {
  targetModuleUuid.value = modUuid
  uploadFileInput.value = undefined
  uploadTitle.value = ''
  uploadDescription.value = ''
  uploadSourceUrl.value = ''
  uploadDialogOpen.value = true
}

function openEditMaterialDialog(modUuid: string, page: FileUpload) {
  editMaterialTargetModUuid.value = modUuid
  editMaterialTargetPageUuid.value = page.uuid
  editMaterialTitle.value = page.title
  editMaterialDescription.value = page.description || ''
  editMaterialDialogOpen.value = true
}

async function handleSaveMaterial() {
  if (
    !editMaterialTitle.value.trim() ||
    !editMaterialTargetPageUuid.value ||
    !resolvedCourseUuid.value
  )
    return
  savingMaterial.value = true
  try {
    await updateFile(
      resolvedCourseUuid.value,
      editMaterialTargetModUuid.value,
      editMaterialTargetPageUuid.value,
      {
        title: editMaterialTitle.value.trim(),
        description: editMaterialDescription.value.trim() || undefined,
      },
    )
    toast.success('Course material updated successfully.')
    editMaterialDialogOpen.value = false
    modulePages.value[editMaterialTargetModUuid.value] = await listFiles(
      resolvedCourseUuid.value,
      editMaterialTargetModUuid.value,
    )
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to update course material.')
  } finally {
    savingMaterial.value = false
  }
}

async function handleUploadMaterial() {
  if (!targetModuleUuid.value || !resolvedCourseUuid.value) return
  const rawFile = Array.isArray(uploadFileInput.value)
    ? uploadFileInput.value[0]
    : uploadFileInput.value

  if (!rawFile && !uploadSourceUrl.value.trim()) {
    toast.error('Please select a file or provide a URL.')
    return
  }
  if (!rawFile && !uploadTitle.value.trim()) {
    toast.error('Title is required when adding an external link.')
    return
  }

  uploading.value = true
  try {
    await uploadFile(resolvedCourseUuid.value, targetModuleUuid.value, {
      file: rawFile || undefined,
      title: uploadTitle.value.trim() || undefined,
      description: uploadDescription.value.trim() || undefined,
      sourceUrl: uploadSourceUrl.value.trim() || undefined,
    })
    toast.success('Course material uploaded successfully.')
    uploadDialogOpen.value = false
    modulePages.value[targetModuleUuid.value] = await listFiles(
      resolvedCourseUuid.value,
      targetModuleUuid.value,
    )
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to upload course material.')
  } finally {
    uploading.value = false
  }
}

async function handleDeleteMaterial(moduleUuid: string, pageUuid: string) {
  if (!confirm('Are you sure you want to delete this course material?')) return
  try {
    await deleteFile(resolvedCourseUuid.value, moduleUuid, pageUuid)
    toast.success('Course material deleted.')
    modulePages.value[moduleUuid] = await listFiles(resolvedCourseUuid.value, moduleUuid)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete course material.')
  }
}

function viewMaterial(moduleUuid: string, pageUuid: string) {
  router.push(`/courses/${resolvedCourseUuid.value}/modules/${moduleUuid}/files/${pageUuid}`)
}

function getPageFormatIcon(page: FileUpload): string {
  const name = page.fileName?.toLowerCase() || ''
  if (page.contentType?.includes('pdf') || name.endsWith('.pdf')) return 'mdi-file-pdf-box'
  if (page.contentType?.includes('word') || name.endsWith('.docx') || name.endsWith('.doc')) {
    return 'mdi-file-word-box'
  }
  if (
    page.contentType?.includes('presentation') ||
    page.contentType?.includes('powerpoint') ||
    name.endsWith('.pptx') ||
    name.endsWith('.ppt')
  ) {
    return 'mdi-file-powerpoint-box'
  }
  if (
    page.contentType?.startsWith('image/') ||
    ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp'].some(ext => name.endsWith(ext))
  ) {
    return 'mdi-file-image'
  }
  if (name.endsWith('.md')) return 'mdi-language-markdown'
  if (page.contentType?.includes('text') || name.endsWith('.txt'))
    return 'mdi-file-document-outline'
  if (page.sourceUrl) return 'mdi-link-variant'
  return 'mdi-file-outline'
}

function getPageFormatColor(page: FileUpload): string {
  const name = page.fileName?.toLowerCase() || ''
  if (page.contentType?.includes('pdf') || name.endsWith('.pdf')) return 'error'
  if (page.contentType?.includes('word') || name.endsWith('.docx') || name.endsWith('.doc')) {
    return 'primary'
  }
  if (
    page.contentType?.includes('presentation') ||
    page.contentType?.includes('powerpoint') ||
    name.endsWith('.pptx') ||
    name.endsWith('.ppt')
  ) {
    return 'warning'
  }
  if (
    page.contentType?.startsWith('image/') ||
    ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp'].some(ext => name.endsWith(ext))
  ) {
    return 'info'
  }
  if (name.endsWith('.md') || page.contentType?.includes('text') || name.endsWith('.txt'))
    return 'primary'
  if (page.sourceUrl) return 'info'
  return 'secondary'
}

function formatBytes(bytes?: number): string {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(isoDate?: string): string {
  if (!isoDate) return ''
  return new Date(isoDate).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

async function fetchCourseDetails() {
  if (!resolvedCourseUuid.value) return
  loading.value = true
  try {
    course.value = await getCourse(resolvedCourseUuid.value)
    editCourseData.value = {
      title: course.value.title,
      description: course.value.description,
      term: course.value.term,
      year: course.value.year,
    }

    // Determine current user's role in this course
    const myCourses = await getMyCourses().catch(() => [])
    const mine = myCourses.find(mc => mc.course.uuid === resolvedCourseUuid.value)
    userRole.value = mine ? mine.role : undefined

    if (isEnrolled.value) {
      // Load modules
      modules.value = await listModules(resolvedCourseUuid.value)
      await loadAllModulePages()

      // Load enrollments if teacher
      if (isTeacher.value) {
        enrollments.value = await getCourseEnrollments(resolvedCourseUuid.value).catch(() => [])
      }

      // Load groups
      await loadGroups()
    } else {
      activeTab.value = 'overview'
      modules.value = []
      groups.value = []
    }
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load course details.')
  } finally {
    loading.value = false
  }
}

async function handleSelfEnroll() {
  enrolling.value = true
  try {
    await selfEnroll(resolvedCourseUuid.value)
    toast.success('Successfully enrolled in the course!')
    await fetchCourseDetails()
    activeTab.value = 'modules'
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to enroll in course.')
  } finally {
    enrolling.value = false
  }
}

function openCreateModuleDialog() {
  isEditingModule.value = false
  editingModuleUuid.value = undefined
  moduleForm.value = { title: '', position: undefined, dependsOnUuid: undefined }
  moduleDialogOpen.value = true
}

function openEditModuleDialog(mod: Module) {
  isEditingModule.value = true
  editingModuleUuid.value = mod.uuid
  moduleForm.value = { title: mod.title, position: mod.position, dependsOnUuid: mod.dependsOnUuid }
  moduleDialogOpen.value = true
}

async function handleSaveModule() {
  if (!moduleForm.value.title.trim()) return
  savingModule.value = true
  try {
    if (isEditingModule.value && editingModuleUuid.value) {
      await updateModule(resolvedCourseUuid.value, editingModuleUuid.value, {
        title: moduleForm.value.title,
        position: moduleForm.value.position,
        dependsOnUuid: moduleForm.value.dependsOnUuid,
      })
      toast.success('Module updated successfully.')
    } else {
      await createModule(resolvedCourseUuid.value, {
        title: moduleForm.value.title,
        dependsOnUuid: moduleForm.value.dependsOnUuid,
      })
      toast.success('Module created successfully.')
    }
    moduleDialogOpen.value = false
    modules.value = await listModules(resolvedCourseUuid.value)
    await loadAllModulePages()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to save module.')
  } finally {
    savingModule.value = false
  }
}

async function moveModule(index: number, direction: -1 | 1) {
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= modules.value.length) return

  const reordered = [...modules.value]
  const temp = reordered[index]
  reordered[index] = reordered[targetIndex]
  reordered[targetIndex] = temp

  modules.value = reordered
  try {
    await reorderModules(
      resolvedCourseUuid.value,
      reordered.map(m => m.uuid),
    )
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to reorder modules.')
    modules.value = await listModules(resolvedCourseUuid.value)
  }
}

async function handleDeleteModule(moduleUuid: string) {
  if (!confirm('Are you sure you want to delete this module?')) return
  try {
    await deleteModule(resolvedCourseUuid.value, moduleUuid)
    toast.success('Module deleted.')
    modules.value = await listModules(resolvedCourseUuid.value)
    await loadAllModulePages()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete module.')
  }
}

async function handleAddEnrollment() {
  if (!newEnrollmentEmail.value.trim()) return
  addingEnrollment.value = true
  try {
    await addEnrollment(resolvedCourseUuid.value, {
      email: newEnrollmentEmail.value,
      role: newEnrollmentRole.value,
    })
    addEnrollmentDialogOpen.value = false
    newEnrollmentEmail.value = ''
    toast.success('Participant enrolled successfully.')
    enrollments.value = await getCourseEnrollments(resolvedCourseUuid.value)
    course.value = await getCourse(resolvedCourseUuid.value)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to add participant.')
  } finally {
    addingEnrollment.value = false
  }
}

async function handleRemoveEnrollment(enrollment: Enrollment) {
  if (!enrollment.userUuid || enrollment.role !== 'STUDENT') return
  if (!confirm(`Are you sure you want to remove ${enrollment.userEmail} from this course?`)) return
  deletingEnrollmentUserUuid.value = enrollment.userUuid
  try {
    await removeEnrollment(resolvedCourseUuid.value, enrollment.userUuid)
    toast.success(`Removed ${enrollment.userEmail} from course.`)
    enrollments.value = await getCourseEnrollments(resolvedCourseUuid.value)
    course.value = await getCourse(resolvedCourseUuid.value)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to remove participant.')
  } finally {
    deletingEnrollmentUserUuid.value = undefined
  }
}

async function handleUpdateCourse() {
  savingCourse.value = true
  try {
    const updated = await updateCourse(resolvedCourseUuid.value, editCourseData.value)
    course.value = updated
    toast.success('Course details updated successfully.')
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to update course.')
  } finally {
    savingCourse.value = false
  }
}

async function handleDeleteCourse() {
  if (
    !confirm(
      'Are you ABSOLUTELY sure you want to delete this course? This action cannot be undone.',
    )
  )
    return
  deletingCourse.value = true
  try {
    await deleteCourse(resolvedCourseUuid.value)
    toast.success('Course deleted.')
    emit('course-deleted')
    router.push('/courses')
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete course.')
  } finally {
    deletingCourse.value = false
  }
}

async function loadGroups() {
  if (!resolvedCourseUuid.value) return
  loadingGroups.value = true
  try {
    groups.value = await listCourseGroups(resolvedCourseUuid.value)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load course groups.')
  } finally {
    loadingGroups.value = false
  }
}

function openCreateGroupDialog() {
  isEditingGroup.value = false
  editingGroupUuid.value = undefined
  groupForm.value = {
    name: `Group ${groups.value.length + 1}`,
    maxMembers: 4,
  }
  groupDialogOpen.value = true
}

function openEditGroupDialog(group: Group) {
  isEditingGroup.value = true
  editingGroupUuid.value = group.uuid
  groupForm.value = {
    name: group.name,
    maxMembers: group.maxMembers,
  }
  groupDialogOpen.value = true
}

async function handleSaveGroup() {
  if (!groupForm.value.name.trim()) {
    toast.error('Group name is required.')
    return
  }
  if (groupForm.value.maxMembers < 1) {
    toast.error('Maximum members must be at least 1.')
    return
  }

  savingGroup.value = true
  try {
    if (isEditingGroup.value && editingGroupUuid.value) {
      await updateGroup(editingGroupUuid.value, {
        name: groupForm.value.name.trim(),
        maxMembers: Number(groupForm.value.maxMembers),
      })
      toast.success('Group updated successfully.')
    } else {
      await createGroup({
        courseUuid: resolvedCourseUuid.value,
        name: groupForm.value.name.trim(),
        maxMembers: Number(groupForm.value.maxMembers),
      })
      toast.success('Group created successfully.')
    }
    groupDialogOpen.value = false
    await loadGroups()
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to save group.'))
  } finally {
    savingGroup.value = false
  }
}

function openBulkDialog() {
  bulkForm.value = {
    prefix: 'Group',
    count: 10,
    maxMembers: 4,
  }
  bulkDialogOpen.value = true
}

async function handleBulkCreateGroups() {
  if (bulkForm.value.count < 1 || bulkForm.value.count > 50) {
    toast.error('Count must be between 1 and 50.')
    return
  }
  if (bulkForm.value.maxMembers < 1) {
    toast.error('Maximum members must be at least 1.')
    return
  }

  savingBulkGroups.value = true
  try {
    const created = await bulkCreateGroups({
      courseUuid: resolvedCourseUuid.value,
      prefix: bulkForm.value.prefix.trim() || 'Group',
      count: Number(bulkForm.value.count),
      maxMembers: Number(bulkForm.value.maxMembers),
    })
    toast.success(`Successfully created ${created.length} groups!`)
    bulkDialogOpen.value = false
    await loadGroups()
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to bulk-create groups.'))
  } finally {
    savingBulkGroups.value = false
  }
}

async function handleDeleteGroup(group: Group) {
  if (!confirm(`Are you sure you want to delete "${group.name}"? Members will be unassigned.`)) {
    return
  }
  actionLoading.value[group.uuid] = true
  try {
    await deleteGroup(group.uuid)
    toast.success(`Group "${group.name}" deleted.`)
    await loadGroups()
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to delete group.'))
  } finally {
    delete actionLoading.value[group.uuid]
  }
}

async function handleJoinGroup(group: Group) {
  actionLoading.value[group.uuid] = true
  try {
    await joinGroup(group.uuid)
    toast.success(`Joined "${group.name}"!`)
    await loadGroups()
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to join group.'))
  } finally {
    delete actionLoading.value[group.uuid]
  }
}

async function handleLeaveGroup(group: Group) {
  if (!confirm(`Are you sure you want to leave "${group.name}"?`)) {
    return
  }
  actionLoading.value[group.uuid] = true
  try {
    await leaveGroup(group.uuid)
    toast.success(`Left "${group.name}".`)
    await loadGroups()
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to leave group.'))
  } finally {
    delete actionLoading.value[group.uuid]
  }
}

onMounted(() => {
  fetchCourseDetails()
})
</script>
