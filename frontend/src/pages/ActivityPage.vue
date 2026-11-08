<script setup lang="ts">
import { CircleDot, GitCommitHorizontal, GitMerge, LoaderCircle, Tag } from '@lucide/vue'
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

import { activityApi, type Activity, type ActivityType, type ListActivitiesQuery } from '@/api/activity'
import ActivityFeed from '@/components/activity/ActivityFeed.vue'
import ActivityHeatmap from '@/components/activity/ActivityHeatmap.vue'
import ContributorList from '@/components/activity/ContributorList.vue'
import ReleaseList from '@/components/activity/ReleaseList.vue'
import BarList, { type BarItem } from '@/components/charts/BarList.vue'
import PanelCard from '@/components/common/PanelCard.vue'
import SegmentedControl from '@/components/common/SegmentedControl.vue'
import SelectMenu from '@/components/common/SelectMenu.vue'
import StatCard from '@/components/common/StatCard.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'
import { useAsync } from '@/composables/useAsync'
import { usePreferences } from '@/composables/usePreferences'
import { groupActivities } from '@/utils/activity'
import { formatDateTime, formatNumber, relativeTime } from '@/utils/format'

const PAGE_SIZE = 30
const TYPES: ActivityType[] = ['commit', 'pull_request', 'issue', 'release']

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const { excludeBots } = usePreferences()

const type = computed<ActivityType | ''>(() => {
  const value = route.query.type as ActivityType
  return TYPES.includes(value) ? value : ''
})
const repository = computed(() => (typeof route.query.repository === 'string' ? route.query.repository : ''))

function updateQuery(patch: Record<string, string>) {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries({ ...route.query, ...patch })) {
    if (typeof value === 'string' && value !== '') query[key] = value
  }
  void router.push({ query })
}

const summary = useAsync(() => activityApi.summary(excludeBots.value), [excludeBots])

function feedQuery(): ListActivitiesQuery {
  return {
    type: type.value || undefined,
    repository: repository.value || undefined,
    excludeBots: excludeBots.value,
    limit: PAGE_SIZE,
  }
}

const feed = useAsync(() => activityApi.list(feedQuery()), [type, repository, excludeBots])

// The pages after the first one, which are reset along with the first page.
const more = shallowRef<Activity[]>([])
const cursor = ref('')
const loadingMore = ref(false)
const moreError = ref('')
watch(feed.data, (page) => {
  more.value = []
  cursor.value = page?.nextCursor ?? ''
  moreError.value = ''
})

async function loadMore() {
  const first = feed.data.value
  loadingMore.value = true
  moreError.value = ''
  try {
    const page = await activityApi.list({ ...feedQuery(), cursor: cursor.value })
    // The filters changed while loading, the page belongs to the previous feed.
    if (feed.data.value !== first) return
    more.value = [...more.value, ...page.activities]
    cursor.value = page.nextCursor
  } catch (e) {
    moreError.value = e instanceof Error ? e.message : String(e)
  } finally {
    loadingMore.value = false
  }
}

const days = computed(() => groupActivities([...(feed.data.value?.activities ?? []), ...more.value]))
const filtered = computed(() => Boolean(type.value || repository.value || excludeBots.value))

const typeOptions = computed(() => [
  { value: '' as const, label: t('activity.types.all') },
  ...TYPES.map((value) => ({ value, label: t(`activity.types.${value}`) })),
])

const repositoryOptions = computed(() => {
  const names = (summary.data.value?.repositories ?? []).map((r) => r.name).sort((a, b) => a.localeCompare(b))
  // Keep the filter selectable before the summary is loaded or if the repository is not tracked.
  if (repository.value && !names.includes(repository.value)) names.unshift(repository.value)
  return [{ value: '', label: t('activity.allRepositories') }, ...names.map((name) => ({ value: name, label: name }))]
})

const stats = computed(() => {
  const s = summary.data.value
  const days = s?.days ?? 30
  const card = (icon: typeof GitMerge, key: 'commits' | 'mergedPullRequests' | 'openedIssues' | 'releases') => ({
    icon,
    label: t(`activity.stats.${key}`),
    value: s ? formatNumber(s.current[key], locale.value) : '',
    hint: s ? t('activity.stats.hint', { days, count: formatNumber(s.previous[key], locale.value) }) : '',
  })
  return [
    card(GitCommitHorizontal, 'commits'),
    card(GitMerge, 'mergedPullRequests'),
    card(CircleDot, 'openedIssues'),
    card(Tag, 'releases'),
  ]
})

const activeRepositories = computed<BarItem[]>(() =>
  (summary.data.value?.repositories ?? [])
    .filter((r) => r.count > 0)
    .slice(0, 8)
    .map((r) => ({
      key: r.name,
      label: r.name,
      value: r.count,
      to: { name: 'activity', query: { ...route.query, repository: r.name } },
    })),
)

