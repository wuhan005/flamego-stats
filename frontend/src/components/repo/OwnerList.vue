<script setup lang="ts">
import { Building2, Star } from '@lucide/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import type { OwnerRank } from '@/api/stats'
import { compactNumber } from '@/utils/format'

const props = defineProps<{ owners: OwnerRank[] }>()

const { t, locale } = useI18n()
const max = computed(() => Math.max(1, ...props.owners.map((o) => o.repositories)))
</script>

<template>
  <ol class="space-y-1">
    <li v-for="(owner, index) in owners" :key="owner.login">
      <RouterLink
        :to="{ name: 'repositories', query: { owner: owner.login } }"
        class="focus-ring flex items-center gap-3 rounded-lg px-2 py-1.5 transition-colors hover:bg-gray-50 dark:hover:bg-neutral-800/60"
      >
        <span
          class="w-5 text-right text-xs font-semibold tabular-nums"
          :class="index < 3 ? 'flame-text' : 'text-gray-400 dark:text-neutral-500'"
        >
          {{ index + 1 }}
        </span>
        <img
          :src="`${owner.avatarURL}${owner.avatarURL.includes('?') ? '&' : '?'}s=64`"
          :alt="owner.login"
          loading="lazy"
          class="size-7 bg-gray-100 ring-1 ring-black/5 dark:bg-neutral-800 dark:ring-white/10"
          :class="owner.type === 'Organization' ? 'rounded-md' : 'rounded-full'"
        />
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1.5 text-sm">
            <span class="truncate font-medium">{{ owner.login }}</span>
            <Building2 v-if="owner.type === 'Organization'" class="size-3.5 shrink-0 text-gray-400" :aria-label="t('common.organization')" />
          </div>
          <div class="mt-1 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800">
            <div class="flame-gradient h-full rounded-full" :style="{ width: `${(owner.repositories / max) * 100}%` }" />
          </div>
        </div>
        <div class="w-20 shrink-0 text-right text-xs">
          <div class="font-medium text-gray-700 tabular-nums dark:text-neutral-200">
            {{ t('home.ownerRepos', { count: owner.repositories }, owner.repositories) }}
          </div>
          <div class="inline-flex items-center gap-0.5 text-gray-400 tabular-nums dark:text-neutral-500">
            <Star class="size-3" />{{ compactNumber(owner.stars, locale) }}
          </div>
        </div>
      </RouterLink>
    </li>
  </ol>
</template>
