import { describe, expect, it } from 'vitest'
import { calculateServerOffset } from '@/lib/server-clock'

describe('server clock calibration', () => {
  it('uses the request midpoint for a millisecond server timestamp', () => {
    expect(calculateServerOffset('1700000010000', 1_700_000_000_000, 1_700_000_000_200)).toBe(9_900)
  })

  it('accepts the standard HTTP Date header as a rollout fallback', () => {
    const server = 'Tue, 14 Nov 2023 22:13:30 GMT'
    expect(calculateServerOffset(server, 1_700_000_000_000, 1_700_000_000_000)).toBe(10_000)
  })

  it('ignores invalid timestamps', () => {
    expect(calculateServerOffset('not-a-time', 1, 2)).toBeNull()
  })
})
