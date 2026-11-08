<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Contributor } from '@/api/activity'
import OwnerAvatar from '@/components/common/OwnerAvatar.vue'

const props = defineProps<{ contributors: Contributor[] }>()

const { t } = useI18n()
const max = computed(() => Math.max(1, ...props.contributors.map((c) => c.count)))
</script>

<template>
  <ol class="space-y-1">
    <li v-for="contributor in contributors" :key="contributor.login">
      <component
        :is="contributor.htmlURL ? 'a' : 'div'"
        :href="contributor.htmlURL || undefined"
        target="_blank"
        rel="noopener"
        class="focus-ring flex items-center gap-3 rounded-lg px-2 py-1.5 transition-colors hover:bg-gray-50 dark:hover:bg-neutral-800/60"
      >
        <OwnerAvatar :owner="{ ...contributor, type: contributor.bot ? 'Bot' : 'User' }" :size="64" class="size-7 shrink-0" />
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1.5 text-sm">
            <span class="truncate font-medium">{{ contributor.login }}</span>
            <span v-if="contributor.bot" class="chip shrink-0 px-1.5! py-0! text-[10px]">{{ t('activity.bot') }}</span>
          </div>
          <div class="mt-1 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800">
            <div class="flame-gradient h-full rounded-full" :style="{ width: `${(contributor.count / max) * 100}%` }" />
          </div>
        </div>
        <span class="w-14 shrink-0 text-right text-xs font-medium text-gray-700 tabular-nums dark:text-neutral-200">
          {{ t('activity.contributorEvents', { count: contributor.count }, contributor.count) }}
        </span>
      </component>
    </li>
  </ol>
</template>
