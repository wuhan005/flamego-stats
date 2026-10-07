<script setup lang="ts">
import { ArrowRight, BadgeCheck, FolderGit2, Star, Users } from '@lucide/vue'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { statsApi, type VersionGranularity } from '@/api/stats'
import BarList, { type BarItem } from '@/components/charts/BarList.vue'
import DonutChart, { type DonutItem } from '@/components/charts/DonutChart.vue'
import TrendChart from '@/components/charts/TrendChart.vue'
import InstallSnippet from '@/components/common/InstallSnippet.vue'
import PanelCard from '@/components/common/PanelCard.vue'
import SegmentedControl from '@/components/common/SegmentedControl.vue'
import StatCard from '@/components/common/StatCard.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'
import OwnerList from '@/components/repo/OwnerList.vue'
import RepoRow from '@/components/repo/RepoRow.vue'
import TopicCloud from '@/components/repo/TopicCloud.vue'
import { useAsync } from '@/composables/useAsync'
import { usePreferences } from '@/composables/usePreferences'
import { compactNumber, formatDateTime, formatNumber, percent, relativeTime } from '@/utils/format'
import { languageColor } from '@/utils/languages'
import { moduleShortName, moduleURL } from '@/utils/modules'

const { t, locale } = useI18n()
const { includeForks } = usePreferences()
const granularity = ref<VersionGranularity>('minor')

const overview = useAsync(() => statsApi.overview(includeForks.value), [includeForks])
const starred = useAsync(() => statsApi.repositories({ sort: 'stars', pageSize: 6, includeForks: includeForks.value }), [includeForks])
const recent = useAsync(() => statsApi.repositories({ sort: 'pushed', pageSize: 6, includeForks: includeForks.value }), [includeForks])
const owners = useAsync(() => statsApi.owners(includeForks.value, 8), [includeForks])
const versions = useAsync(() => statsApi.versions(includeForks.value, granularity.value), [includeForks, granularity])
const modules = useAsync(() => statsApi.modules(includeForks.value, 10), [includeForks])
const goVersions = useAsync(() => statsApi.goVersions(includeForks.value), [includeForks])
const languages = useAsync(() => statsApi.languages(includeForks.value, 100), [includeForks])
const topics = useAsync(() => statsApi.topics(includeForks.value, 40), [includeForks])
const syncRuns = useAsync(() => statsApi.syncRuns(100))

const stats = computed(() => {
  const o = overview.data.value
  return [
    {
      icon: FolderGit2,
      label: t('home.stats.dependents'),
      value: o ? formatNumber(o.repositories, locale.value) : '',
      hint: o?.reportedTotal ? t('home.stats.dependentsHint', { count: formatNumber(o.reportedTotal, locale.value) }) : '',
    },
    {
      icon: Users,
      label: t('home.stats.owners'),
      value: o ? formatNumber(o.owners, locale.value) : '',
      hint: t('home.stats.ownersHint'),
    },
    {
      icon: Star,
      label: t('home.stats.stars'),
      value: o ? compactNumber(o.stars, locale.value) : '',
      hint: t('home.stats.starsHint'),
    },
    {
      icon: BadgeCheck,
      label: t('home.stats.resolved'),
      value: o ? formatNumber(o.resolved, locale.value) : '',
      hint: o ? t('home.stats.resolvedHint', { percent: percent(o.resolved, o.repositories) }) : '',
    },
  ]
})

/** The most used versions, so the long tail of the pseudo-versions does not flood the list. */
const versionItems = computed<BarItem[]>(() => {
  const items = versions.data.value ?? []
  const total = items.reduce((sum, item) => sum + item.count, 0)
  return [...items]
    .sort((a, b) => b.count - a.count)
    .slice(0, 8)
    .map((item) => ({
      key: item.name,
      label: item.name,
      value: item.count,
      hint: `${percent(item.count, total)}%`,
      to: granularity.value === 'exact' ? { name: 'repositories', query: { version: item.name } } : undefined,
    }))
})

const moduleItems = computed<BarItem[]>(() =>
  (modules.data.value ?? []).map((item) => ({
    key: item.name,
    label: moduleShortName(item.name),
    value: item.count,
    href: moduleURL(item.name),
  })),
)

const goVersionItems = computed<BarItem[]>(() =>
  (goVersions.data.value ?? []).slice(0, 8).map((item) => ({ key: item.name, label: `go ${item.name}`, value: item.count })),
)

