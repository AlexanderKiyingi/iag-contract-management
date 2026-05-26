FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/contract-management .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /out/contract-management .
EXPOSE 4103
ENV GIN_MODE=release
ENV PORT=4103
USER nobody
ENTRYPOINT ["/app/contract-management"]
