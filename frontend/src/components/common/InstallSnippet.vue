<script setup lang="ts">
import { Check, Copy } from '@lucide/vue'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ module: string }>()

const { t } = useI18n()
const copied = ref(false)
const command = computed(() => `go get ${props.module}`)

async function copy() {
  await navigator.clipboard.writeText(command.value)
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <div
    class="group flex items-center gap-3 rounded-xl bg-gray-50 py-2.5 pr-2.5 pl-4 font-mono text-sm ring-1 ring-gray-200 dark:bg-neutral-900 dark:ring-neutral-800"
  >
    <span class="text-gray-400 select-none dark:text-neutral-600">$</span>
    <code class="truncate">{{ command }}</code>
    <button
      type="button"
      class="focus-ring ml-auto inline-flex size-7 shrink-0 items-center justify-center rounded-md border border-black/5 text-gray-500 transition hover:text-gray-900 dark:border-white/10 dark:text-neutral-400 dark:hover:text-gray-100"
      :title="copied ? t('common.copied') : t('common.copy')"
      @click="copy"
    >
      <Check v-if="copied" class="size-3.5 text-green-500" />
      <Copy v-else class="size-3.5" />
    </button>
  </div>
</template>
