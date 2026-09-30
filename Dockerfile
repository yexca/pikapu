# syntax=docker/dockerfile:1

# ---- Frontend ----
FROM node:24-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
# vite.config.ts reads ../VERSION for the version shown in Settings.
COPY VERSION /src/VERSION
COPY frontend/ ./
RUN npm run build

# ---- Backend (embeds the frontend build) ----
FROM golang:1.26-alpine AS server
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-trimpath
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/frontend/dist ./web/dist
COPY VERSION /VERSION
RUN go build \
      -ldflags="-s -w -X pikapu/internal/buildinfo.Version=$(tr -d '\r\n' < /VERSION)" \
      -o /out/pikapu ./cmd/pikapu

# ---- Runtime ----
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
COPY --from=server /out/pikapu /usr/local/bin/pikapu
ENV PIKAPU_ADDR=:7660 \
    PIKAPU_DATA_DIR=/data
VOLUME /data
EXPOSE 7660
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["pikapu", "healthcheck"]
ENTRYPOINT ["pikapu"]
