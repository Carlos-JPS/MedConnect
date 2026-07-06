module github.com/MedConnect/api-gateway

go 1.26.2

require (
	github.com/Carlos-JPS/medconnect/availability-service v0.0.0
	github.com/MedConnect/auth-service v0.0.0-00010101000000-000000000000
	github.com/MedConnect/booking-service v0.0.0
	github.com/prometheus/client_golang v1.23.2
	github.com/sllanoscaro/payment-service v0.0.0
	google.golang.org/grpc v1.81.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
)

replace github.com/MedConnect/booking-service => ../booking-service

replace github.com/sllanoscaro/payment-service => ../payment-service

replace github.com/Carlos-JPS/medconnect/availability-service => ../availability-service

replace github.com/MedConnect/auth-service => ../auth-service
