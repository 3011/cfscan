import * as React from "react"

const MOBILE_BREAKPOINT = 768
const QUERY = `(max-width: ${MOBILE_BREAKPOINT - 1}px)`

function subscribe(callback: () => void) {
  if (typeof window === "undefined") return () => undefined
  const media = window.matchMedia(QUERY)
  media.addEventListener("change", callback)
  return () => media.removeEventListener("change", callback)
}

function getSnapshot() {
  return typeof window !== "undefined" && window.matchMedia(QUERY).matches
}

function getServerSnapshot() {
  return false
}

export function useIsMobile() {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}
