module github.com/RAF-SI-2025/EXBanka-4-Backend/services/card-service

go 1.26.1

require (
	github.com/RAF-SI-2025/EXBanka-4-Backend/shared v0.0.0-00010101000000-000000000000
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.48.0
	google.golang.org/grpc v1.79.3
)

require (
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace github.com/RAF-SI-2025/EXBanka-4-Backend/shared => ../../shared
