<script setup lang="ts">
import { computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'

export interface BarItem {
  key: string
  label: string
  value: number
  /** Internal route of the item. */
  to?: RouteLocationRaw
  /** External link of the item, it is ignored if to is set. */
  href?: string
  /** Shown after the value, e.g. the percentage. */
  hint?: string
  /** 悬停提示，默认为 label。 */
  title?: string
}

const props = withDefaults(defineProps<{ items: BarItem[]; mono?: boolean; formatValue?: (value: number) => string }>(), {
  mono: false,
  formatValue: (value: number) => String(value),
})

const max = computed(() => Math.max(1, ...props.items.map((item) => item.value)))
/** The first item with the largest value is highlighted with the gradient of the logo. */
const leader = computed(() => props.items.findIndex((item) => item.value === max.value))

function linkOf(item: BarItem) {
  if (item.to) return { is: 'RouterLink', attrs: { to: item.to } }
  if (item.href) return { is: 'a', attrs: { href: item.href, target: '_blank', rel: 'noopener' } }
  return { is: 'div', attrs: {} }
}
</script>

<template>
  <!-- 各行共用同一套列宽（subgrid），数值列宽度不同也不影响条形长度的比例。 -->
  <ul class="grid grid-cols-[minmax(0,1fr)_auto] gap-y-1.5">
    <li v-for="(item, index) in items" :key="item.key" class="col-span-2 grid grid-cols-subgrid">
      <component
        :is="linkOf(item).is"
        v-bind="linkOf(item).attrs"
        class="group focus-ring col-span-2 grid h-8 grid-cols-subgrid items-center gap-x-3 rounded-md text-sm"
        :class="item.to || item.href ? 'cursor-pointer' : ''"
      >
        <!-- 宽度取比例与标签宽度的较大值，标签过长时在整列宽度内截断。 -->
        <div
          class="relative flex h-full w-max max-w-full items-center rounded-md px-2.5 transition-[min-width] duration-500"
          :style="{ minWidth: `${(item.value / max) * 100}%` }"
        >
          <div
            class="absolute inset-0 rounded-md transition-opacity"
            :class="
              index === leader
                ? 'flame-gradient opacity-25 group-hover:opacity-35 dark:opacity-30'
                : 'bg-primary-500/12 group-hover:bg-primary-500/20 dark:bg-primary-400/15'
            "
          />
          <span class="relative min-w-0 truncate" :class="mono ? 'font-mono text-[13px]' : ''" :title="item.title ?? item.label">{{ item.label }}</span>
        </div>
        <span class="pr-2.5 text-right text-gray-600 tabular-nums dark:text-neutral-300">
          {{ formatValue(item.value) }}
          <span v-if="item.hint" class="ml-1 inline-block min-w-[5ch] text-xs text-gray-400 dark:text-neutral-500">{{ item.hint }}</span>
        </span>
      </component>
    </li>
  </ul>
</template>
