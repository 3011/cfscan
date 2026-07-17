let clockOffsetMS = 0
let calibrated = false
let currentNow = Date.now()
let timer: ReturnType<typeof setInterval> | null = null
const listeners = new Set<() => void>()

function parseServerTime(value: string | null) {
  if (!value) return null
  const numeric = Number(value)
  if (Number.isFinite(numeric) && numeric > 0) return numeric
  const parsed = Date.parse(value)
  return Number.isFinite(parsed) ? parsed : null
}

export function calculateServerOffset(
  serverTime: string | null,
  requestStartedAt: number,
  responseReceivedAt: number,
) {
  const serverTimeMS = parseServerTime(serverTime)
  if (serverTimeMS === null || !Number.isFinite(requestStartedAt) || !Number.isFinite(responseReceivedAt)) return null
  const midpoint = requestStartedAt + Math.max(responseReceivedAt - requestStartedAt, 0) / 2
  return serverTimeMS - midpoint
}

export function synchronizeServerClock(
  serverTime: string | null,
  requestStartedAt: number,
  responseReceivedAt = Date.now(),
) {
  const measuredOffset = calculateServerOffset(serverTime, requestStartedAt, responseReceivedAt)
  if (measuredOffset === null) return

  // Apply the first measurement immediately so large client clock drift is fixed at once.
  // Smooth later samples to avoid visible jumps caused by network latency variance.
  clockOffsetMS = calibrated ? clockOffsetMS * 0.8 + measuredOffset * 0.2 : measuredOffset
  calibrated = true
  currentNow = Date.now() + clockOffsetMS
  listeners.forEach((listener) => listener())
}

export function subscribeServerClock(listener: () => void) {
  listeners.add(listener)
  if (!timer) {
    timer = setInterval(() => {
      currentNow = Date.now() + clockOffsetMS
      listeners.forEach((callback) => callback())
    }, 1_000)
  }
  return () => {
    listeners.delete(listener)
    if (!listeners.size && timer) {
      clearInterval(timer)
      timer = null
    }
  }
}

export function getServerNowSnapshot() {
  return currentNow
}
