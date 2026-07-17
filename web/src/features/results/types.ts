export interface ScanResult {
  id: number
  job_id: string
  job_name: string
  agent_id: string
  agent_name: string
  region: string
  continent: string
  target_ip: string
  available: boolean
  latency_ms: number
  packet_loss: number
  tcp_connect_ms: number
  tls_handshake_ms: number
  ttfb_ms: number
  total_ms: number
  http_status: number
  http_version: string
  tls_version: string
  colo: string
  colo_city: string
  colo_country: string
  colo_continent: string
  cf_ray: string
  error_code?: string
  scanned_at: string
}

export type ResultView = 'latest' | 'history'
export type ResultSort = 'target_ip' | 'agent_name' | 'colo' | 'available' | 'latency_ms' | 'packet_loss' | 'http_status' | 'scanned_at'
export type ResultOrder = 'asc' | 'desc'
export type ResultTimeRange = '1h' | '24h' | '7d' | '30d' | 'all'

export interface ResultFilters {
  view?: ResultView
  agent_id?: string
  job_id?: string
  region?: string
  continent?: string
  search?: string
  colo?: string
  colo_city?: string
  colo_country?: string
  colo_continent?: string
  available?: string | boolean
  time_range?: ResultTimeRange
  page?: number
  page_size?: number
  sort?: ResultSort
  order?: ResultOrder
}

export interface ResultFacetFilters {
  view?: ResultView
  agent_id?: string
  job_id?: string
  region?: string
  continent?: string
  search?: string
  available?: string | boolean
  time_range?: ResultTimeRange
}

export type ResultJobFilters = ResultFilters

export interface ResultStatusCounts {
  all: number
  available: number
  failed: number
}

export interface ResultPage {
  items: ScanResult[]
  total: number
  page: number
  page_size: number
  total_pages: number
  counts: ResultStatusCounts
}

export interface ResultColoFacet {
  code: string
  city: string
  country: string
  continent: string
  count: number
}

export interface ResultJobFacet {
  id: string
  name: string
  kind: string
  count: number
  created_at: string
}
