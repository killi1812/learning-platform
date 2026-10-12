<template>
  <div class="dashboard">
    <!-- Campus Header -->
    <div class="campus-header">
      <h1>University of Bergen</h1>
      <p class="campus-description">
        A comprehensive learning management system for courses and student engagement.
      </p>
    </div>

    <!-- Courses Section -->
    <div class="courses-section">
      <h2>My Courses</h2>
      <div class="courses-grid">
        <v-card v-for="course in courses" :key="course.id" class="course-card" elevation="1">
          <v-card-title class="course-title">{{ course.title }}</v-card-title>
          <v-card-text>
            <p class="course-instructor">{{ course.instructor }}</p>
            <p class="course-term">{{ course.term }}</p>
            <div class="course-progress">
              <span class="progress-label">Progress</span>
              <v-progress-linear :model-value="course.progress" color="primary" />
            </div>
          </v-card-text>
          <v-card-actions>
            <v-chip size="small" variant="outlined">{{ course.status }}</v-chip>
          </v-card-actions>
        </v-card>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

interface Course {
  id: number
  title: string
  instructor: string
  term: string
  progress: number
  status: string
}

const courses = ref<Course[]>([
  {
    id: 1,
    title: 'Data Structures & Algorithms',
    instructor: 'Dr. Smith',
    term: 'Autumn 2026',
    progress: 65,
    status: 'Active',
  },
  {
    id: 2,
    title: 'Web Development Fundamentals',
    instructor: 'Prof. Johnson',
    term: 'Autumn 2026',
    progress: 45,
    status: 'Active',
  },
  {
    id: 3,
    title: 'Database Systems',
    instructor: 'Dr. Williams',
    term: 'Autumn 2026',
    progress: 80,
    status: 'Active',
  },
  {
    id: 4,
    title: 'Software Engineering',
    instructor: 'Prof. Brown',
    term: 'Spring 2026',
    progress: 100,
    status: 'Completed',
  },
  {
    id: 5,
    title: 'Machine Learning Basics',
    instructor: 'Dr. Davis',
    term: 'Autumn 2026',
    progress: 30,
    status: 'Active',
  },
  {
    id: 6,
    title: 'Cloud Computing',
    instructor: 'Prof. Miller',
    term: 'Autumn 2026',
    progress: 55,
    status: 'Active',
  },
])
</script>
