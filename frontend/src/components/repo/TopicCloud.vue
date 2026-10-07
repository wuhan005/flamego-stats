<script setup lang="ts">
import { computed } from 'vue'

import type { CountItem } from '@/api/stats'

const props = defineProps<{ topics: CountItem[] }>()

const max = computed(() => Math.max(1, ...props.topics.map((t) => t.count)))

/** Scales the font size of the topics by their counts, between 12px and 18px. */
function size(count: number) {
  return `${12 + Math.round((Math.sqrt(count) / Math.sqrt(max.value)) * 6)}px`
}
</script>

<template>
  <div class="flex flex-wrap gap-2">
    <RouterLink
      v-for="topic in topics"
      :key="topic.name"
      :to="{ name: 'repositories', query: { topic: topic.name } }"
      class="chip-interactive focus-ring px-3! py-1!"
      :style="{ fontSize: size(topic.count) }"
    >
      <span>{{ topic.name }}</span>
      <span class="text-[11px] text-gray-400 tabular-nums dark:text-neutral-500">{{ topic.count }}</span>
    </RouterLink>
  </div>
</template>
