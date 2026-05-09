# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/availability-service .

# ── Run stage ─────────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /out/availability-service ./availability-service

EXPOSE 50051

CMD ["./availability-service"]
