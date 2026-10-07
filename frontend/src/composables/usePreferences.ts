import { ref, watch } from 'vue'

const STORAGE_KEY = 'flamego-stats.includeForks'

function savedIncludeForks(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

const includeForks = ref(savedIncludeForks())
watch(includeForks, (value) => {
  try {
    localStorage.setItem(STORAGE_KEY, String(value))
  } catch {
    // Ignore the unavailable storage.
  }
})

/** The preferences shared by all the pages. */
export function usePreferences() {
  return { includeForks }
}
