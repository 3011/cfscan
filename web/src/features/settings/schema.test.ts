import { describe, expect, it } from 'vitest'
import { defaultScanScheduleValues, scanScheduleSchema } from '@/features/settings/schema'

describe('scanScheduleSchema', () => {
  it('accepts standard cron and descriptors', () => {
    expect(scanScheduleSchema.safeParse(defaultScanScheduleValues).success).toBe(true)
    expect(scanScheduleSchema.safeParse({ ...defaultScanScheduleValues, cron_expression: '@daily' }).success).toBe(true)
  })

  it('rejects malformed cron expressions', () => {
    expect(scanScheduleSchema.safeParse({ ...defaultScanScheduleValues, cron_expression: '* * *' }).success).toBe(false)
  })
})

import { blacklistRecheckSchema, sourceSyncSchema } from '@/features/settings/schema'

describe('automation setting schemas', () => {
  it('accepts a complete blacklist recovery policy', () => {
    expect(blacklistRecheckSchema.safeParse({
      enabled: true,
      cron_expression: '*/15 * * * *',
      timezone: 'Asia/Shanghai',
      due_only: true,
      fraction: 0.5,
      max_targets: 500,
      skip_if_running: true,
      attempts: 3,
      timeout_ms: 5000,
      max_latency_ms: 1000,
      max_packet_loss: 50,
      retry_minutes: 120,
    }).success).toBe(true)
  })

  it('rejects an invalid blacklist selection ratio', () => {
    expect(blacklistRecheckSchema.safeParse({
      enabled: true,
      cron_expression: '*/15 * * * *',
      timezone: 'Asia/Shanghai',
      due_only: true,
      fraction: 1.5,
      max_targets: 500,
      skip_if_running: true,
      attempts: 3,
      timeout_ms: 5000,
      max_latency_ms: 1000,
      max_packet_loss: 50,
      retry_minutes: 120,
    }).success).toBe(false)
  })

  it('accepts source synchronization startup behavior', () => {
    expect(sourceSyncSchema.safeParse({ enabled: true, cron_expression: '0 */6 * * *', timezone: 'Asia/Shanghai', run_on_startup: true }).success).toBe(true)
  })
})
