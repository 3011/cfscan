import { describe, expect, it } from 'vitest'
import { formatMS, formatNumber, formatPercent, formatRelativeTime } from '@/lib/format'

describe('format helpers', () => {
  it('formats latency with adaptive precision', () => {
    expect(formatMS(8.42)).toBe('8.4 ms')
    expect(formatMS(234.5)).toBe('235 ms')
    expect(formatMS(0)).toBe('—')
  })

  it('formats numeric values for the Chinese locale', () => {
    expect(formatNumber(6171)).toBe('6,171')
    expect(formatPercent(12.4)).toBe('12%')
  })

  it('formats relative time from an injected clock', () => {
    const base = new Date('2026-07-16T06:00:00Z').getTime()
    expect(formatRelativeTime('2026-07-16T05:59:59Z', base)).toBe('1 秒前')
    expect(formatRelativeTime('2026-07-16T05:59:58Z', base)).toBe('2 秒前')
  })

})
