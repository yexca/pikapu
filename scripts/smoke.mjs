#!/usr/bin/env node
// Runtime smoke test for a built image. Starts a disposable container with
// password protection enabled, exercises the public HTTP contract, and
// removes the container. Needs no outbound network access: the only feed it
// subscribes to uses the reserved .invalid domain, which never resolves.
//
// Usage: node scripts/smoke.mjs [image]   (default image: pikapu:dev)
// Env:   PIKAPU_SMOKE_PORT                (default 17660)

import { spawnSync } from "node:child_process"

const image = process.argv[2] ?? "pikapu:dev"
const port = Number(process.env.PIKAPU_SMOKE_PORT ?? 17660)
const base = `http://127.0.0.1:${port}`
const password = "synthetic-password"
const container = `pikapu-smoke-${process.pid}`

function docker(args, { allowFail = false } = {}) {
  const r = spawnSync("docker", args, { encoding: "utf8" })
  if (r.error) throw r.error
  if (r.status !== 0 && !allowFail) {
    throw new Error(`docker ${args.join(" ")} failed:\n${r.stderr.trim()}`)
  }
  return r
}

let cookie = ""

async function call(method, path, { body, form, auth = true } = {}) {
  const headers = {}
  if (auth && cookie) headers.cookie = cookie
  let payload
  if (form) {
    payload = form
  } else if (body !== undefined) {
    payload = JSON.stringify(body)
    headers["content-type"] = "application/json"
  }
  const res = await fetch(base + path, { method, headers, body: payload })
  const text = await res.text()
  let json
  try {
    json = JSON.parse(text)
  } catch {
    json = undefined
  }
  return { status: res.status, headers: res.headers, text, json }
}

function check(name, ok, detail) {
  if (!ok) throw new Error(`✗ ${name}${detail ? `\n  ${detail}` : ""}`)
  console.log(`✓ ${name}`)
}

async function waitFor(label, timeoutMs, probe) {
  const deadline = Date.now() + timeoutMs
  for (;;) {
    try {
      const value = await probe()
      if (value) return value
    } catch {
      // Not ready yet.
    }
    if (Date.now() > deadline) throw new Error(`✗ timed out waiting for ${label}`)
    await new Promise((r) => setTimeout(r, 500))
  }
}

async function run() {
  await waitFor("the health check", 30_000, async () => {
    return (await call("GET", "/api/healthz")).status === 200
  })
  const health = await call("GET", "/api/healthz")
  check(
    "health check reports ok and a version",
    health.json?.status === "ok" && typeof health.json?.version === "string",
    JSON.stringify(health.json)
  )

  const index = await call("GET", "/")
  check("index serves the web app", index.status === 200 && index.text.includes('id="root"'))
  const deep = await call("GET", "/feeds/1")
  check("client routes fall back to index.html", deep.status === 200 && deep.text.includes('id="root"'))

  const anon = await call("GET", "/api/feeds", { auth: false })
  check("API requires sign-in", anon.status === 401 && anon.json?.code === "unauthorized")
  const wrong = await call("POST", "/api/auth/login", { body: { password: "wrong" } })
  check("wrong password is rejected", wrong.status === 401 && wrong.json?.code === "invalid_password")
  const login = await call("POST", "/api/auth/login", { body: { password } })
  const setCookie = login.headers.get("set-cookie") ?? ""
  cookie = setCookie.split(";")[0]
  check("sign-in sets an HttpOnly session cookie", login.status === 200 && /httponly/i.test(setCookie))

  const feeds = await call("GET", "/api/feeds")
  check("a fresh database has no feeds", feeds.status === 200 && feeds.json?.length === 0)

  const category = await call("POST", "/api/categories", { body: { name: "Example" } })
  check("categories can be created", category.status === 201 && category.json?.name === "Example")
  const duplicate = await call("POST", "/api/categories", { body: { name: "example" } })
  check(
    "category names are unique, ignoring case",
    duplicate.status === 409 && duplicate.json?.code === "category_exists"
  )

  const settings = await call("PUT", "/api/settings", {
    body: { refresh_interval_minutes: 1, retention_days: 30 },
  })
  check(
    "settings are validated",
    settings.status === 400 && settings.json?.code === "invalid_refresh_interval"
  )

  const badUrl = await call("POST", "/api/feeds", { body: { url: "ftp://feed.example.invalid" } })
  check("non-HTTP feed URLs are rejected", badUrl.status === 400 && badUrl.json?.code === "invalid_url")

  const opml =
    '<?xml version="1.0"?><opml version="2.0"><body><outline text="Example">' +
    '<outline text="Example Feed" xmlUrl="https://feed.example.invalid/rss"/>' +
    "</outline></body></opml>"
  const form = new FormData()
  form.append("file", new Blob([opml], { type: "text/xml" }), "subscriptions.opml")
  const imported = await call("POST", "/api/opml", { form })
  check("OPML import adds the feed", imported.status === 200 && imported.json?.added === 1, imported.text)

  const feed = await waitFor("the background fetch", 45_000, async () => {
    const list = (await call("GET", "/api/feeds")).json
    return list?.[0]?.last_fetched_at ? list[0] : null
  })
  check(
    "a failed fetch is recorded with a stable error code",
    /^fetch_/.test(feed.last_error_code) && feed.error_count >= 1,
    JSON.stringify(feed)
  )

  const exported = await call("GET", "/api/opml")
  check(
    "OPML export contains the subscription",
    exported.status === 200 && exported.text.includes("feed.example.invalid/rss")
  )

  const logout = await call("POST", "/api/auth/logout")
  check("sign-out succeeds", logout.status === 204)
}

async function main() {
  console.log(`Smoke testing ${image} on ${base}`)
  docker([
    "run", "-d", "--rm",
    "--name", container,
    "-p", `127.0.0.1:${port}:7660`,
    "-e", `PIKAPU_PASSWORD=${password}`,
    image,
  ])
  try {
    await run()
    console.log("\nSmoke test passed.")
  } catch (err) {
    console.error(`\n${err.message}`)
    const logs = docker(["logs", container], { allowFail: true })
    console.error(`\n--- container logs ---\n${logs.stdout}${logs.stderr}`)
    process.exitCode = 1
  } finally {
    docker(["rm", "-f", container], { allowFail: true })
  }
}

await main()
