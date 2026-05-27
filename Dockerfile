# Standalone image for Railway and iag-contract-management repo root builds.
# Monorepo compose/CI: use Dockerfile.monorepo (--target monorepo).

FROM golang:1.23-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY third_party/platform-go /deps/platform-go
COPY go.mod go.sum ./
RUN go mod edit -replace=github.com/alvor-technologies/iag-platform-go=/deps/platform-go \
    && go mod download
COPY . .
RUN set -eu; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/contract-management .; \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/jobs ./cmd/jobs

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/contract-management .
COPY --from=build /out/jobs /app/jobs
EXPOSE 4103
ENV GIN_MODE=release \
    ENVIRONMENT=production \
    PORT=4103
HEALTHCHECK --interval=15s --timeout=5s --start-period=25s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:4103/ready || exit 1
USER nobody
ENTRYPOINT ["/app/contract-management"]
