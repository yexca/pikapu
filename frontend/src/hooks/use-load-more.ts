import { useEffect } from "react"

/**
 * Calls fetchNextPage when the sentinel element comes within 400px of the
 * scroll container's viewport.
 */
export function useLoadMore(
  sentinelRef: React.RefObject<HTMLElement | null>,
  scrollRef: React.RefObject<HTMLElement | null>,
  {
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    itemCount,
  }: {
    hasNextPage: boolean
    isFetchingNextPage: boolean
    fetchNextPage: () => void
    /** Re-observes after new items render, in case the sentinel stays visible. */
    itemCount: number
  }
) {
  useEffect(() => {
    const el = sentinelRef.current
    if (!el || !hasNextPage) return
    const io = new IntersectionObserver(
      ([e]) => {
        if (e.isIntersecting && !isFetchingNextPage) fetchNextPage()
      },
      { root: scrollRef.current, rootMargin: "400px" }
    )
    io.observe(el)
    return () => io.disconnect()
  }, [
    sentinelRef,
    scrollRef,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
    itemCount,
  ])
}
