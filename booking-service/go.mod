module github.com/MedConnect/booking-service

go 1.26.2

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/sllanoscaro/payment-service v0.0.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
)

replace github.com/sllanoscaro/payment-service => ../payment-service
