FROM golang:1.26-alpine AS build

WORKDIR /workspace

COPY booking-service ./booking-service
COPY payment-service ./payment-service
COPY availability-service ./availability-service
COPY auth-service ./auth-service
COPY api-gateway ./api-gateway

WORKDIR /workspace/api-gateway
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api-gateway ./cmd/api-gateway

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

USER app
COPY --from=build /out/api-gateway /usr/local/bin/api-gateway

EXPOSE 8080

ENTRYPOINT ["api-gateway"]
