module github.com/olive-io/olive

go 1.23.7

replace (
	github.com/olive-io/olive/api => ./api
	github.com/olive-io/olive/clientgo => ./clientgo
	github.com/olive-io/olive/console => ./console
	github.com/olive-io/olive/pkg => ./pkg
	github.com/olive-io/olive/runner => ./runner
	github.com/olive-io/olive/server => ./server
)

require github.com/olive-io/olive/clientgo v0.0.0-00010101000000-000000000000

require (
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/olive-io/olive/api v0.0.0-00010101000000-000000000000 // indirect
	github.com/srikrsna/protoc-gen-gotag v1.0.2 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)
