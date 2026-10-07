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
  <ul class="space-y-1.5">
    <li v-for="(item, index) in items" :key="item.key">
      <component
        :is="linkOf(item).is"
        v-bind="linkOf(item).attrs"
        class="group focus-ring relative flex h-8 items-center justify-between gap-3 overflow-hidden rounded-md px-2.5 text-sm"
        :class="item.to || item.href ? 'cursor-pointer' : ''"
      >
        <div
          class="absolute inset-y-0 left-0 rounded-md transition-[width,opacity] duration-500"
          :class="
            index === leader
              ? 'flame-gradient opacity-25 group-hover:opacity-35 dark:opacity-30'
              : 'bg-primary-500/12 group-hover:bg-primary-500/20 dark:bg-primary-400/15'
          "
          :style="{ width: `${Math.max(2, (item.value / max) * 100)}%` }"
        />
        <span class="relative truncate" :class="mono ? 'font-mono text-[13px]' : ''">{{ item.label }}</span>
        <span class="relative shrink-0 text-gray-600 tabular-nums dark:text-neutral-300">
          {{ formatValue(item.value) }}
          <span v-if="item.hint" class="ml-1 text-xs text-gray-400 dark:text-neutral-500">{{ item.hint }}</span>
        </span>
      </component>
    </li>
  </ul>
</template>
