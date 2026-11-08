<script setup lang="ts">
import { Archive, GitFork, Star } from '@lucide/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Repository } from '@/api/stats'
import OwnerAvatar from '@/components/common/OwnerAvatar.vue'
import { compactNumber, relativeTime } from '@/utils/format'
import { languageColor } from '@/utils/languages'
import { shortVersion } from '@/utils/modules'

const props = withDefaults(defineProps<{ repo: Repository; compact?: boolean }>(), { compact: false })

const { t, locale } = useI18n()
const version = computed(() => shortVersion(props.repo.moduleVersion ?? ''))
/** Only an abbreviated pseudo-version may be truncated further, a release version is always shown in full. */
const pseudoVersion = computed(() => version.value !== props.repo.moduleVersion)
</script>

<template>
  <article class="group flex gap-3 py-3.5">
    <RouterLink :to="{ name: 'repositories', query: { owner: repo.owner.login } }" class="shrink-0" :title="repo.owner.login">
      <OwnerAvatar :owner="repo.owner" :size="80" class="size-9" />
    </RouterLink>

    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <!-- Short of room, the owner is truncated first, then the pseudo-version, and the repository name last. -->
        <a :href="repo.htmlURL" target="_blank" rel="noopener" class="flex min-w-0 text-[15px] whitespace-nowrap hover:underline">
          <span class="min-w-[3ch] shrink-[100] truncate text-gray-500 dark:text-neutral-400">{{ repo.owner.login }}</span>
          <span class="mx-1 text-gray-500 dark:text-neutral-400">/</span>
          <span class="min-w-0 truncate font-semibold">{{ repo.name }}</span>
        </a>
        <span v-if="repo.fork" class="chip shrink-0 py-0! text-[11px]"><GitFork class="size-3" /><span class="max-sm:sr-only">{{ t('common.fork') }}</span></span>
        <span v-if="repo.archived" class="chip-warning shrink-0 py-0! text-[11px]">
          <Archive class="size-3" /><span class="max-sm:sr-only">{{ t('common.archived') }}</span>
        </span>
        <RouterLink
          v-if="repo.moduleVersion"
          :to="{ name: 'repositories', query: { version: repo.moduleVersion } }"
          class="badge-mono hover:text-primary-600 dark:hover:text-primary-400"
          :class="pseudoVersion ? 'min-w-[7ch] shrink-[10]' : 'shrink-0'"
          :title="`${repo.moduleVersion} · ${t('repos.goMod', { path: repo.goModPath })}`"
        >
          <span class="truncate">{{ version }}</span>
        </RouterLink>
      </div>

      <p v-if="repo.description" class="mt-1 text-sm text-gray-600 dark:text-neutral-400" :class="compact ? 'line-clamp-1' : 'line-clamp-2'">
        {{ repo.description }}
      </p>

      <div v-if="!compact && repo.topics.length" class="mt-2 flex flex-wrap gap-1.5">
        <RouterLink
          v-for="topic in repo.topics.slice(0, 6)"
          :key="topic"
          :to="{ name: 'repositories', query: { topic } }"
          class="chip-interactive border-transparent! bg-primary-50/60! text-primary-700! dark:bg-primary-500/10! dark:text-primary-300!"
        >
          {{ topic }}
        </RouterLink>
      </div>

      <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-neutral-400">
        <span v-if="repo.language" class="inline-flex items-center gap-1.5">
          <span class="size-2.5 rounded-full" :style="{ backgroundColor: languageColor(repo.language) }" />
          {{ repo.language }}
        </span>
        <span class="inline-flex items-center gap-1" :title="t('common.stars')">
          <Star class="size-3.5" />{{ compactNumber(repo.stars, locale) }}
        </span>
        <span class="inline-flex items-center gap-1" :title="t('common.forks')">
          <GitFork class="size-3.5" />{{ compactNumber(repo.forks, locale) }}
        </span>
        <span v-if="repo.pushedAt" :title="repo.pushedAt">{{ t('repos.pushed', { time: relativeTime(repo.pushedAt, locale) }) }}</span>
      </div>
    </div>
  </article>
</template>
