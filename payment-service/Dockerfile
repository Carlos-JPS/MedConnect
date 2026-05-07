# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/payment-service .

# ── Run stage ─────────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /out/payment-service ./payment-service

EXPOSE 50051

CMD ["./payment-service"]
