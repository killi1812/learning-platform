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
              <v-progress-linear :value="course.progress" color="primary" />
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

<style lang="scss" scoped>
.dashboard {
  padding: 2rem;
  max-width: 1400px;
  margin: 0 auto;
}

.campus-header {
  margin-bottom: 3rem;
  padding-bottom: 2rem;
  border-bottom: 2px solid #e0e0e0;

  h1 {
    margin: 0 0 0.5rem 0;
    font-size: 2.5rem;
    font-weight: 600;
    color: #1f1f1f;
  }

  .campus-description {
    margin: 0;
    font-size: 1.1rem;
    color: #666;
    line-height: 1.5;
  }
}

.courses-section {
  h2 {
    margin: 0 0 1.5rem 0;
    font-size: 1.5rem;
    font-weight: 500;
    color: #1f1f1f;
  }
}

.courses-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.5rem;
}

.course-card {
  transition: all 0.2s ease;
  cursor: pointer;

  &:hover {
    elevation: 4;
  }

  .course-title {
    font-weight: 600;
    font-size: 1.1rem;
    padding-bottom: 0.5rem;
  }

  .course-instructor {
    margin: 0 0 0.5rem 0;
    font-size: 0.95rem;
    color: #666;
  }

  .course-term {
    margin: 0 0 1rem 0;
    font-size: 0.9rem;
    color: #999;
  }

  .course-progress {
    .progress-label {
      display: block;
      font-size: 0.85rem;
      color: #666;
      margin-bottom: 0.5rem;
    }
  }
}
</style>
