import { describe, expect, it } from 'vitest'
import { createASNSourceSchema } from '@/features/sources/schema'

describe('createASNSourceSchema', () => {
  it('accepts a valid ASN source', () => {
    expect(createASNSourceSchema.safeParse({
      asn: 13335,
      name: 'CLOUDFLARENET',
      organization: 'Cloudflare, Inc.',
      enabled: true,
    }).success).toBe(true)
  })

  it('rejects ASN values outside the 32-bit range', () => {
    expect(createASNSourceSchema.safeParse({
      asn: 4_294_967_296,
      name: 'Invalid ASN',
      organization: 'Cloudflare, Inc.',
      enabled: true,
    }).success).toBe(false)
  })
})
