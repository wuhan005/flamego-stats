<script setup lang="ts">
import { Languages, Monitor, Moon, Sun } from '@lucide/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import logo from '@/assets/logo.png'
import GitHubIcon from '@/components/common/GitHubIcon.vue'
import { useTheme, type ThemeMode } from '@/composables/useTheme'
import { LOCALES, setLocale, type Locale } from '@/i18n'

const { t, locale } = useI18n()
const { mode, setMode } = useTheme()

const THEME_ORDER: ThemeMode[] = ['system', 'light', 'dark']
const themeIcon = computed(() => ({ light: Sun, dark: Moon, system: Monitor })[mode.value])
const themeLabel = computed(
  () => ({ light: t('nav.themeLight'), dark: t('nav.themeDark'), system: t('nav.themeSystem') })[mode.value],
)

function cycleTheme() {
  setMode(THEME_ORDER[(THEME_ORDER.indexOf(mode.value) + 1) % THEME_ORDER.length])
}

const nextLocale = computed(() => LOCALES.find((l) => l.value !== locale.value) ?? LOCALES[0])

function toggleLocale() {
  setLocale(nextLocale.value.value as Locale)
}

const navLinks = computed(() => [
  { to: { name: 'home' }, label: t('nav.overview') },
  { to: { name: 'repositories' }, label: t('nav.repositories') },
])
</script>

<template>
  <header
    class="sticky top-0 z-20 border-b border-gray-200 bg-white/85 backdrop-blur-md dark:border-neutral-800 dark:bg-dark/80"
  >
    <nav class="mx-auto flex h-(--navbar-height) max-w-7xl items-center gap-6 px-6">
      <RouterLink :to="{ name: 'home' }" class="focus-ring flex shrink-0 items-center gap-2 rounded-md">
        <img :src="logo" alt="Flamego" class="size-7 rounded-md" />
        <span class="text-lg font-semibold tracking-tight">
          flamego<span class="text-flame-orange">.</span>
        </span>
        <span
          class="rounded-full border border-gray-200 px-2 py-px text-[11px] font-medium text-gray-500 dark:border-neutral-700 dark:text-neutral-400"
        >
          stats
        </span>
      </RouterLink>

      <div class="hidden items-center gap-1 sm:flex">
        <RouterLink
          v-for="link in navLinks"
          :key="link.label"
          :to="link.to"
          class="focus-ring rounded-md px-3 py-1.5 text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-neutral-400 dark:hover:text-gray-100"
          exact-active-class="font-medium text-gray-900! dark:text-gray-100!"
        >
          {{ link.label }}
        </RouterLink>
      </div>

      <div class="ml-auto flex items-center gap-1">
        <a
          href="https://flamego.dev/"
          target="_blank"
          rel="noopener"
          class="focus-ring hidden rounded-md px-3 py-1.5 text-sm text-gray-600 transition-colors hover:text-gray-900 md:inline-flex dark:text-neutral-400 dark:hover:text-gray-100"
        >
          {{ t('nav.docs') }}
        </a>
        <button
          type="button"
          class="focus-ring inline-flex h-8 items-center gap-1.5 rounded-md px-2 text-sm text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-gray-100"
          :title="t('nav.toggleLanguage')"
          @click="toggleLocale"
        >
          <Languages class="size-4" />
          <span class="hidden md:inline">{{ nextLocale.label }}</span>
        </button>
        <button
          type="button"
          class="focus-ring inline-flex size-8 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-gray-100"
          :title="`${t('nav.toggleTheme')}: ${themeLabel}`"
          :aria-label="t('nav.toggleTheme')"
          @click="cycleTheme"
        >
          <component :is="themeIcon" class="size-4" />
        </button>
        <a
          href="https://github.com/flamego/flamego"
          target="_blank"
          rel="noopener"
          class="focus-ring inline-flex size-8 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-gray-100"
          aria-label="GitHub"
        >
          <GitHubIcon class="size-5" />
        </a>
      </div>
    </nav>

    <div class="flex gap-1 border-t border-gray-200 px-4 py-1.5 sm:hidden dark:border-neutral-800">
      <RouterLink
        v-for="link in navLinks"
        :key="link.label"
        :to="link.to"
        class="rounded-md px-3 py-1 text-sm text-gray-600 dark:text-neutral-400"
        exact-active-class="bg-gray-100 text-gray-900! dark:bg-neutral-800 dark:text-gray-100!"
      >
        {{ link.label }}
      </RouterLink>
    </div>
  </header>
</template>
