<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  owner: { login: string; avatarURL: string; type: string }
  /** 请求的头像边长（像素），通常为显示尺寸的两倍。 */
  size: number
}>()

const src = computed(() => `${props.owner.avatarURL}${props.owner.avatarURL.includes('?') ? '&' : '?'}s=${props.size}`)
const loaded = ref(false)
</script>

<template>
  <!-- 透明头像按浅色背景设计，暗色主题下加载完成后垫浅色底，加载前保持深色占位。 -->
  <img
    :src="src"
    :alt="owner.login"
    loading="lazy"
    :data-loaded="loaded || undefined"
    class="bg-gray-100 ring-1 ring-black/5 dark:bg-neutral-800 dark:ring-white/10 dark:data-loaded:bg-neutral-100"
    :class="owner.type === 'Organization' ? 'rounded-[22%]' : 'rounded-full'"
    @load="loaded = true"
  />
</template>
