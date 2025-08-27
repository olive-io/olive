module github.com/olive-io/olive/runner

go 1.23.7

replace (
	github.com/olive-io/olive/api => ../api
	github.com/olive-io/olive/clientgo => ../clientgo
	github.com/olive-io/olive/pkg => ../pkg
)

require (
	github.com/google/uuid v1.6.0
	github.com/olive-io/olive/api v0.0.0-00010101000000-000000000000
	github.com/shirou/gopsutil/v3 v3.24.5
	go.uber.org/zap v1.27.0
)

require (
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/srikrsna/protoc-gen-gotag v1.0.2 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)
