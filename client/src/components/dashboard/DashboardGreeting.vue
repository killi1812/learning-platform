<template>
  <div>
    <h1 class="text-h4 font-weight-bold text-high-emphasis">{{ greeting }}</h1>
    <p class="text-body-2 text-medium-emphasis mt-1">{{ dateLabel }} · {{ summary }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  username?: string
  now: Date
  upcomingCount: number
  /** Ungraded submissions waiting for the teacher. Undefined for non-teachers. */
  pendingCount?: number
}>()

function dayPart(hour: number): string {
  if (hour < 12) {
    return 'Good morning'
  }
  return hour < 18 ? 'Good afternoon' : 'Good evening'
}

const greeting = computed(() => {
  const part = dayPart(props.now.getHours())
  return props.username ? `${part}, ${props.username}` : part
})

const dateLabel = computed(() =>
  props.now.toLocaleDateString(undefined, { weekday: 'long', day: 'numeric', month: 'long' }),
)

const summary = computed(() => {
  const parts = [`${props.upcomingCount} deadline${props.upcomingCount === 1 ? '' : 's'} this week`]
  if (props.pendingCount !== undefined) {
    parts.push(`${props.pendingCount} to grade`)
  }
  return parts.join(' · ')
})
</script>
