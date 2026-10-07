<script setup lang="ts">
import { Archive, GitFork, Star } from '@lucide/vue'
import { useI18n } from 'vue-i18n'

import type { Repository } from '@/api/stats'
import { compactNumber, relativeTime } from '@/utils/format'
import { languageColor } from '@/utils/languages'

withDefaults(defineProps<{ repo: Repository; compact?: boolean }>(), { compact: false })

const { t, locale } = useI18n()
</script>

<template>
  <article class="group flex gap-3 py-3.5">
    <RouterLink :to="{ name: 'repositories', query: { owner: repo.owner.login } }" class="shrink-0" :title="repo.owner.login">
      <img
        :src="`${repo.owner.avatarURL}${repo.owner.avatarURL.includes('?') ? '&' : '?'}s=80`"
        :alt="repo.owner.login"
        loading="lazy"
        class="size-9 bg-gray-100 ring-1 ring-black/5 dark:bg-neutral-800 dark:ring-white/10"
        :class="repo.owner.type === 'Organization' ? 'rounded-lg' : 'rounded-full'"
      />
    </RouterLink>

    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
        <a :href="repo.htmlURL" target="_blank" rel="noopener" class="min-w-0 truncate text-[15px] hover:underline">
          <span class="text-gray-500 dark:text-neutral-400">{{ repo.owner.login }} /</span>
          <span class="font-semibold"> {{ repo.name }}</span>
        </a>
        <span v-if="repo.fork" class="chip py-0! text-[11px]"><GitFork class="size-3" />{{ t('common.fork') }}</span>
        <span
          v-if="repo.archived"
          class="chip border-amber-300/60! bg-amber-50! py-0! text-[11px] text-amber-700! dark:border-amber-500/30! dark:bg-amber-500/10! dark:text-amber-400!"
        >
          <Archive class="size-3" />{{ t('common.archived') }}
        </span>
        <RouterLink
          v-if="repo.moduleVersion"
          :to="{ name: 'repositories', query: { version: repo.moduleVersion } }"
          class="badge-mono hover:text-primary-600 dark:hover:text-primary-400"
          :title="t('repos.goMod', { path: repo.goModPath })"
        >
          {{ repo.moduleVersion }}
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
