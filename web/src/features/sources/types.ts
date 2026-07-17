export interface SourceStatus {
  source: string
  status: string
  prefix_count: number
  ipv4_count: number
  ipv6_count: number
  last_synced_at?: string
  last_error?: string
}

export interface ASNSource {
  asn: number
  name: string
  organization: string
  enabled: boolean
  managed: boolean
  status: string
  prefix_count: number
  ipv4_count: number
  ipv6_count: number
  last_synced_at?: string
  last_error?: string
  created_at: string
  updated_at: string
}

export interface CreateASNSource {
  asn: number
  name: string
  organization: string
  enabled?: boolean
}

export interface UpdateASNSource {
  name?: string
  organization?: string
  enabled?: boolean
}

export interface ASNSyncSummary {
  items: ASNSource[]
  synced: number
  failed: number
}
