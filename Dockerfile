# syntax=docker/dockerfile:1.7
# Build from monorepo root:
#   docker build -f services/commercial/contract-management/Dockerfile .
FROM golang:1.23-alpine AS build

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY shared/platform-go /src/shared/platform-go

WORKDIR /src/services/commercial/contract-management
COPY services/commercial/contract-management/go.mod services/commercial/contract-management/go.sum ./
RUN go mod download

COPY services/commercial/contract-management/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/contract-management .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/contract-management .
EXPOSE 4103
ENV GIN_MODE=release \
    ENVIRONMENT=production \
    PORT=4103
HEALTHCHECK --interval=15s --timeout=5s --start-period=25s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:4103/ready || exit 1
USER nobody
ENTRYPOINT ["/app/contract-management"]
