<script setup lang="ts">
import { computed, ref } from 'vue'

export interface TrendPoint {
  time: string
  value: number
}

const props = defineProps<{ points: TrendPoint[]; locale: string }>()

const WIDTH = 600
const HEIGHT = 160
const PAD_X = 8
const PAD_Y = 16

const hovered = ref<number>()

const range = computed(() => {
  const values = props.points.map((p) => p.value)
  const min = Math.min(...values)
  const max = Math.max(...values)
  // Keep a flat line in the middle rather than at the edge.
  const span = max - min || Math.max(1, max * 0.2)
  return { min: min - span * 0.15, max: max + span * 0.15 }
})

const coords = computed(() =>
  props.points.map((p, i) => ({
    x: PAD_X + (props.points.length > 1 ? (i / (props.points.length - 1)) * (WIDTH - PAD_X * 2) : (WIDTH - PAD_X * 2) / 2),
    y: PAD_Y + (1 - (p.value - range.value.min) / (range.value.max - range.value.min)) * (HEIGHT - PAD_Y * 2),
    ...p,
  })),
)

const line = computed(() => coords.value.map((c, i) => `${i ? 'L' : 'M'}${c.x.toFixed(1)},${c.y.toFixed(1)}`).join(' '))
const area = computed(() => {
  const first = coords.value[0]
  const last = coords.value[coords.value.length - 1]
  return `${line.value} L${last.x.toFixed(1)},${HEIGHT} L${first.x.toFixed(1)},${HEIGHT} Z`
})

const tooltip = computed(() => {
  if (hovered.value === undefined) return undefined
  const c = coords.value[hovered.value]
  return {
    left: `${(c.x / WIDTH) * 100}%`,
    value: new Intl.NumberFormat(props.locale).format(c.value),
    time: new Intl.DateTimeFormat(props.locale, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(c.time)),
  }
})

const axis = computed(() => {
  const values = props.points.map((p) => p.value)
  const date = (iso: string) => new Intl.DateTimeFormat(props.locale, { month: 'short', day: 'numeric' }).format(new Date(iso))
  return {
    max: new Intl.NumberFormat(props.locale).format(Math.max(...values)),
    min: new Intl.NumberFormat(props.locale).format(Math.min(...values)),
    from: date(props.points[0].time),
    to: date(props.points[props.points.length - 1].time),
  }
})

function onMove(e: MouseEvent) {
  const rect = (e.currentTarget as SVGElement).getBoundingClientRect()
  const x = ((e.clientX - rect.left) / rect.width) * WIDTH
  let nearest = 0
  coords.value.forEach((c, i) => {
    if (Math.abs(c.x - x) < Math.abs(coords.value[nearest].x - x)) nearest = i
  })
  hovered.value = nearest
}
</script>

<template>
  <div class="relative">
    <svg
      :viewBox="`0 0 ${WIDTH} ${HEIGHT}`"
      preserveAspectRatio="none"
      class="h-40 w-full overflow-visible"
      @mousemove="onMove"
      @mouseleave="hovered = undefined"
    >
      <defs>
        <linearGradient id="trend-stroke" x1="0" x2="1" y1="0" y2="0">
          <stop offset="0%" stop-color="var(--color-flame-orange)" />
          <stop offset="55%" stop-color="var(--color-flame-pink)" />
          <stop offset="100%" stop-color="var(--color-flame-purple)" />
        </linearGradient>
        <linearGradient id="trend-fill" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stop-color="var(--color-flame-pink)" stop-opacity="0.25" />
          <stop offset="100%" stop-color="var(--color-flame-pink)" stop-opacity="0" />
        </linearGradient>
      </defs>
      <line
        v-for="i in 3"
        :key="i"
        :x1="0"
        :x2="WIDTH"
        :y1="(HEIGHT / 4) * i"
        :y2="(HEIGHT / 4) * i"
        class="stroke-gray-100 dark:stroke-neutral-800"
        vector-effect="non-scaling-stroke"
      />
      <path :d="area" fill="url(#trend-fill)" />
      <path :d="line" fill="none" stroke="url(#trend-stroke)" stroke-width="2.5" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
      <line
        v-if="hovered !== undefined"
        :x1="coords[hovered].x"
        :x2="coords[hovered].x"
        :y1="0"
        :y2="HEIGHT"
        class="stroke-gray-300 dark:stroke-neutral-600"
        stroke-dasharray="3 3"
        vector-effect="non-scaling-stroke"
      />
    </svg>
    <div
      v-if="hovered !== undefined"
      class="pointer-events-none absolute size-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-flame-pink shadow dark:border-dark"
      :style="{ left: `${(coords[hovered].x / WIDTH) * 100}%`, top: `${(coords[hovered].y / HEIGHT) * 100}%` }"
    />
    <div
      v-if="tooltip"
      class="pointer-events-none absolute -top-2 -translate-x-1/2 -translate-y-full rounded-lg border border-gray-200 bg-white px-2.5 py-1.5 text-xs whitespace-nowrap shadow-lg dark:border-neutral-700 dark:bg-neutral-900"
      :style="{ left: tooltip.left }"
    >
      <div class="font-semibold tabular-nums">{{ tooltip.value }}</div>
      <div class="text-gray-500 dark:text-neutral-400">{{ tooltip.time }}</div>
    </div>
    <div class="pointer-events-none absolute top-0 right-0 text-[11px] text-gray-400 tabular-nums dark:text-neutral-500">
      {{ axis.max }}
    </div>
    <div class="pointer-events-none absolute right-0 bottom-0 text-[11px] text-gray-400 tabular-nums dark:text-neutral-500">
      {{ axis.min }}
    </div>
    <div class="mt-2 flex justify-between text-[11px] text-gray-400 dark:text-neutral-500">
      <span>{{ axis.from }}</span>
      <span>{{ axis.to }}</span>
    </div>
  </div>
</template>
