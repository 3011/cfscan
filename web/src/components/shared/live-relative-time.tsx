import { useSyncExternalStore } from 'react'
import { formatRelativeTime } from '@/lib/format'
import { getServerNowSnapshot, subscribeServerClock } from '@/lib/server-clock'

export function LiveRelativeTime({ value }: { value?: string | null }) {
  const now = useSyncExternalStore(subscribeServerClock, getServerNowSnapshot, getServerNowSnapshot)
  return <>{formatRelativeTime(value, now)}</>
}