/** The top languages with the rest grouped as "Other". */
const languageItems = computed<DonutItem[]>(() => {
  const items = languages.data.value ?? []
  const top: DonutItem[] = items.slice(0, 6).map((item) => ({
    key: item.name,
    label: item.name,
    value: item.count,
    color: languageColor(item.name),
    to: { name: 'repositories', query: { language: item.name } },
  }))
  const rest = items.slice(6).reduce((sum, item) => sum + item.count, 0)
  if (rest > 0) top.push({ key: '__other', label: t('common.other'), value: rest, color: '#9ca3af' })
  return top
})

const trendPoints = computed(() =>
  (syncRuns.data.value ?? [])
    // The partial runs miss the repositories of the failed sources.
    .filter((run) => run.status === 'succeeded' && !run.partial && run.finishedAt)
    .reverse()
    .map((run) => ({ time: run.finishedAt as string, value: run.discovered })),
)

const lastSynced = computed(() => {
  const at = overview.data.value?.lastSyncedAt
  return at ? { relative: relativeTime(at, locale.value), absolute: formatDateTime(at, locale.value) } : undefined
})

const versionOptions = computed(() => [
  { value: 'minor' as const, label: t('home.minor') },
  { value: 'exact' as const, label: t('home.exact') },
])
</script>

