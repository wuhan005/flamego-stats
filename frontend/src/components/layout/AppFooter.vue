<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import logo from '@/assets/logo.png'

const { t } = useI18n()
</script>

<template>
  <footer class="mt-16 border-t border-gray-200 dark:border-neutral-800">
    <div
      class="mx-auto flex max-w-7xl flex-col gap-3 px-6 py-8 text-sm text-gray-500 sm:flex-row sm:items-center sm:justify-between dark:text-neutral-500"
    >
      <div class="flex items-center gap-2">
        <img :src="logo" alt="" class="size-5 rounded" />
        <span>{{ t('footer.dataSource') }}</span>
      </div>
      <div class="flex items-center gap-4">
        <a href="https://flamego.dev/" target="_blank" rel="noopener" class="hover:text-gray-900 dark:hover:text-gray-100">
          {{ t('footer.poweredBy') }}
        </a>
        <a
          href="https://github.com/wuhan005/flamego-stats"
          target="_blank"
          rel="noopener"
          class="hover:text-gray-900 dark:hover:text-gray-100"
        >
          {{ t('footer.source') }}
        </a>
      </div>
    </div>
  </footer>
</template>
