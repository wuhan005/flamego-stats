<script setup lang="ts">
defineProps<{ label: string }>()

const model = defineModel<boolean>({ required: true })
</script>

<template>
  <div class="inline-flex cursor-pointer items-center gap-2 text-sm text-gray-600 select-none dark:text-neutral-400">
    <button
      type="button"
      role="switch"
      :aria-checked="model"
      class="focus-ring relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors"
      :class="model ? 'bg-primary-500' : 'bg-gray-200 dark:bg-neutral-700'"
      @click="model = !model"
    >
      <span
        class="inline-block size-4 rounded-full bg-white shadow transition-transform"
        :class="model ? 'translate-x-4.5' : 'translate-x-0.5'"
      />
    </button>
    <span @click="model = !model">{{ label }}</span>
  </div>
</template>
