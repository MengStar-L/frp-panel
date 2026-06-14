// Small formatting helpers shared across views.

export function formatBytes(n) {
  n = Number(n) || 0
  if (n < 1024) return n + ' B'
  const units = ['KB', 'MB', 'GB', 'TB', 'PB']
  let i = -1
  do {
    n /= 1024
    i++
  } while (n >= 1024 && i < units.length - 1)
  return n.toFixed(n < 10 ? 2 : 1) + ' ' + units[i]
}

export function formatDuration(sec) {
  sec = Math.max(0, Math.floor(Number(sec) || 0))
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  if (d) return `${d} 天 ${h} 小时`
  if (h) return `${h} 小时 ${m} 分`
  if (m) return `${m} 分 ${s} 秒`
  return `${s} 秒`
}

export function shortTime(iso) {
  if (!iso) return ''
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}
