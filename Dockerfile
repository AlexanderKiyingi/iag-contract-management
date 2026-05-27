# syntax=docker/dockerfile:1.7
#
# Targets:
#   standalone (default) — iag-contract-management repo root on Railway
#   monorepo             — IAG_multi_backend root context (deploy/docker-compose)
#
# Monorepo:  docker build -f services/commercial/contract-management/Dockerfile --target monorepo .
# Standalone: docker build -f Dockerfile --target standalone .

FROM golang:1.23-alpine AS base
RUN apk add --no-cache git ca-certificates
ENV PLATFORM_GO_DEP=/deps/platform-go

FROM base AS platform-go-clone
ARG IAG_META_REF=main
ARG IAG_META_REPO=https://github.com/AlexanderKiyingi/IAG_multi_backend.git
RUN git clone --depth 1 --branch "${IAG_META_REF}" "${IAG_META_REPO}" /tmp/iag \
    && mv /tmp/iag/shared/platform-go "${PLATFORM_GO_DEP}" \
    && rm -rf /tmp/iag

FROM base AS platform-go-copy
COPY shared/platform-go ${PLATFORM_GO_DEP}

# ─── Standalone iag-contract-management (repo root = service root) ─────────
FROM base AS build-standalone
COPY --from=platform-go-clone ${PLATFORM_GO_DEP} ${PLATFORM_GO_DEP}
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod edit -replace=github.com/alvor-technologies/iag-platform-go=${PLATFORM_GO_DEP} \
    && go mod download
COPY . .
RUN set -eu; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/contract-management .; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/jobs ./cmd/jobs

# ─── Monorepo (context = repo root) ────────────────────────────────────────
FROM base AS build-monorepo
COPY --from=platform-go-copy ${PLATFORM_GO_DEP} ${PLATFORM_GO_DEP}
WORKDIR /src/services/commercial/contract-management
COPY services/commercial/contract-management/go.mod services/commercial/contract-management/go.sum ./
RUN go mod edit -replace=github.com/alvor-technologies/iag-platform-go=${PLATFORM_GO_DEP} \
    && go mod download
COPY services/commercial/contract-management/ .
RUN set -eu; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/contract-management .; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/jobs ./cmd/jobs

FROM alpine:3.20 AS monorepo
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build-monorepo /out/contract-management .
COPY --from=build-monorepo /out/jobs /app/jobs
EXPOSE 4103
ENV GIN_MODE=release \
    ENVIRONMENT=production \
    PORT=4103
HEALTHCHECK --interval=15s --timeout=5s --start-period=25s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:4103/ready || exit 1
USER nobody
ENTRYPOINT ["/app/contract-management"]

FROM alpine:3.20 AS standalone
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build-standalone /out/contract-management .
COPY --from=build-standalone /out/jobs /app/jobs
EXPOSE 4103
ENV GIN_MODE=release \
    ENVIRONMENT=production \
    PORT=4103
HEALTHCHECK --interval=15s --timeout=5s --start-period=25s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:4103/ready || exit 1
USER nobody
ENTRYPOINT ["/app/contract-management"]
