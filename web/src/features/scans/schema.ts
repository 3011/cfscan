import { z } from 'zod'

export const createScanJobSchema = z.object({
  name: z.string().trim().min(2, '请输入至少 2 个字符的任务名称').max(80, '任务名称不能超过 80 个字符'),
  agent_ids: z.array(z.string()),
  sampling_mode: z.enum(['count', 'one_per_prefix']),
  target_count: z.number().int().min(1).max(10_000),
  scheme: z.enum(['http', 'https']),
  hostname: z.string().trim().min(1, '请输入测试域名').refine((value) => !/[ /\\]/.test(value), '域名格式不正确'),
  path: z.string().trim().min(1, '请输入请求路径').refine((value) => value.startsWith('/'), '请求路径必须以 / 开头'),
  port: z.number().int().min(1).max(65_535),
  attempts: z.number().int().min(1).max(10),
  timeout_ms: z.number().int().min(500).max(30_000),
  max_latency_ms: z.number().positive().max(60_000),
  max_packet_loss: z.number().min(0).max(100),
  blacklist_minutes: z.number().int().min(1).max(10_080),
  include_ipv6: z.boolean(),
  include_blocked: z.boolean(),
})

export type CreateScanJobValues = z.infer<typeof createScanJobSchema>

export const defaultScanJobValues: CreateScanJobValues = {
  name: '',
  agent_ids: [],
  sampling_mode: 'count',
  target_count: 128,
  scheme: 'https',
  hostname: 'cloudflare.com',
  path: '/cdn-cgi/trace',
  port: 443,
  attempts: 3,
  timeout_ms: 5_000,
  max_latency_ms: 1_000,
  max_packet_loss: 50,
  blacklist_minutes: 60,
  include_ipv6: false,
  include_blocked: false,
}
