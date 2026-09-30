import { useEffect, useState } from "react"

export function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(t)
  }, [value, delay])
  return debounced
}

/** Re-renders the caller periodically, for relative timestamps. */
export function useTick(interval: number) {
  const [, setTick] = useState(0)
  useEffect(() => {
    const t = setInterval(() => setTick((n) => n + 1), interval)
    return () => clearInterval(t)
  }, [interval])
}
