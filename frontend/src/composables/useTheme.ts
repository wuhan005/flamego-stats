import { computed, ref } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'flamego-stats.theme'
const media = window.matchMedia('(prefers-color-scheme: dark)')

function savedMode(): ThemeMode {
  try {
    const mode = localStorage.getItem(STORAGE_KEY)
    if (mode === 'light' || mode === 'dark' || mode === 'system') return mode
  } catch {
    // Ignore the unavailable storage.
  }
  return 'system'
}

const mode = ref<ThemeMode>(savedMode())
const systemDark = ref(media.matches)
media.addEventListener('change', (e) => {
  systemDark.value = e.matches
  apply()
})

const isDark = computed(() => mode.value === 'dark' || (mode.value === 'system' && systemDark.value))

function apply() {
  document.documentElement.classList.toggle('dark', isDark.value)
}

/** The theme follows the system by default like flamego.dev, the choice is saved in the local storage. */
export function useTheme() {
  function setMode(next: ThemeMode) {
    mode.value = next
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // Ignore the unavailable storage.
    }
    apply()
  }
  return { mode, isDark, setMode }
}
