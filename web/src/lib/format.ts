export function formatDate(value?: string | null) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(value))
}

export function formatDateInTimezone(value?: string | null, timeZone?: string) {
  if (!value) return '—'
  try {
    return new Intl.DateTimeFormat('zh-CN', {
      timeZone: timeZone || undefined,
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(new Date(value))
  } catch {
    return formatDate(value)
  }
}

export function formatNumber(value?: number | null) {
  return new Intl.NumberFormat('zh-CN').format(value ?? 0)
}

export function formatMS(value?: number | null) {
  if (!value || value <= 0) return '—'
  return `${value.toFixed(value >= 100 ? 0 : 1)} ms`
}

export function formatPercent(value?: number | null, digits = 0) {
  return `${(value ?? 0).toFixed(digits)}%`
}

export function formatRelativeTime(value?: string | null, now = Date.now()) {
  if (!value) return '—'
  const delta = now - new Date(value).getTime()
  const seconds = Math.round(delta / 1000)
  if (seconds < 60) return `${Math.max(seconds, 0)} 秒前`
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  return formatDate(value)
}
