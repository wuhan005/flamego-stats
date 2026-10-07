<script setup lang="ts" generic="T extends string">
defineProps<{ options: { value: T; label: string }[] }>()

const model = defineModel<T>({ required: true })
</script>

<template>
  <div class="inline-flex rounded-lg bg-gray-100 p-0.5 text-xs dark:bg-neutral-800">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="focus-ring rounded-md px-2.5 py-1 font-medium transition-colors"
      :class="
        model === option.value
          ? 'bg-white text-gray-900 shadow-sm dark:bg-neutral-700 dark:text-gray-100'
          : 'text-gray-500 hover:text-gray-900 dark:text-neutral-400 dark:hover:text-gray-100'
      "
      :aria-pressed="model === option.value"
      @click="model = option.value"
    >
      {{ option.label }}
    </button>
  </div>
</template>
