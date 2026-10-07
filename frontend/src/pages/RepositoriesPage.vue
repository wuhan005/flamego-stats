<script setup lang="ts">
import { ChevronLeft, ChevronRight, Search, X } from '@lucide/vue'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'

import { statsApi, type RepositorySort } from '@/api/stats'
import PanelCard from '@/components/common/PanelCard.vue'
import ToggleSwitch from '@/components/common/ToggleSwitch.vue'
import RepoRow from '@/components/repo/RepoRow.vue'
import { useAsync } from '@/composables/useAsync'
import { usePreferences } from '@/composables/usePreferences'
import { formatNumber } from '@/utils/format'

const PAGE_SIZE = 20
const SORTS: RepositorySort[] = ['stars', 'pushed', 'forks', 'created', 'name']
const FILTERS = ['topic', 'language', 'owner', 'version'] as const
type Filter = (typeof FILTERS)[number]

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const { includeForks } = usePreferences()

function queryString(query: LocationQuery, key: string): string {
  const value = query[key]
  return typeof value === 'string' ? value : ''
}

const sort = computed<RepositorySort>(() => {
  const value = queryString(route.query, 'sort') as RepositorySort
  return SORTS.includes(value) ? value : 'stars'
})
const keyword = computed(() => queryString(route.query, 'q'))
const page = computed(() => Math.max(1, Number(queryString(route.query, 'page')) || 1))
const filters = computed(() => FILTERS.map((key) => ({ key, value: queryString(route.query, key) })).filter((f) => f.value))

/** Updates the query of the page, the page number resets unless it is changed. */
function updateQuery(patch: Record<string, string | undefined>) {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries({ ...route.query, page: undefined, ...patch })) {
    if (typeof value === 'string' && value !== '') query[key] = value
  }
  if (query.sort === 'stars') delete query.sort
  if (query.page === '1') delete query.page
  void router.push({ query })
}

const search = ref(keyword.value)
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, (value) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => updateQuery({ q: value.trim() || undefined }), 300)
})
watch(keyword, (value) => {
  if (value !== search.value.trim()) search.value = value
})

const result = useAsync(
  () =>
    statsApi.repositories({
      page: page.value,
      pageSize: PAGE_SIZE,
      sort: sort.value,
      keyword: keyword.value || undefined,
      topic: queryString(route.query, 'topic') || undefined,
      language: queryString(route.query, 'language') || undefined,
      owner: queryString(route.query, 'owner') || undefined,
      version: queryString(route.query, 'version') || undefined,
      includeForks: includeForks.value,
    }),
  [() => route.query, includeForks],
)

const total = computed(() => result.data.value?.total ?? 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

function filterLabel(key: Filter) {
  return t(`repos.filters.${key}`)
}
</script>

<template>
  <div class="mx-auto max-w-5xl px-6 pt-12">
    <header>
      <h1 class="text-3xl font-bold tracking-tight">{{ t('repos.title') }}</h1>
      <p class="mt-2 text-gray-600 dark:text-neutral-400">
        <span v-if="result.data.value">{{ t('repos.subtitle', { count: formatNumber(total, locale) }) }}</span>
        <span v-else class="skeleton inline-block h-5 w-56 align-middle" />
      </p>
    </header>

    <div class="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
      <label class="relative flex-1">
        <Search class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-gray-400" />
        <input
          v-model="search"
          type="search"
          :placeholder="t('repos.search')"
          class="w-full rounded-lg border-0 bg-black/5 py-2 pr-3 pl-9 text-sm transition-colors outline-none placeholder:text-gray-500 focus:bg-white focus:ring-2 focus:ring-primary-500/50 dark:bg-gray-50/10 dark:placeholder:text-neutral-500 dark:focus:bg-dark"
        />
      </label>
      <div class="flex items-center gap-4">
        <select
          :value="sort"
          class="focus-ring rounded-lg border border-gray-200 bg-white py-1.5 pr-8 pl-3 text-sm dark:border-neutral-700 dark:bg-neutral-900"
          @change="updateQuery({ sort: ($event.target as HTMLSelectElement).value })"
        >
          <option v-for="s in SORTS" :key="s" :value="s">{{ t(`repos.sort.${s}`) }}</option>
        </select>
        <ToggleSwitch v-model="includeForks" :label="t('common.includeForks')" />
      </div>
    </div>

    <div v-if="filters.length" class="mt-4 flex flex-wrap items-center gap-2">
      <button
        v-for="filter in filters"
        :key="filter.key"
        type="button"
        class="chip-interactive focus-ring py-1!"
        @click="updateQuery({ [filter.key]: undefined })"
      >
        <span class="text-gray-500 dark:text-neutral-400">{{ filterLabel(filter.key) }}:</span>
        <span class="font-medium" :class="filter.key === 'version' ? 'font-mono' : ''">{{ filter.value }}</span>
        <X class="size-3" />
      </button>
      <button
        v-if="filters.length > 1"
        type="button"
        class="link focus-ring rounded text-xs"
        @click="updateQuery(Object.fromEntries(FILTERS.map((key) => [key, undefined])))"
      >
        {{ t('repos.filters.clear') }}
      </button>
    </div>

    <PanelCard
      class="mt-6"
      :loading="result.loading.value && !result.data.value"
      :error="result.error.value"
      :skeleton="8"
      @retry="result.reload"
    >
      <p v-if="!result.data.value?.repositories.length" class="py-12 text-center text-sm text-gray-500 dark:text-neutral-400">
        {{ t('repos.noResults') }}
      </p>
      <div
        v-else
        class="-my-3.5 divide-y divide-gray-100 transition-opacity dark:divide-neutral-800"
        :class="result.loading.value ? 'opacity-60' : ''"
      >
        <RepoRow v-for="repo in result.data.value.repositories" :key="repo.fullName" :repo="repo" />
      </div>
      <template v-if="totalPages > 1" #footer>
        <nav class="flex items-center justify-between text-sm">
          <button
            type="button"
            class="focus-ring inline-flex items-center gap-1 rounded-md px-2 py-1 text-gray-600 hover:bg-gray-100 disabled:pointer-events-none disabled:opacity-40 dark:text-neutral-400 dark:hover:bg-neutral-800"
            :disabled="page <= 1"
            @click="updateQuery({ page: String(page - 1) })"
          >
            <ChevronLeft class="size-4" />{{ t('repos.previous') }}
          </button>
          <span class="text-gray-500 tabular-nums dark:text-neutral-400">{{ t('repos.page', { page, total: totalPages }) }}</span>
          <button
            type="button"
            class="focus-ring inline-flex items-center gap-1 rounded-md px-2 py-1 text-gray-600 hover:bg-gray-100 disabled:pointer-events-none disabled:opacity-40 dark:text-neutral-400 dark:hover:bg-neutral-800"
            :disabled="page >= totalPages"
            @click="updateQuery({ page: String(page + 1) })"
          >
            {{ t('repos.next') }}<ChevronRight class="size-4" />
          </button>
        </nav>
      </template>
    </PanelCard>
  </div>
</template>
