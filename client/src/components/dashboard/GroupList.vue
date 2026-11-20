<template>
  <v-list class="pa-0 bg-transparent" density="compact">
    <v-list-item v-for="entry in groups" :key="entry.group.uuid" class="px-0" to="/groups">
      <v-list-item-title class="font-weight-medium">{{ entry.group.name }}</v-list-item-title>
      <v-list-item-subtitle>{{ entry.courseTitle }}</v-list-item-subtitle>
      <template #append>
        <v-chip size="small" variant="tonal">
          {{ entry.group.members.length }} / {{ entry.group.maxMembers }} members
        </v-chip>
      </template>
    </v-list-item>
  </v-list>
</template>

<script setup lang="ts">
import type { Group } from '@/types/course'

defineProps<{ groups: readonly { courseTitle: string; group: Group }[] }>()
</script>
