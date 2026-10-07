# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend

WORKDIR /src/frontend
RUN npm install --global pnpm@10
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine3.24 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /src/frontend/dist ./frontend/dist
ARG BUILD_COMMIT
RUN --mount=type=cache,target=/root/.cache/go-build \
    for cmd in flamego-stats-server flamego-stats-worker; do \
      CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -mod=readonly -trimpath \
      -ldflags="-s -w -X github.com/wuhan005/flamego-stats/internal/appconst.BuildCommit=$BUILD_COMMIT" \
      -o /out/$cmd ./cmd/$cmd || exit 1; \
    done

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser -S -D -H -u 10001 -G app app

WORKDIR /home/app
COPY --from=build --chmod=0555 /out/flamego-stats-server /out/flamego-stats-worker ./

# Mount configuration at /home/app/config/flamego-stats.yaml, readable by UID 10001.
# The image runs the server by default, override the entrypoint with /home/app/flamego-stats-worker to run the worker.
USER 10001:10001

EXPOSE 8080
ENTRYPOINT ["/home/app/flamego-stats-server"]
