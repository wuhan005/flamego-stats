<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  label: string
  value: string
  hint?: string
  icon: Component
  loading?: boolean
}>()
</script>

<template>
  <div class="surface group relative overflow-hidden p-5">
    <div
      class="flame-gradient pointer-events-none absolute -top-16 -right-16 size-32 rounded-full opacity-0 blur-3xl transition-opacity duration-500 group-hover:opacity-20"
    />
    <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-neutral-400">
      <component :is="icon" class="size-4" />
      <span>{{ label }}</span>
    </div>
    <div v-if="loading" class="skeleton mt-3 h-9 w-24" />
    <div v-else class="mt-2 text-3xl font-semibold tracking-tight tabular-nums">{{ value }}</div>
    <div v-if="hint" class="mt-1 truncate text-xs text-gray-500 dark:text-neutral-500">{{ hint }}</div>
  </div>
</template>
