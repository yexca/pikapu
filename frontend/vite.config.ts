import fs from "node:fs"
import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// The repository VERSION file is the single source of the app version.
function appVersion(): string {
  try {
    return fs
      .readFileSync(path.resolve(import.meta.dirname, "../VERSION"), "utf8")
      .trim()
  } catch {
    return "dev"
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    // A single-user app served from one container: one bundle is fine.
    chunkSizeWarningLimit: 900,
  },
  define: {
    __APP_VERSION__: JSON.stringify(appVersion()),
  },
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    // During development the Go backend runs separately on :7660.
    proxy: {
      "/api": "http://localhost:7660",
    },
  },
})
