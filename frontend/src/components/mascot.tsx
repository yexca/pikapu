import caughtUp from "@/assets/mascot/caught-up.webp"
import oops from "@/assets/mascot/oops.webp"
import pointing from "@/assets/mascot/pointing.webp"
import reading from "@/assets/mascot/reading.webp"
import searching from "@/assets/mascot/searching.webp"
import starred from "@/assets/mascot/starred.webp"
import waiting from "@/assets/mascot/waiting.webp"
import welcome from "@/assets/mascot/welcome.webp"
import { cn } from "@/lib/utils"

const poses = {
  welcome,
  pointing,
  reading,
  "caught-up": caughtUp,
  searching,
  starred,
  waiting,
  oops,
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
