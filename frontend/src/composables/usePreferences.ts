import { ref, watch, type Ref } from 'vue'

/** Returns a boolean saved in the local storage under key, it is false if not saved or the storage is unavailable. */
function persistedBoolean(key: string): Ref<boolean> {
  let saved = false
  try {
    saved = localStorage.getItem(key) === 'true'
  } catch {
    // Ignore the unavailable storage.
  }

  const value = ref(saved)
  watch(value, (next) => {
    try {
      localStorage.setItem(key, String(next))
    } catch {
      // Ignore the unavailable storage.
    }
  })
  return value
}

const includeForks = persistedBoolean('flamego-stats.includeForks')
const excludeBots = persistedBoolean('flamego-stats.excludeBots')

/** The preferences shared by all the pages. */
export function usePreferences() {
  return { includeForks, excludeBots }
}
