FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth-service ./cmd/auth-service

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

USER app
COPY --from=build /out/auth-service /usr/local/bin/auth-service

EXPOSE 50051

ENTRYPOINT ["auth-service"]
