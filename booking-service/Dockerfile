FROM golang:1.26-alpine AS build

WORKDIR /workspace

COPY payment-service ./payment-service
COPY availability-service ./availability-service
COPY booking-service ./booking-service

WORKDIR /workspace/booking-service
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/booking-service ./cmd/booking-service

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

USER app
COPY --from=build /out/booking-service /usr/local/bin/booking-service

EXPOSE 50051

ENTRYPOINT ["booking-service"]
