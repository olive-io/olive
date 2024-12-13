module github.com/olive-io/olive/runner

go 1.23

replace (
	github.com/olive-io/olive/api => ../api
	github.com/olive-io/olive/client => ../client
	github.com/olive-io/olive/pkg => ../pkg
)

require (
	github.com/bytedance/sonic v1.12.5
	github.com/dgraph-io/badger/v4 v4.5.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.24.0
	github.com/olive-io/bpmn/schema v1.4.1
	github.com/olive-io/bpmn/v2 v2.2.3
	github.com/olive-io/olive/api v0.0.0-00010101000000-000000000000
	github.com/olive-io/olive/client v0.0.0-00010101000000-000000000000
	github.com/olive-io/olive/pkg v0.1.1
	github.com/panjf2000/ants/v2 v2.10.0
	github.com/prometheus/client_golang v1.20.5
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.10.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20220101234140-673ab2c3ae75
	go.etcd.io/etcd/client/v3 v3.5.17
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.68.1
)

require (
	github.com/ChrisTrenkamp/xsel v0.9.16 // indirect
	github.com/Chronokeeper/anyxml v0.0.0-20160530174208-54457d8e98c6 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.19.1 // indirect
	github.com/bytedance/sonic/loader v0.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/cockroachdb/errors v1.11.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto/v2 v2.0.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/expr-lang/expr v1.16.9 // indirect
	github.com/getsentry/sentry-go v0.18.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/goccmack/goutil v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/muyo/sno v1.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.etcd.io/etcd/api/v3 v3.5.17 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.17 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.0.0-20210923205945-b76863e36670 // indirect
	golang.org/x/net v0.31.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241118233622-e639e219e697 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
