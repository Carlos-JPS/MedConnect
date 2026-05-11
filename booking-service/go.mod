module github.com/MedConnect/booking-service

go 1.26.2

require (
	github.com/Carlos-JPS/medconnect/availability-service v0.0.0-00010101000000-000000000000
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.12.3
	github.com/sllanoscaro/payment-service v0.0.0
	google.golang.org/grpc v1.81.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
)

replace github.com/sllanoscaro/payment-service => ../payment-service

replace github.com/Carlos-JPS/medconnect/availability-service => ../availability-service
