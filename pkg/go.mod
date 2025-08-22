module github.com/olive-io/olive/pkg

go 1.23.7

replace github.com/olive-io/olive/api => ../api

require (
	github.com/coreos/go-semver v0.3.1
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.7
	go.etcd.io/etcd/client/pkg/v3 v3.6.4
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.75.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)
