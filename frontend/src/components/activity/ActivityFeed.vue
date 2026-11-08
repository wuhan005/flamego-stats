<script setup lang="ts">
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Activity } from '@/api/activity'
import ActivityIcon from '@/components/activity/ActivityIcon.vue'
import OwnerAvatar from '@/components/common/OwnerAvatar.vue'
import { formatDateTime } from '@/utils/format'
import { localDate, parseLocalDate, type ActivityDay, type ActivityGroup } from '@/utils/activity'

defineProps<{ days: ActivityDay[] }>()

const { t, locale } = useI18n()

/** The number of the items shown in a collapsed group. */
const COLLAPSED = 3
const expanded = reactive(new Set<string>())

function toggle(key: string) {
  if (expanded.has(key)) expanded.delete(key)
  else expanded.add(key)
}

function visibleItems(group: ActivityGroup) {
  return expanded.has(group.key) ? group.items : group.items.slice(0, COLLAPSED)
}

/** Returns the name of the day, and the full date as the hint for today and yesterday. */
function dayLabel(date: string): { label: string; hint: string } {
  const day = parseLocalDate(date)
  const yesterday = new Date()
  yesterday.setDate(yesterday.getDate() - 1)
  const full = new Intl.DateTimeFormat(locale.value, {
    year: day.getFullYear() === new Date().getFullYear() ? undefined : 'numeric',
    month: 'long',
    day: 'numeric',
    weekday: 'short',
  }).format(day)

  if (date === localDate(new Date())) return { label: t('activity.today'), hint: full }
  if (date === localDate(yesterday)) return { label: t('activity.yesterday'), hint: full }
  return { label: full, hint: '' }
}

function timeOf(iso: string): string {
  return new Intl.DateTimeFormat(locale.value, { hour: '2-digit', minute: '2-digit' }).format(new Date(iso))
}

/** Returns the short reference of the subject, e.g. #42 of a pull request. */
function reference(activity: Activity): string {
  switch (activity.type) {
    case 'commit':
      return activity.sha.slice(0, 7)
    case 'release':
      return activity.tag
    default:
      return `#${activity.number}`
  }
}

function itemTitle(activity: Activity): string {
  // A release is usually named after its tag, which is shown as the reference already.
  return activity.type === 'release' && activity.title === activity.tag ? '' : activity.title
}
</script>

<template>
  <div class="space-y-8">
    <section v-for="day in days" :key="day.date">
      <h3 class="mb-4 flex items-baseline gap-2 text-sm">
        <span class="font-semibold">{{ dayLabel(day.date).label }}</span>
        <span class="text-gray-500 dark:text-neutral-400">{{ dayLabel(day.date).hint }}</span>
      </h3>

      <ol>
        <li v-for="(group, index) in day.groups" :key="group.key" class="relative flex gap-3 pb-6 last:pb-0">
          <span
            v-if="index < day.groups.length - 1"
            class="absolute top-10 bottom-2 left-4 w-px -translate-x-1/2 bg-gray-200 dark:bg-neutral-800"
            aria-hidden="true"
          />
          <ActivityIcon :type="group.type" :action="group.action" />

          <div class="min-w-0 flex-1">
            <div class="flex min-h-8 items-center gap-3">
              <p class="min-w-0 flex-1 text-sm text-gray-600 dark:text-neutral-400">
                <i18n-t :keypath="`activity.events.${group.type}.${group.action}`" :plural="group.items.length" scope="global">
                  <template #actor>
                    <component
                      :is="group.actor.htmlURL ? 'a' : 'span'"
                      :href="group.actor.htmlURL || undefined"
                      target="_blank"
                      rel="noopener"
                      class="inline-flex items-center gap-1.5 align-bottom font-medium text-gray-900 hover:underline dark:text-gray-100"
                    >
                      <OwnerAvatar
                        :owner="{ ...group.actor, type: group.actor.bot ? 'Bot' : 'User' }"
                        :size="40"
                        class="size-5 shrink-0"
                      />
                      {{ group.actor.login }}
                    </component>
                  </template>
                  <template #count>{{ group.items.length }}</template>
                </i18n-t>
              </p>
              <time
                class="shrink-0 text-xs text-gray-400 tabular-nums dark:text-neutral-500"
                :datetime="group.items[0].occurredAt"
                :title="formatDateTime(group.items[0].occurredAt, locale)"
              >
                {{ timeOf(group.items[0].occurredAt) }}
              </time>
            </div>

            <ul
              class="mt-2 divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-100 bg-gray-50/60 dark:divide-neutral-800 dark:border-neutral-800 dark:bg-neutral-800/30"
            >
              <li v-for="activity in visibleItems(group)" :key="activity.id">
                <a
                  :href="activity.htmlURL"
                  target="_blank"
                  rel="noopener"
                  class="group/item flex min-w-0 items-center gap-2 px-3 py-2 text-sm transition-colors hover:bg-gray-100/70 dark:hover:bg-neutral-800/70"
                >
                  <span class="shrink-0 font-medium text-gray-700 dark:text-neutral-300">{{ activity.repository.name }}</span>
                  <span class="shrink-0 font-mono text-xs text-gray-400 dark:text-neutral-500">{{ reference(activity) }}</span>
                  <span
                    v-if="itemTitle(activity)"
                    class="min-w-0 truncate text-gray-600 group-hover/item:text-gray-900 dark:text-neutral-400 dark:group-hover/item:text-gray-100"
                  >
                    {{ itemTitle(activity) }}
                  </span>
                  <span v-if="activity.draft" class="chip shrink-0 py-0! text-[11px]">{{ t('activity.draft') }}</span>
                  <span v-if="activity.prerelease" class="chip-warning shrink-0 py-0! text-[11px]">{{ t('activity.prerelease') }}</span>
                </a>
              </li>
            </ul>
            <button
              v-if="group.items.length > COLLAPSED"
              type="button"
              class="link focus-ring mt-2 rounded text-xs"
              @click="toggle(group.key)"
            >
              {{ expanded.has(group.key) ? t('activity.showLess') : t('activity.showMore', { count: group.items.length - COLLAPSED }) }}
            </button>
          </div>
        </li>
      </ol>
    </section>
  </div>
</template>
