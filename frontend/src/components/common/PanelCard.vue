<script setup lang="ts">
import { CircleAlert, Inbox } from '@lucide/vue'
import { useI18n } from 'vue-i18n'

withDefaults(
  defineProps<{
    title?: string
    description?: string
    loading?: boolean
    error?: string
    empty?: boolean
    /** Number of the skeleton lines while loading. */
    skeleton?: number
  }>(),
  { title: '', description: '', loading: false, error: '', empty: false, skeleton: 5 },
)

defineEmits<{ retry: [] }>()

const { t } = useI18n()
</script>

<template>
  <section class="surface flex flex-col">
    <header v-if="title || $slots.actions" class="flex items-start justify-between gap-4 px-5 pt-5 pb-4">
      <div class="min-w-0">
        <h2 class="text-base font-semibold tracking-tight">{{ title }}</h2>
        <p v-if="description" class="mt-0.5 text-sm text-gray-500 dark:text-neutral-400">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="shrink-0">
        <slot name="actions" />
      </div>
    </header>

    <div class="flex-1 px-5 pb-5" :class="title || $slots.actions ? '' : 'pt-5'">
      <div v-if="loading" class="space-y-3 pt-1">
        <div v-for="i in skeleton" :key="i" class="skeleton h-7" :style="{ width: `${100 - i * 9}%` }" />
      </div>
      <div
        v-else-if="error"
        class="flex flex-col items-center justify-center gap-3 py-10 text-center text-sm text-gray-500 dark:text-neutral-400"
      >
        <CircleAlert class="size-6 text-red-500" />
        <p>{{ t('common.loadFailed', { message: error }) }}</p>
        <button type="button" class="link focus-ring rounded text-sm" @click="$emit('retry')">
          {{ t('common.retry') }}
        </button>
      </div>
      <div
        v-else-if="empty"
        class="flex flex-col items-center justify-center gap-2 py-10 text-center text-sm text-gray-500 dark:text-neutral-400"
      >
        <Inbox class="size-6 text-gray-400 dark:text-neutral-600" />
        <p>{{ t('common.empty') }}</p>
      </div>
      <slot v-else />
    </div>

    <footer v-if="$slots.footer && !loading && !error && !empty" class="border-t border-gray-100 px-5 py-3 dark:border-neutral-800">
      <slot name="footer" />
    </footer>
  </section>
</template>
