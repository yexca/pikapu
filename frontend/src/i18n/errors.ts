import { ApiError } from "@/lib/api"
import type { Feed } from "@/lib/types"

import i18n from "."

function byCode(code: string, detail: string): string | undefined {
  const key = `errors.${code}`
  if (!code || !i18n.exists(key)) return undefined
  // Codes come from the server, so the key can't be checked statically.
  return i18n.t(key as "errors.internal", { detail })
}

/** Localized text for an error thrown by the API client. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    return byCode(err.code, err.message) ?? err.message
  }
  return err instanceof Error ? err.message : String(err)
}

/** Localized reason for a feed's most recent fetch failure. */
export function feedErrorMessage(
  feed: Pick<Feed, "last_error" | "last_error_code">
): string {
  return byCode(feed.last_error_code, feed.last_error) ?? feed.last_error
}
