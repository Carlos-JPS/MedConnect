FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/booking-service ./cmd/booking-service

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

USER app
COPY --from=build /out/booking-service /usr/local/bin/booking-service

EXPOSE 50051

ENTRYPOINT ["booking-service"]
