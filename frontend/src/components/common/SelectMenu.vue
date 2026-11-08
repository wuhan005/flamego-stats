<script setup lang="ts" generic="T extends string">
import { ChevronDown } from '@lucide/vue'

defineProps<{ options: { value: T; label: string }[]; label?: string }>()

const model = defineModel<T>({ required: true })
</script>

<template>
  <label class="relative inline-flex min-w-0">
    <select
      v-model="model"
      :aria-label="label"
      class="w-full min-w-0 appearance-none truncate rounded-lg border-0 bg-black/5 py-2 pr-8 pl-3 text-sm transition-colors outline-none focus:ring-2 focus:ring-primary-500/50 dark:bg-gray-50/10"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
    </select>
    <ChevronDown class="pointer-events-none absolute top-1/2 right-2.5 size-4 -translate-y-1/2 text-gray-400" />
  </label>
</template>
