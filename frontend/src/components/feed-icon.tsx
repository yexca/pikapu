import { useState } from "react"

import { feedIconUrl } from "@/lib/api"
import type { Feed } from "@/lib/types"
import { cn } from "@/lib/utils"

// Feeds whose icon failed to load; shared so every list row skips the request.
const failed = new Set<number>()

function hue(seed: string) {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) % 360
  return h
}

export function FeedIcon({
  feed,
  className,
}: {
  feed?: Pick<Feed, "id" | "title">
  className?: string
}) {
  const [, rerender] = useState(0)
  const base = cn("size-4 shrink-0 rounded-[4px]", className)

  if (!feed || failed.has(feed.id)) {
    const title = feed?.title?.trim() || "?"
    return (
      <span
        aria-hidden="true"
        className={cn(
          base,
          "inline-flex items-center justify-center text-[0.6rem] leading-none font-semibold text-white"
        )}
        style={{ backgroundColor: `oklch(0.68 0.12 ${hue(title)})` }}
      >
        {Array.from(title)[0]?.toUpperCase()}
      </span>
    )
  }

  return (
    <img
      src={feedIconUrl(feed.id)}
      alt=""
      loading="lazy"
      // A light tile keeps dark monochrome favicons visible in dark mode.
      className={cn(base, "object-contain dark:bg-white dark:p-px")}
      onError={() => {
        failed.add(feed.id)
        rerender((n) => n + 1)
      }}
    />
  )
}
