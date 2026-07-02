FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notification-service ./cmd/notification-service

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

USER app
COPY --from=build /out/notification-service /usr/local/bin/notification-service

ENTRYPOINT ["notification-service"]
