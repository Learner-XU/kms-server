# ---- Build Stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy source + vendor (no network needed)
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o kms-server ./cmd/server/

# ---- Runtime Stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/kms-server .

EXPOSE 8000

ENTRYPOINT ["./kms-server"]
