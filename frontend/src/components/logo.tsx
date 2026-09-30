import { useId } from "react"

import { cn } from "@/lib/utils"

export function Logo({ className }: { className?: string }) {
  // Each instance needs its own gradient id: a url(#id) reference to a copy
  // inside a hidden subtree would render nothing.
  const gradient = useId()
  return (
    <svg
      viewBox="0 0 64 64"
      aria-hidden="true"
      className={cn("size-7 shrink-0", className)}
    >
      <defs>
        <linearGradient id={gradient} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#fb923c" />
          <stop offset="1" stopColor="#ea580c" />
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx="16" fill={`url(#${gradient})`} />
      <circle cx="21" cy="43" r="5.5" fill="#fff" />
      <path
        d="M17 32A15 15 0 0 1 32 47"
        fill="none"
        stroke="#fff"
        strokeWidth="6"
        strokeLinecap="round"
      />
      <path
        d="M17 19A28 28 0 0 1 45 47"
        fill="none"
        stroke="#fff"
        strokeWidth="6"
        strokeLinecap="round"
      />
    </svg>
  )
}
