module github.com/olive-io/olive/server

go 1.23.7

replace (
	github.com/olive-io/olive/api => ../api
	github.com/olive-io/olive/pkg => ../pkg
)

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2
	github.com/olive-io/bpmn/schema v1.5.1
	github.com/olive-io/olive/api v0.0.0-00010101000000-000000000000
	github.com/olive-io/olive/pkg v0.0.0-00010101000000-000000000000
	github.com/panjf2000/ants/v2 v2.11.3
	github.com/prometheus/client_golang v1.23.0
	github.com/spf13/cobra v1.9.1
	github.com/tmc/grpc-websocket-proxy v0.0.0-20220101234140-673ab2c3ae75
	go.uber.org/zap v1.27.0
	golang.org/x/net v0.43.0
	google.golang.org/grpc v1.75.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.30.1
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.13.3 // indirect
	github.com/bytedance/sonic/loader v0.2.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	github.com/srikrsna/protoc-gen-gotag v1.0.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/arch v0.14.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