const lastSynced = computed(() => {
  const at = summary.data.value?.lastSyncedAt
  return at ? { relative: relativeTime(at, locale.value), absolute: formatDateTime(at, locale.value) } : undefined
})
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 pt-12">
    <header>
      <h1 class="text-3xl font-bold tracking-tight">{{ t('activity.title') }}</h1>
      <p class="mt-2 text-pretty text-gray-600 dark:text-neutral-400">
        <i18n-t v-if="summary.data.value" keypath="activity.subtitle" scope="global">
          <template #org>
            <a :href="`https://github.com/${summary.data.value.organization}`" target="_blank" rel="noopener" class="link font-mono text-[0.95em]">
              {{ summary.data.value.organization }}
            </a>
          </template>
          <template #count>{{ summary.data.value.repositories.length }}</template>
        </i18n-t>
        <span v-else class="skeleton inline-block h-5 w-80 max-w-full align-middle" />
      </p>
    </header>

    <div class="mt-8 flex flex-wrap items-center justify-between gap-3">
      <p class="flex items-center gap-2 text-sm text-gray-500 dark:text-neutral-400">
        <span class="relative flex size-2">
          <span v-if="lastSynced" class="absolute inline-flex size-full animate-ping rounded-full bg-green-400 opacity-60" />
          <span class="relative inline-flex size-2 rounded-full" :class="lastSynced ? 'bg-green-500' : 'bg-gray-400'" />
        </span>
        <span v-if="lastSynced" :title="lastSynced.absolute">{{ t('home.lastSynced', { time: lastSynced.relative }) }}</span>
        <span v-else-if="!summary.loading.value">{{ t('home.neverSynced') }}</span>
        <span v-if="summary.data.value" class="text-gray-400 dark:text-neutral-500">
          · {{ t('activity.stats.period', { days: summary.data.value.days }) }}
        </span>
      </p>
      <ToggleSwitch v-model="excludeBots" :label="t('activity.excludeBots')" />
    </div>

    <div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
      <StatCard v-for="stat in stats" :key="stat.label" v-bind="stat" :loading="summary.loading.value && !summary.data.value" />
    </div>

    <div class="mt-6 grid grid-cols-1 items-start gap-6 lg:grid-cols-3">
      <PanelCard
        class="lg:col-span-2"
        :loading="feed.loading.value && !feed.data.value"
        :error="feed.error.value"
        :empty="!filtered && !days.length"
        :skeleton="8"
        @retry="feed.reload"
      >
        <template #header>
          <div class="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-3">
            <SegmentedControl :model-value="type" :options="typeOptions" @update:model-value="updateQuery({ type: $event })" />
            <SelectMenu
              class="w-44"
              :label="t('activity.allRepositories')"
              :model-value="repository"
              :options="repositoryOptions"
              @update:model-value="updateQuery({ repository: $event })"
            />
          </div>
        </template>

        <p v-if="!days.length" class="py-12 text-center text-sm text-gray-500 dark:text-neutral-400">
          {{ t('activity.noResults') }}
        </p>
        <ActivityFeed v-else :days="days" class="transition-opacity" :class="feed.loading.value ? 'opacity-60' : ''" />

        <template v-if="cursor || moreError" #footer>
          <button
            type="button"
            class="focus-ring inline-flex w-full items-center justify-center gap-1.5 rounded-md py-1 text-sm text-gray-600 transition-colors hover:bg-gray-100 disabled:pointer-events-none disabled:opacity-60 dark:text-neutral-400 dark:hover:bg-neutral-800"
            :disabled="loadingMore"
            @click="loadMore"
          >
            <LoaderCircle v-if="loadingMore" class="size-4 animate-spin" />
            {{ moreError ? t('common.loadFailed', { message: moreError }) : t('activity.loadMore') }}
          </button>
        </template>
      </PanelCard>

      <aside class="grid grid-cols-1 gap-6">
        <PanelCard
          :title="t('activity.heatmap')"
          :description="t('activity.heatmapDesc')"
          :loading="summary.loading.value && !summary.data.value"
          :error="summary.error.value"
          :skeleton="4"
          @retry="summary.reload"
        >
          <ActivityHeatmap :daily="summary.data.value?.daily ?? []" />
        </PanelCard>

        <PanelCard
          :title="t('activity.activeRepos')"
          :description="t('activity.activeReposDesc', { days: summary.data.value?.days ?? 30 })"
          :loading="summary.loading.value && !summary.data.value"
          :error="summary.error.value"
          :empty="!activeRepositories.length"
          @retry="summary.reload"
        >
          <BarList :items="activeRepositories" mono />
        </PanelCard>

        <PanelCard
          :title="t('activity.contributors')"
          :description="t('activity.contributorsDesc', { days: summary.data.value?.days ?? 30 })"
          :loading="summary.loading.value && !summary.data.value"
          :error="summary.error.value"
          :empty="!summary.data.value?.contributors.length"
          @retry="summary.reload"
        >
          <ContributorList :contributors="summary.data.value?.contributors ?? []" />
        </PanelCard>

        <PanelCard
          :title="t('activity.releases')"
          :description="t('activity.releasesDesc')"
          :loading="summary.loading.value && !summary.data.value"
          :error="summary.error.value"
          :empty="!summary.data.value?.releases.length"
          :skeleton="3"
          @retry="summary.reload"
        >
          <ReleaseList :releases="summary.data.value?.releases ?? []" />
        </PanelCard>
      </aside>
    </div>
  </div>
</template>
