<script setup lang="ts">
import { computed, ref } from 'vue'
import type { RouteLocationRaw } from 'vue-router'

import { percent } from '@/utils/format'

export interface DonutItem {
  key: string
  label: string
  value: number
  color: string
  to?: RouteLocationRaw
}

const props = defineProps<{ items: DonutItem[]; totalLabel: string; locale: string }>()

const RADIUS = 42
const CIRCUMFERENCE = 2 * Math.PI * RADIUS
/** The gap between the segments in the stroke length. */
const GAP = 1.5

const total = computed(() => props.items.reduce((sum, item) => sum + item.value, 0))
const active = ref<string>()

const segments = computed(() => {
  let offset = 0
  return props.items.map((item) => {
    const length = total.value ? (item.value / total.value) * CIRCUMFERENCE : 0
    const segment = {
      ...item,
      dash: `${Math.max(0, length - (props.items.length > 1 ? GAP : 0))} ${CIRCUMFERENCE}`,
      offset: -offset,
      percent: percent(item.value, total.value),
    }
    offset += length
    return segment
  })
})

const center = computed(() => {
  const segment = segments.value.find((s) => s.key === active.value)
  return segment
    ? { value: `${segment.percent}%`, label: segment.label }
    : { value: new Intl.NumberFormat(props.locale).format(total.value), label: props.totalLabel }
})
</script>

<template>
  <div class="flex flex-col items-center gap-6 sm:flex-row sm:items-center">
    <div class="relative size-44 shrink-0">
      <svg viewBox="0 0 100 100" class="size-full -rotate-90">
        <circle cx="50" cy="50" :r="RADIUS" fill="none" class="stroke-gray-100 dark:stroke-neutral-800" stroke-width="10" />
        <circle
          v-for="segment in segments"
          :key="segment.key"
          cx="50"
          cy="50"
          :r="RADIUS"
          fill="none"
          :stroke="segment.color"
          stroke-width="10"
          :stroke-dasharray="segment.dash"
          :stroke-dashoffset="segment.offset"
          class="cursor-pointer transition-opacity duration-200"
          :class="active && active !== segment.key ? 'opacity-30' : ''"
          @mouseenter="active = segment.key"
          @mouseleave="active = undefined"
        />
      </svg>
      <div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
        <span class="text-2xl font-semibold tabular-nums">{{ center.value }}</span>
        <span class="max-w-24 truncate text-xs text-gray-500 dark:text-neutral-400">{{ center.label }}</span>
      </div>
    </div>

    <ul class="grid w-full grid-cols-2 gap-x-4 gap-y-1.5 text-sm sm:grid-cols-1">
      <li v-for="segment in segments" :key="segment.key">
        <component
          :is="segment.to ? 'RouterLink' : 'div'"
          :to="segment.to"
          class="flex items-center gap-2 rounded px-1 py-0.5 transition-colors hover:bg-gray-50 dark:hover:bg-neutral-800/60"
          @mouseenter="active = segment.key"
          @mouseleave="active = undefined"
        >
          <span class="size-2.5 shrink-0 rounded-full" :style="{ backgroundColor: segment.color }" />
          <span class="truncate">{{ segment.label }}</span>
          <span class="ml-auto text-gray-500 tabular-nums dark:text-neutral-400">{{ segment.percent }}%</span>
        </component>
      </li>
    </ul>
  </div>
</template>
