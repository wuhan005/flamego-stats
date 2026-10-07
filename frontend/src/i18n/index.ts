import { createI18n } from 'vue-i18n'

import en from '@/locales/en'
import zhCN from '@/locales/zh-CN'

export type Locale = 'en' | 'zh-CN'

export const LOCALES: { value: Locale; label: string }[] = [
  { value: 'en', label: 'English' },
  { value: 'zh-CN', label: '简体中文' },
]

const STORAGE_KEY = 'flamego-stats.lang'

/** Returns the saved language, or the language of the browser. */
function detectLocale(): Locale {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved === 'en' || saved === 'zh-CN') return saved
  } catch {
    // Ignore the unavailable storage, e.g. in private browsing.
  }
  return navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en'
}

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: { en, 'zh-CN': zhCN },
})

export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale
  document.documentElement.lang = locale
  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    // Ignore the unavailable storage.
  }
}
