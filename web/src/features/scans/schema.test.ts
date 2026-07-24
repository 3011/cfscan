import { describe, expect, it } from 'vitest'
import { createScanJobSchema, defaultScanJobValues } from '@/features/scans/schema'

describe('createScanJobSchema', () => {
  it('accepts the default operational scan profile after a name is provided', () => {
    const result = createScanJobSchema.safeParse({ ...defaultScanJobValues, name: '亚洲常规扫描' })
    expect(result.success).toBe(true)
  })

  it('rejects unsafe or invalid probe settings', () => {
    const result = createScanJobSchema.safeParse({
      ...defaultScanJobValues,
      name: '无效任务',
      hostname: 'https://example.com/path',
      path: 'cdn-cgi/trace',
      max_packet_loss: 120,
    })
    expect(result.success).toBe(false)
  })
})

it('每前缀模式不依赖用户填写的采样数量', () => {
  const result = createScanJobSchema.safeParse({
    ...defaultScanJobValues,
    name: '全前缀扫描',
    sampling_mode: 'one_per_prefix',
  })
  expect(result.success).toBe(true)
})


it('联赛模式要求有效的单 Agent 扫描预算', () => {
  expect(createScanJobSchema.safeParse({
    ...defaultScanJobValues,
    name: '最佳 IP 联赛',
    sampling_mode: 'league',
    target_count: 256,
  }).success).toBe(true)
  expect(createScanJobSchema.safeParse({
    ...defaultScanJobValues,
    name: '无效联赛预算',
    sampling_mode: 'league',
    target_count: 0,
  }).success).toBe(false)
})
