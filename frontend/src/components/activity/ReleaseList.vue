<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import type { Activity } from '@/api/activity'
import { formatDateTime, relativeTime } from '@/utils/format'

defineProps<{ releases: Activity[] }>()

const { t, locale } = useI18n()
</script>

<template>
  <ul class="-my-1 divide-y divide-gray-100 dark:divide-neutral-800">
    <li v-for="release in releases" :key="release.id">
      <a :href="release.htmlURL" target="_blank" rel="noopener" class="group flex items-center gap-3 py-2.5 text-sm">
        <span class="flex min-w-0 flex-1 items-center gap-2">
          <span class="truncate font-medium group-hover:underline">{{ release.repository.name }}</span>
          <span class="badge-mono shrink-0">{{ release.tag }}</span>
          <span v-if="release.prerelease" class="chip-warning shrink-0 py-0! text-[11px]">{{ t('activity.prerelease') }}</span>
        </span>
        <time
          class="shrink-0 text-xs text-gray-500 dark:text-neutral-400"
          :datetime="release.occurredAt"
          :title="formatDateTime(release.occurredAt, locale)"
        >
          {{ relativeTime(release.occurredAt, locale) }}
        </time>
      </a>
    </li>
  </ul>
</template>
