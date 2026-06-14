<!--
  Select — a themed, fully styleable dropdown replacing the native <select>,
  whose popup list browsers won't let us style. Options may be plain strings
  (value === label) or { value, label } objects. Supports v-model, keyboard
  navigation (↑/↓/Enter/Esc), outside-click close and flips upward when there
  isn't room below.
-->
<template>
  <div class="rs" :class="{ open, disabled }" ref="root">
    <button
      type="button"
      class="rs-trigger"
      :disabled="disabled"
      role="combobox"
      aria-haspopup="listbox"
      :aria-expanded="open"
      @click="toggle"
      @keydown="onKeydown"
    >
      <span class="rs-value" :class="{ placeholder: !selected }">{{ selected ? selected.label : placeholder }}</span>
      <Icon name="chevron-down" class="rs-caret" />
    </button>

    <transition name="rs-pop">
      <ul v-if="open" class="rs-menu" :class="{ up: dropUp }" role="listbox" ref="menu">
        <li
          v-for="(o, i) in normalized"
          :key="o.value"
          class="rs-opt"
          :class="{ sel: o.value === modelValue, hl: i === highlighted }"
          role="option"
          :aria-selected="o.value === modelValue"
          @click="choose(o)"
          @mouseenter="highlighted = i"
        >
          <span class="rs-opt-label">{{ o.label }}</span>
          <Icon v-if="o.value === modelValue" name="check" class="rs-check" />
        </li>
      </ul>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onBeforeUnmount } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '请选择…' },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const root = ref(null)
const menu = ref(null)
const open = ref(false)
const dropUp = ref(false)
const highlighted = ref(-1)

// Normalize "tcp" → { value: 'tcp', label: 'tcp' } so callers can pass either shape.
const normalized = computed(() =>
  props.options.map((o) => (typeof o === 'object' && o !== null ? o : { value: o, label: String(o) })),
)
const selected = computed(() => normalized.value.find((o) => o.value === props.modelValue) || null)
const selectedIndex = computed(() => normalized.value.findIndex((o) => o.value === props.modelValue))

function toggle() {
  open.value ? close() : openMenu()
}

async function openMenu() {
  if (props.disabled) return
  open.value = true
  highlighted.value = selectedIndex.value >= 0 ? selectedIndex.value : 0
  await nextTick()
  // Flip upward if the popup would overflow the viewport.
  const r = root.value?.getBoundingClientRect()
  const h = menu.value?.offsetHeight || 0
  dropUp.value = !!r && r.bottom + h + 12 > window.innerHeight && r.top - h > 12
  menu.value?.querySelector('.rs-opt.hl')?.scrollIntoView({ block: 'nearest' })
}

function close() {
  open.value = false
}

function choose(o) {
  emit('update:modelValue', o.value)
  close()
}

function move(delta) {
  const n = normalized.value.length
  if (!n) return
  highlighted.value = (highlighted.value + delta + n) % n
  nextTick(() => menu.value?.querySelector('.rs-opt.hl')?.scrollIntoView({ block: 'nearest' }))
}

function onKeydown(e) {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      open.value ? move(1) : openMenu()
      break
    case 'ArrowUp':
      e.preventDefault()
      open.value ? move(-1) : openMenu()
      break
    case 'Enter':
    case ' ':
      e.preventDefault()
      if (open.value && normalized.value[highlighted.value]) choose(normalized.value[highlighted.value])
      else openMenu()
      break
    case 'Escape':
      if (open.value) {
        e.preventDefault()
        close()
      }
      break
    case 'Tab':
      close()
      break
  }
}

function onDocClick(e) {
  if (open.value && root.value && !root.value.contains(e.target)) close()
}

onMounted(() => document.addEventListener('mousedown', onDocClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))
</script>

<style scoped>
.rs {
  position: relative;
  width: 100%;
}

/* Trigger mirrors the .input / .select look from the theme. */
.rs-trigger {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 13px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-strong);
  background: var(--surface);
  color: var(--ink);
  font-family: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s var(--ease), box-shadow 0.2s var(--ease), background 0.2s;
}
.rs-trigger:hover:not(:disabled) {
  border-color: var(--accent);
}
.rs.open .rs-trigger,
.rs-trigger:focus-visible {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3.5px var(--accent-soft);
}
.rs.disabled .rs-trigger {
  opacity: 0.55;
  cursor: not-allowed;
}
.rs-value {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rs-value.placeholder {
  color: var(--ink-faint);
}
.rs-caret {
  width: 17px;
  height: 17px;
  flex-shrink: 0;
  color: var(--ink-faint);
  transition: transform 0.25s var(--ease-out), color 0.2s var(--ease);
}
.rs.open .rs-caret {
  transform: rotate(180deg);
  color: var(--accent-ink);
}

/* Popup list */
.rs-menu {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 60;
  margin: 0;
  padding: 5px;
  list-style: none;
  max-height: 264px;
  overflow-y: auto;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-lg);
}
.rs-menu.up {
  top: auto;
  bottom: calc(100% + 6px);
}
.rs-opt {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 11px;
  border-radius: var(--radius-xs);
  font-size: 14px;
  color: var(--ink);
  cursor: pointer;
  transition: background 0.12s var(--ease), color 0.12s var(--ease);
}
.rs-opt-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rs-opt.hl:not(.sel) {
  background: var(--surface-2);
}
.rs-opt.sel {
  background: var(--accent-soft);
  color: var(--accent-ink);
  font-weight: 600;
}
.rs-check {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  color: var(--accent-ink);
}

/* open / close transition */
.rs-pop-enter-active {
  transition: opacity 0.18s var(--ease-out), transform 0.18s var(--ease-out);
}
.rs-pop-leave-active {
  transition: opacity 0.13s var(--ease), transform 0.13s var(--ease);
}
.rs-pop-enter-from,
.rs-pop-leave-to {
  opacity: 0;
  transform: translateY(-6px) scale(0.98);
}
.rs-menu.up.rs-pop-enter-from,
.rs-menu.up.rs-pop-leave-to {
  transform: translateY(6px) scale(0.98);
}
</style>
