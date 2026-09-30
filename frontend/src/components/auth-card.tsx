import { Logo } from "@/components/logo"

/** The centered card shared by the sign-in and setup pages. */
export function AuthCard({
  subtitle,
  onSubmit,
  children,
}: {
  subtitle: React.ReactNode
  onSubmit: () => void
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4">
      <form
        onSubmit={(e) => {
          e.preventDefault()
          onSubmit()
        }}
        className="w-full max-w-xs rounded-2xl border bg-background p-6 shadow-sm"
      >
        <div className="flex flex-col items-center gap-3 text-center">
          <Logo className="size-11" />
          <div>
            <h1 className="text-lg font-semibold tracking-tight">Pikapu</h1>
            <p className="text-xs text-muted-foreground">{subtitle}</p>
          </div>
        </div>
        <div className="mt-6 grid gap-3">{children}</div>
      </form>
    </div>
  )
}
