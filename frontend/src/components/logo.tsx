import { cn } from "@/lib/utils"

export function Logo({ className }: { className?: string }) {
  // The installed-app icon doubles as the in-app logo, so both stay in step.
  return (
    <img
      src="/icon-192.png"
      alt=""
      aria-hidden="true"
      draggable={false}
      className={cn("size-7 shrink-0 select-none", className)}
    />
  )
}
