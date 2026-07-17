import { z } from 'zod'
import { createScanJobSchema, defaultScanJobValues } from '@/features/scans/schema'

export const scanScheduleSchema = createScanJobSchema.extend({
  enabled: z.boolean(),
  cron_expression: z.string().trim().min(1, '请输入 Cron 表达式').refine((value) => value.startsWith('@') || value.split(/\s+/).length === 5, '请输入标准 5 段 Cron，或使用 @hourly / @daily'),
  timezone: z.string().trim().min(1, '请选择时区'),
})

export type ScanScheduleValues = z.infer<typeof scanScheduleSchema>

export const defaultScanScheduleValues: ScanScheduleValues = {
  ...defaultScanJobValues,
  name: 'Cloudflare 定时扫描',
  enabled: true,
  cron_expression: '0 */6 * * *',
  timezone: 'Asia/Shanghai',
}

export const blacklistRecheckSchema = z.object({
  enabled: z.boolean(),
  cron_expression: z.string().trim().min(1, '请输入 Cron 表达式').refine((value) => value.startsWith('@') || value.split(/\s+/).length === 5, '请输入标准 5 段 Cron，或使用 @hourly / @daily'),
  timezone: z.string().trim().min(1, '请选择时区'),
  due_only: z.boolean(),
  fraction: z.number().min(0.01).max(1),
  max_targets: z.number().int().min(1).max(5_000),
  skip_if_running: z.boolean(),
  attempts: z.number().int().min(1).max(10),
  timeout_ms: z.number().int().min(500).max(30_000),
  max_latency_ms: z.number().positive().max(60_000),
  max_packet_loss: z.number().min(0).max(100),
  retry_minutes: z.number().int().min(1).max(10_080),
})

export type BlacklistRecheckValues = z.infer<typeof blacklistRecheckSchema>

export const sourceSyncSchema = z.object({
  enabled: z.boolean(),
  cron_expression: z.string().trim().min(1, '请输入 Cron 表达式').refine((value) => value.startsWith('@') || value.split(/\s+/).length === 5, '请输入标准 5 段 Cron，或使用 @hourly / @daily'),
  timezone: z.string().trim().min(1, '请选择时区'),
  run_on_startup: z.boolean(),
})

export type SourceSyncValues = z.infer<typeof sourceSyncSchema>
