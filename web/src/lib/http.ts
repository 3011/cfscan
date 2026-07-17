import { synchronizeServerClock } from '@/lib/server-clock'

export type ItemsResponse<T> = { items: T[] }

type APIErrorBody = { error?: { code?: string; message?: string } }

export class APIError extends Error {
  readonly status: number
  readonly code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
  }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const requestStartedAt = Date.now()
  const response = await fetch(path, {
    credentials: 'same-origin',
    ...init,
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })
  const responseReceivedAt = Date.now()
  synchronizeServerClock(
    response.headers.get('X-CFScan-Server-Time') ?? response.headers.get('Date'),
    requestStartedAt,
    responseReceivedAt,
  )

  if (!response.ok) {
    if (response.status === 401 && path !== '/api/v1/auth/login' && typeof window !== 'undefined') {
      window.dispatchEvent(new Event('cfscan:unauthorized'))
    }
    let body: APIErrorBody | undefined
    try {
      body = (await response.json()) as APIErrorBody
    } catch {
      body = undefined
    }
    throw new APIError(body?.error?.message ?? `请求失败（${response.status}）`, response.status, body?.error?.code)
  }

  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export function createQueryString(values: object) {
  const params = new URLSearchParams()
  Object.entries(values).forEach(([key, value]) => {
    if (value !== undefined && value !== '') params.set(key, String(value))
  })
  const text = params.toString()
  return text ? `?${text}` : ''
}
