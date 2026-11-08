<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  /** type is User, Organization or Bot, only the users have round avatars. avatarURL is empty for the accounts unknown to GitHub. */
  owner: { login: string; avatarURL: string; type: string }
  /** Requested side length of the avatar in pixels, usually twice the displayed size. */
  size: number
}>()

const src = computed(() => `${props.owner.avatarURL}${props.owner.avatarURL.includes('?') ? '&' : '?'}s=${props.size}`)
const loaded = ref(false)
</script>

<template>
  <!-- The transparent avatars are designed for light grounds, so they get one in the dark theme once loaded, keeping the dark placeholder before. -->
  <img
    v-if="owner.avatarURL"
    :src="src"
    :alt="owner.login"
    loading="lazy"
    :data-loaded="loaded || undefined"
    class="bg-gray-100 ring-1 ring-black/5 dark:bg-neutral-800 dark:ring-white/10 dark:data-loaded:bg-neutral-100"
    :class="owner.type === 'User' ? 'rounded-full' : 'rounded-[22%]'"
    @load="loaded = true"
  />
  <span
    v-else
    class="inline-flex items-center justify-center rounded-full bg-gray-100 text-[0.6em] font-semibold text-gray-500 uppercase ring-1 ring-black/5 dark:bg-neutral-800 dark:text-neutral-400 dark:ring-white/10"
    :aria-label="owner.login"
  >
    {{ owner.login.slice(0, 1) }}
  </span>
</template>