<template>
  <div>
    <section class="relative overflow-hidden border-b border-gray-200 dark:border-neutral-800">
      <div
        class="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_60%_80%_at_50%_-20%,rgba(242,97,111,0.16),transparent)] dark:bg-[radial-gradient(ellipse_60%_80%_at_50%_-20%,rgba(242,97,111,0.22),transparent)]"
      />
      <div
        class="pointer-events-none absolute inset-0 [background-image:linear-gradient(to_right,rgb(0_0_0/0.04)_1px,transparent_1px),linear-gradient(to_bottom,rgb(0_0_0/0.04)_1px,transparent_1px)] [background-size:32px_32px] [mask-image:radial-gradient(ellipse_70%_60%_at_50%_0%,black,transparent)] dark:[background-image:linear-gradient(to_right,rgb(255_255_255/0.05)_1px,transparent_1px),linear-gradient(to_bottom,rgb(255_255_255/0.05)_1px,transparent_1px)]"
      />
      <div class="relative mx-auto max-w-7xl px-6 pt-16 pb-14 sm:pt-20">
        <div class="max-w-3xl">
          <h1 class="text-4xl font-bold tracking-tight sm:text-5xl">
            {{ t('home.titlePrefix') }} <span class="flame-text">Flamego</span>?
          </h1>
          <p class="mt-4 text-lg leading-relaxed text-gray-600 dark:text-neutral-400">
            <i18n-t keypath="home.subtitle" scope="global">
              <template #module>
                <code class="font-mono text-[0.9em] text-gray-900 dark:text-gray-100">{{ overview.data.value?.module ?? 'github.com/flamego/flamego' }}</code>
              </template>
            </i18n-t>
          </p>
          <div class="mt-8 flex flex-col gap-4 sm:flex-row sm:items-center">
            <InstallSnippet :module="overview.data.value?.module ?? 'github.com/flamego/flamego'" class="sm:w-96" />
            <a
              href="https://flamego.dev/starter-guide/"
              target="_blank"
              rel="noopener"
              class="focus-ring group inline-flex items-center gap-1.5 rounded-lg text-sm font-medium text-gray-700 hover:text-gray-900 dark:text-neutral-300 dark:hover:text-gray-100"
            >
              {{ t('home.getStarted') }}
              <ArrowRight class="size-4 transition-transform group-hover:translate-x-0.5" />
            </a>
          </div>
        </div>

        <div class="mt-10 flex flex-wrap items-center justify-between gap-3">
          <p class="flex items-center gap-2 text-sm text-gray-500 dark:text-neutral-400">
            <span class="relative flex size-2">
              <span v-if="lastSynced" class="absolute inline-flex size-full animate-ping rounded-full bg-green-400 opacity-60" />
              <span class="relative inline-flex size-2 rounded-full" :class="lastSynced ? 'bg-green-500' : 'bg-gray-400'" />
            </span>
            <span v-if="lastSynced" :title="lastSynced.absolute">{{ t('home.lastSynced', { time: lastSynced.relative }) }}</span>
            <span v-else-if="!overview.loading.value">{{ t('home.neverSynced') }}</span>
          </p>
          <ToggleSwitch v-model="includeForks" :label="t('common.includeForks')" />
        </div>

        <div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard v-for="stat in stats" :key="stat.label" v-bind="stat" :loading="overview.loading.value" />
        </div>
      </div>
    </section>

    <div class="mx-auto grid max-w-7xl gap-6 px-6 pt-10 lg:grid-cols-2">
      <PanelCard
        :title="t('home.mostStarred')"
        :description="t('home.mostStarredDesc')"
        :loading="starred.loading.value"
        :error="starred.error.value"
        :empty="!starred.data.value?.repositories.length"
        @retry="starred.reload"
      >
        <div class="-my-3.5 divide-y divide-gray-100 dark:divide-neutral-800">
          <RepoRow v-for="repo in starred.data.value?.repositories" :key="repo.fullName" :repo="repo" compact />
        </div>
        <template #footer>
          <RouterLink :to="{ name: 'repositories', query: { sort: 'stars' } }" class="link inline-flex items-center gap-1 text-sm">
            {{ t('common.viewAll') }} <ArrowRight class="size-3.5" />
          </RouterLink>
        </template>
      </PanelCard>

      <PanelCard
        :title="t('home.recentlyUpdated')"
        :description="t('home.recentlyUpdatedDesc')"
        :loading="recent.loading.value"
        :error="recent.error.value"
        :empty="!recent.data.value?.repositories.length"
        @retry="recent.reload"
      >
        <div class="-my-3.5 divide-y divide-gray-100 dark:divide-neutral-800">
          <RepoRow v-for="repo in recent.data.value?.repositories" :key="repo.fullName" :repo="repo" compact />
        </div>
        <template #footer>
          <RouterLink :to="{ name: 'repositories', query: { sort: 'pushed' } }" class="link inline-flex items-center gap-1 text-sm">
            {{ t('common.viewAll') }} <ArrowRight class="size-3.5" />
          </RouterLink>
        </template>
      </PanelCard>

      <PanelCard
        :title="t('home.topOwners')"
        :description="t('home.topOwnersDesc')"
        :loading="owners.loading.value"
        :error="owners.error.value"
        :empty="!owners.data.value?.length"
        @retry="owners.reload"
      >
        <OwnerList :owners="owners.data.value ?? []" />
      </PanelCard>

      <PanelCard
        :title="t('home.versions')"
        :description="t('home.versionsDesc')"
        :loading="versions.loading.value"
        :error="versions.error.value"
        :empty="!versionItems.length"
        @retry="versions.reload"
      >
        <template #actions>
          <SegmentedControl v-model="granularity" :options="versionOptions" />
        </template>
        <BarList :items="versionItems" mono />
      </PanelCard>

      <PanelCard
        :title="t('home.middleware')"
        :description="t('home.middlewareDesc')"
        :loading="modules.loading.value"
        :error="modules.error.value"
        :empty="!moduleItems.length"
        @retry="modules.reload"
      >
        <BarList :items="moduleItems" mono />
      </PanelCard>

      <PanelCard
        :title="t('home.goVersions')"
        :description="t('home.goVersionsDesc')"
        :loading="goVersions.loading.value"
        :error="goVersions.error.value"
        :empty="!goVersionItems.length"
        @retry="goVersions.reload"
      >
        <BarList :items="goVersionItems" mono />
      </PanelCard>

      <PanelCard
        :title="t('home.languages')"
        :description="t('home.languagesDesc')"
        :loading="languages.loading.value"
        :error="languages.error.value"
        :empty="!languageItems.length"
        @retry="languages.reload"
      >
        <DonutChart :items="languageItems" :total-label="t('common.repositories')" :locale="locale" />
      </PanelCard>

      <PanelCard
        :title="t('home.topics')"
        :description="t('home.topicsDesc')"
        :loading="topics.loading.value"
        :error="topics.error.value"
        :empty="!topics.data.value?.length"
        @retry="topics.reload"
      >
        <TopicCloud :topics="topics.data.value ?? []" />
      </PanelCard>

      <PanelCard
        class="lg:col-span-2"
        :title="t('home.trend')"
        :description="t('home.trendDesc')"
        :loading="syncRuns.loading.value"
        :error="syncRuns.error.value"
        :skeleton="3"
        @retry="syncRuns.reload"
      >
        <TrendChart v-if="trendPoints.length >= 2" :points="trendPoints" :locale="locale" />
        <p v-else class="py-10 text-center text-sm text-gray-500 dark:text-neutral-400">{{ t('home.trendEmpty') }}</p>
      </PanelCard>
    </div>
  </div>
</template>
