import type { Entry } from "./types"

/** Consecutive updates shown as one card: one feed's entries, or one entry. */
export interface Stack {
  key: string
  feedId: number | null
  entries: Entry[]
}

export interface DayGroup {
  key: string
  date: Date
  stacks: Stack[]
}

/**
 * Groups a newest-first entry list by local day and, when byFeed is set, by
 * feed within each day, like notifications grouped by app. Stacks keep the
 * order of their newest entry.
 */
export function groupByDay(entries: Entry[], byFeed: boolean): DayGroup[] {
  const days: DayGroup[] = []
  let day: DayGroup | undefined
  let stacks = new Map<number, Stack>()
  for (const entry of entries) {
    const date = new Date(entry.published_at)
    const key = `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`
    if (day?.key !== key) {
      day = { key, date, stacks: [] }
      days.push(day)
      stacks = new Map()
    }
    if (!byFeed) {
      day.stacks.push({
        key: `${key}/e${entry.id}`,
        feedId: null,
        entries: [entry],
      })
      continue
    }
    let stack = stacks.get(entry.feed_id)
    if (!stack) {
      stack = {
        key: `${key}/f${entry.feed_id}`,
        feedId: entry.feed_id,
        entries: [],
      }
      stacks.set(entry.feed_id, stack)
      day.stacks.push(stack)
    }
    stack.entries.push(entry)
  }
  return days
}

/** The entries of grouped days in display order, for keyboard navigation. */
export function flattenDays(days: DayGroup[]): Entry[] {
  return days.flatMap((d) => d.stacks.flatMap((s) => s.entries))
}
