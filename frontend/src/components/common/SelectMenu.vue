<script setup lang="ts" generic="T extends string">
import { Check, ChevronDown } from '@lucide/vue'
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    options: { value: T; label: string }[]
    /** Accessible name of the menu. */
    label?: string
    /** sm matches the height of SegmentedControl, md matches the height of the text fields. */
    size?: 'sm' | 'md'
  }>(),
  { label: undefined, size: 'md' },
)

const model = defineModel<T>({ required: true })

const id = useId()
const root = ref<HTMLElement>()
const list = ref<HTMLElement>()
const open = ref(false)
/** Index of the option highlighted by the pointer or the arrow keys. */
const active = ref(-1)

const selectedIndex = computed(() => props.options.findIndex((option) => option.value === model.value))
const selected = computed(() => props.options[selectedIndex.value])

function show() {
  open.value = true
  active.value = Math.max(0, selectedIndex.value)
}

function hide() {
  open.value = false
}

function choose(index: number) {
  const option = props.options[index]
  if (option) model.value = option.value
  hide()
}

function move(delta: number) {
  const n = props.options.length
  if (n) active.value = (active.value + delta + n) % n
}

function onKeydown(event: KeyboardEvent) {
  if (!open.value) {
    if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(event.key)) {
      event.preventDefault()
      show()
    }
    return
  }
  switch (event.key) {
    case 'ArrowDown':
      move(1)
      break
    case 'ArrowUp':
      move(-1)
      break
    case 'Home':
      active.value = 0
      break
    case 'End':
      active.value = props.options.length - 1
      break
    case 'Enter':
    case ' ':
      choose(active.value)
      break
    case 'Escape':
      hide()
      break
    case 'Tab':
      hide()
      return
    default:
      return
  }
  event.preventDefault()
}

function onPointerdown(event: PointerEvent) {
  if (!root.value?.contains(event.target as Node)) hide()
}

watch(open, (value) => {
  if (value) document.addEventListener('pointerdown', onPointerdown)
  else document.removeEventListener('pointerdown', onPointerdown)
})
onBeforeUnmount(() => document.removeEventListener('pointerdown', onPointerdown))

// Keep the highlighted option in view while moving with the keys.
watch(active, async (index) => {
  await nextTick()
  list.value?.querySelector<HTMLElement>(`[data-index="${index}"]`)?.scrollIntoView({ block: 'nearest' })
})
</script>

<template>
  <div ref="root" class="relative inline-flex min-w-0">
    <button
      type="button"
      role="combobox"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-controls="`${id}-list`"
      :aria-activedescendant="open && active >= 0 ? `${id}-${active}` : undefined"
      :aria-label="label"
      class="focus-ring inline-flex w-full min-w-0 items-center gap-2 rounded-lg transition-colors"
      :class="[
        size === 'sm'
          ? 'h-7 bg-gray-100 pr-2 pl-2.5 text-xs font-medium text-gray-700 hover:text-gray-900 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:text-gray-100'
          : 'h-9 bg-black/5 pr-2.5 pl-3 text-sm dark:bg-gray-50/10',
        open ? 'ring-2 ring-primary-500/50' : '',
      ]"
      @click="open ? hide() : show()"
      @keydown="onKeydown"
    >
      <!-- All the labels share one cell, so the button is as wide as the longest one and keeps its width. -->
      <span class="grid min-w-0 flex-1 text-left">
        <span
          v-for="option in options"
          :key="option.value"
          class="truncate [grid-area:1/1]"
          :class="option.value === selected?.value ? '' : 'invisible'"
          :aria-hidden="option.value !== selected?.value"
        >
          {{ option.label }}
        </span>
      </span>
      <ChevronDown class="size-4 shrink-0 text-gray-400 transition-transform" :class="open ? 'rotate-180' : ''" />
    </button>

    <Transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="-translate-y-1 opacity-0"
      leave-active-class="transition duration-75 ease-in"
      leave-to-class="-translate-y-1 opacity-0"
    >
      <ul
        v-show="open"
        :id="`${id}-list`"
        ref="list"
        role="listbox"
        :aria-label="label"
        class="absolute top-full right-0 z-30 mt-1.5 max-h-72 w-max max-w-80 min-w-full overflow-y-auto rounded-lg border border-gray-200 bg-white p-1 shadow-lg shadow-black/5 dark:border-neutral-700 dark:bg-neutral-900 dark:shadow-black/40"
        :class="size === 'sm' ? 'text-xs' : 'text-sm'"
      >
        <li
          v-for="(option, index) in options"
          :id="`${id}-${index}`"
          :key="option.value"
          role="option"
          :aria-selected="index === selectedIndex"
          :data-index="index"
          class="flex cursor-pointer items-center gap-2 rounded-md px-2"
          :class="[
            size === 'sm' ? 'py-1.5' : 'py-2',
            index === active ? 'bg-gray-100 dark:bg-neutral-800' : '',
            index === selectedIndex ? 'font-medium text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-neutral-400',
          ]"
          @pointerenter="active = index"
          @click="choose(index)"
        >
          <span class="min-w-0 flex-1 truncate" :title="option.label">{{ option.label }}</span>
          <Check v-if="index === selectedIndex" class="size-3.5 shrink-0 text-primary-600 dark:text-primary-400" />
        </li>
      </ul>
    </Transition>
  </div>
</template>
