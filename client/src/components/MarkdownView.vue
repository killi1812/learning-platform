<template>
  <div class="markdown-view">
    <div
      v-if="htmlContent"
      class="markdown-view__content"
      tabindex="0"
      aria-label="Markdown content"
      v-html="htmlContent"
    />
    <div v-else class="markdown-view__empty text-medium-emphasis text-caption font-italic">
      {{ emptyText }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '@/util/markdown'

const props = withDefaults(
  defineProps<{
    source?: string
    emptyText?: string
  }>(),
  {
    source: '',
    emptyText: 'No content provided.',
  },
)

const htmlContent = computed(() => renderMarkdown(props.source))
</script>

<style scoped>
.markdown-view {
  width: 100%;
}

.markdown-view__content :deep(p) {
  margin-bottom: 0.75rem;
  line-height: 1.6;
}

.markdown-view__content :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-view__content :deep(h1),
.markdown-view__content :deep(h2),
.markdown-view__content :deep(h3),
.markdown-view__content :deep(h4) {
  margin-top: 1rem;
  margin-bottom: 0.5rem;
  font-weight: 600;
  line-height: 1.3;
}

.markdown-view__content :deep(h1) {
  font-size: 1.35rem;
}

.markdown-view__content :deep(h2) {
  font-size: 1.2rem;
}

.markdown-view__content :deep(h3) {
  font-size: 1.05rem;
}

.markdown-view__content :deep(ul),
.markdown-view__content :deep(ol) {
  margin-bottom: 0.75rem;
  padding-left: 1.5rem;
}

.markdown-view__content :deep(li) {
  margin-bottom: 0.25rem;
}

.markdown-view__content :deep(code) {
  padding: 0.15rem 0.35rem;
  border-radius: 4px;
  background-color: rgba(var(--v-theme-surface-variant), 0.7);
  color: rgb(var(--v-theme-primary));
  font-family: monospace;
  font-size: 0.9em;
}

.markdown-view__content :deep(pre) {
  margin-bottom: 0.75rem;
  padding: 0.75rem 1rem;
  overflow-x: auto;
  border-radius: 6px;
  background-color: rgba(var(--v-theme-surface-variant), 0.5);
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.markdown-view__content :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
}

.markdown-view__content :deep(blockquote) {
  margin: 0.75rem 0;
  padding: 0.5rem 1rem;
  border-left: 4px solid rgb(var(--v-theme-primary));
  background-color: rgba(var(--v-theme-surface-variant), 0.3);
  border-radius: 0 4px 4px 0;
  font-style: italic;
}

.markdown-view__content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 0.75rem;
}

.markdown-view__content :deep(th),
.markdown-view__content :deep(td) {
  padding: 0.5rem 0.75rem;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}

.markdown-view__content :deep(th) {
  background-color: rgba(var(--v-theme-surface-variant), 0.5);
  font-weight: 600;
}

.markdown-view__content :deep(a) {
  color: rgb(var(--v-theme-primary));
  text-decoration: underline;
}

.markdown-view__content :deep(hr) {
  margin: 1rem 0;
  border: 0;
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
</style>
