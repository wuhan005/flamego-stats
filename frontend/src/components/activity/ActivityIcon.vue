<script setup lang="ts">
import { CircleCheck, CircleDot, GitCommitHorizontal, GitMerge, GitPullRequestArrow, GitPullRequestClosed, Tag } from '@lucide/vue'
import { computed, type Component } from 'vue'

import type { Activity } from '@/api/activity'

const props = defineProps<{ type: Activity['type']; action: Activity['action'] }>()

/** The icon and the status color of each kind of activity, following the conventions of GitHub. */
const STYLES: Record<string, { icon: Component; class: string }> = {
  'commit:committed': {
    icon: GitCommitHorizontal,
    class: 'bg-gray-100 text-gray-500 dark:bg-neutral-800 dark:text-neutral-400',
  },
  'pull_request:opened': {
    icon: GitPullRequestArrow,
    class: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400',
  },
  'pull_request:merged': {
    icon: GitMerge,
    class: 'bg-flame-purple/10 text-flame-purple dark:bg-flame-purple/20 dark:text-violet-300',
  },
  'pull_request:closed': {
    icon: GitPullRequestClosed,
    class: 'bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400',
  },
  'issue:opened': {
    icon: CircleDot,
    class: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400',
  },
  'issue:closed': {
    icon: CircleCheck,
    class: 'bg-flame-purple/10 text-flame-purple dark:bg-flame-purple/20 dark:text-violet-300',
  },
  'release:published': {
    icon: Tag,
    class: 'bg-flame-orange/10 text-orange-600 dark:bg-flame-orange/15 dark:text-flame-orange',
  },
}

const style = computed(() => STYLES[`${props.type}:${props.action}`] ?? STYLES['commit:committed'])
</script>

<template>
  <span class="inline-flex size-8 shrink-0 items-center justify-center rounded-full" :class="style.class">
    <component :is="style.icon" class="size-4" />
  </span>
</template>
