import { reactive } from 'vue'

// Global toast queue.
export const toasts = reactive([])
let seq = 0

export function dismiss(id) {
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}

function push(type, title, msg, timeout) {
  const id = ++seq
  toasts.push({ id, type, title, msg })
  if (timeout) setTimeout(() => dismiss(id), timeout)
  return id
}

export const toast = {
  ok: (title, msg) => push('ok', title, msg, 3200),
  err: (title, msg) => push('err', title, msg, 6000),
  info: (title, msg) => push('info', title, msg, 3600),
  warn: (title, msg) => push('warn', title, msg, 4800),
}
