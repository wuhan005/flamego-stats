<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import type { DailyCount } from '@/api/activity'
import { formatNumber } from '@/utils/format'
import { heatmapMonths, heatmapWeeks, localDate, parseLocalDate, type HeatmapCell } from '@/utils/activity'

const props = defineProps<{ daily: DailyCount[] }>()

const { t, locale } = useI18n()

const weeks = computed(() => heatmapWeeks(props.daily))
const months = computed(() =>
  heatmapMonths(weeks.value).map((month) => ({
    week: month.week,
    label: new Intl.DateTimeFormat(locale.value, { month: 'short' }).format(parseLocalDate(month.date)),
  })),
)
const total = computed(() => props.daily.reduce((sum, day) => sum + day.count, 0))
const today = localDate(new Date())

const LEVELS: Record<HeatmapCell['level'], string> = {
  0: 'bg-gray-100 dark:bg-neutral-800',
  1: 'bg-primary-500/25 dark:bg-primary-400/25',
  2: 'bg-primary-500/45 dark:bg-primary-400/45',
  3: 'bg-primary-500/70 dark:bg-primary-400/70',
  4: 'bg-primary-500 dark:bg-primary-400',
}

function cellTitle(cell: HeatmapCell): string {
  const date = new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium' }).format(parseLocalDate(cell.date))
  return t('activity.heatmapCell', { date, count: cell.count })
}
</script>

<template>
  <div>
    <div class="grid gap-1 text-[11px] text-gray-400 dark:text-neutral-500" :style="{ gridTemplateColumns: `repeat(${weeks.length}, 1fr)` }">
      <span v-for="month in months" :key="month.week" class="truncate" :style="{ gridColumnStart: month.week + 1 }">
        {{ month.label }}
      </span>
    </div>

    <div class="mt-1.5 grid grid-flow-col grid-rows-7 gap-1" :style="{ gridTemplateColumns: `repeat(${weeks.length}, 1fr)` }">
      <template v-for="(week, index) in weeks" :key="index">
        <span
          v-for="(cell, day) in week"
          :key="day"
          class="aspect-square rounded-[3px]"
          :class="[
            cell ? LEVELS[cell.level] : '',
            cell?.date === today ? 'ring-1 ring-gray-400 ring-offset-1 ring-offset-white dark:ring-neutral-500 dark:ring-offset-neutral-900' : '',
          ]"
          :title="cell ? cellTitle(cell) : undefined"
        />
      </template>
    </div>

    <div class="mt-3 flex items-center justify-between text-xs text-gray-500 dark:text-neutral-400">
      <span class="tabular-nums">{{ t('activity.heatmapTotal', { count: formatNumber(total, locale) }) }}</span>
      <span class="flex items-center gap-1">
        {{ t('activity.less') }}
        <span v-for="level in [0, 1, 2, 3, 4] as const" :key="level" class="size-2.5 rounded-[2px]" :class="LEVELS[level]" />
        {{ t('activity.more') }}
      </span>
    </div>
  </div>
</template>
