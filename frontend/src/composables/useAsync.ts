import { ref, shallowRef, watch, type Ref, type WatchSource } from 'vue'

export interface AsyncState<T> {
  data: Ref<T | undefined>
  loading: Ref<boolean>
  error: Ref<string>
  reload: () => Promise<void>
}

/** Loads fn immediately and whenever sources change, the responses of the stale loads are dropped. */
export function useAsync<T>(fn: () => Promise<T>, sources: WatchSource[] = []): AsyncState<T> {
  const data = shallowRef<T>()
  const loading = ref(true)
  const error = ref('')
  let seq = 0

  async function reload() {
    const current = ++seq
    loading.value = true
    error.value = ''
    try {
      const result = await fn()
      if (current === seq) data.value = result
    } catch (e) {
      if (current === seq) error.value = e instanceof Error ? e.message : String(e)
    } finally {
      if (current === seq) loading.value = false
    }
  }

  watch(sources, reload, { immediate: true })
  return { data, loading, error, reload }
}
