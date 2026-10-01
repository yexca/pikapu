import caughtUp from "@/assets/mascot/caught-up.webp"
import reading from "@/assets/mascot/reading.webp"
import searching from "@/assets/mascot/searching.webp"
import welcome from "@/assets/mascot/welcome.webp"
import { cn } from "@/lib/utils"

const poses = {
  welcome,
  reading,
  "caught-up": caughtUp,
  searching,
}

export type MascotPose = keyof typeof poses

/**
 * Pikapu's mascot, a messenger mage who delivers feeds. Purely decorative:
 * every place that shows her also has text saying the same thing.
 */
export function Mascot({
  pose,
  className,
}: {
  pose: MascotPose
  className?: string
}) {
  return (
    <img
      src={poses[pose]}
      alt=""
      aria-hidden="true"
      draggable={false}
      className={cn("h-40 w-auto shrink-0 select-none", className)}
    />
  )
}
