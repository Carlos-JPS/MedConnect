module github.com/MedConnect/api-gateway

go 1.26.2

require (
	github.com/Carlos-JPS/medconnect/availability-service v0.0.0
	github.com/MedConnect/booking-service v0.0.0
	github.com/sllanoscaro/payment-service v0.0.0
	google.golang.org/grpc v1.81.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/golang/protobuf v1.5.4 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
)

replace github.com/MedConnect/booking-service => ../booking-service

replace github.com/sllanoscaro/payment-service => ../payment-service

replace github.com/Carlos-JPS/medconnect/availability-service => ../availability-service
