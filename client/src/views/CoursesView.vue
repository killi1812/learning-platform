<template>
  <v-container class="py-6">
    <div class="d-flex align-center justify-space-between mb-6">
      <div>
        <h1 class="text-h4 font-weight-bold">My Courses</h1>
        <p class="text-medium-emphasis">Enrolled courses for the current semester</p>
      </div>
      <v-btn color="primary" prepend-icon="mdi-plus">Explore Courses</v-btn>
    </div>

    <v-row>
      <v-col v-for="course in courseList" :key="course.id" cols="12" md="6" lg="4">
        <v-card elevation="2" rounded="lg">
          <v-card-item>
            <template #prepend>
              <v-avatar color="primary" rounded="lg">
                <v-icon color="white" icon="mdi-book-open-page-variant" />
              </v-avatar>
            </template>
            <v-card-title class="font-weight-bold">{{ course.code }}</v-card-title>
            <v-card-subtitle>{{ course.name }}</v-card-subtitle>
          </v-card-item>
          <v-card-text>
            <div class="d-flex justify-space-between text-caption mb-1">
              <span>Progress</span>
              <span>{{ course.progress }}%</span>
            </div>
            <v-progress-linear :model-value="course.progress" color="primary" height="6" rounded />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const courseList = ref([
  { id: 1, code: 'DAT250', name: 'Advanced Software Technologies', progress: 75 },
  { id: 2, code: 'DAT251', name: 'Modern Systems Architecture', progress: 40 },
  { id: 3, code: 'INF100', name: 'Introduction to Programming', progress: 100 },
])
</script>
