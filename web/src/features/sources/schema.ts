import { z } from 'zod'

export const createASNSourceSchema = z.object({
  asn: z.number().int().min(1, 'ASN 必须大于 0').max(4_294_967_295, 'ASN 超出有效范围'),
  name: z.string().trim().min(2, '请输入 ASN 名称').max(100),
  organization: z.string().trim().min(2, '请输入组织名称').max(160),
  enabled: z.boolean(),
})

export type CreateASNSourceValues = z.infer<typeof createASNSourceSchema>

export const defaultASNSourceValues: CreateASNSourceValues = {
  asn: 0,
  name: '',
  organization: 'Cloudflare, Inc.',
  enabled: true,
}
