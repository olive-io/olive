module github.com/olive-io/olive/client

go 1.22.0

require (
	github.com/cockroachdb/errors v1.11.1
	github.com/google/go-cmp v0.6.0
	github.com/imdario/mergo v0.3.6
	github.com/olive-io/olive/apis v0.1.1
	github.com/spf13/pflag v1.0.5
	go.etcd.io/etcd/api/v3 v3.5.13
	go.etcd.io/etcd/client/v3 v3.5.13
	go.uber.org/zap v1.27.0
	golang.org/x/term v0.19.0
	google.golang.org/grpc v1.64.0
	k8s.io/apimachinery v0.30.1
	k8s.io/client-go v0.30.1
	k8s.io/klog/v2 v2.120.1
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/getsentry/sentry-go v0.18.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/tidwall/gjson v1.17.1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.13 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/oauth2 v0.20.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240513163218-0867130af1f8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240513163218-0867130af1f8 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.30.1 // indirect
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
)

replace github.com/olive-io/olive/apis => ../apis
