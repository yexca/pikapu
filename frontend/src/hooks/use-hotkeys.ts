import { useEffect, useRef } from "react"

type Handlers = Record<string, () => void>

/**
 * Binds single-key shortcuts (matched on KeyboardEvent.key). Keys are ignored
 * while typing in a field, with modifier keys held, or when a dialog or menu
 * is open.
 */
export function useHotkeys(handlers: Handlers) {
  const ref = useRef(handlers)
  useEffect(() => {
    ref.current = handlers
  })

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return
      const target = e.target as HTMLElement | null
      if (
        target?.closest?.("input, textarea, select, [contenteditable='true']")
      ) {
        if (e.key === "Escape") target.blur()
        return
      }
      if (
        document.querySelector(
          '[role="dialog"], [role="alertdialog"], [role="menu"]'
        )
      ) {
        return
      }
      const handler = ref.current[e.key]
      if (handler) {
        e.preventDefault()
        handler()
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [])
}
