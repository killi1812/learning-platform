<template>
  <v-container class="py-6" max-width="1400">
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <div>
        <h1 class="text-h4 font-weight-bold text-high-emphasis">Dashboard</h1>
        <p class="text-body-2 text-medium-emphasis mt-1">Your active courses overview</p>
      </div>

      <v-btn color="primary" prepend-icon="mdi-compass-outline" to="/catalog" variant="tonal">
        Browse Catalog
      </v-btn>
    </div>

    <EnrolledCoursesGrid
      :courses="myCourses"
      :error="loadError"
      :loading="loading"
      @retry="fetchMyCourses"
    />
  </v-container>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import EnrolledCoursesGrid from '@/components/EnrolledCoursesGrid.vue'
import { useMyCourses } from '@/composables/useMyCourses'

const { myCourses, loading, loadError, fetchMyCourses } = useMyCourses()

onMounted(() => {
  void fetchMyCourses()
})
</script>
