NAME=olive
IMAGE_NAME=olive-io/$(NAME)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --abbrev=0 --tags --always --match "v*")
GIT_VERSION=github.com/olive-io/olive/pkg/version
CGO_ENABLED=0
BUILD_DATE=$(shell date +%s)
LDFLAGS=-X $(GIT_VERSION).GitCommit=$(GIT_COMMIT) -X $(GIT_VERSION).GitTag=$(GIT_TAG) -X $(GIT_VERSION).BuildDate=$(BUILD_DATE)
IMAGE_TAG=$(GIT_TAG)-$(GIT_COMMIT)
ROOT=github.com/olive-io/olive

all: build

vendor:
	go mod vendor

test-coverage:
	go test ./... -bench=. -coverage

lint:
	golint -set_exit_status ./..

install:

genclients:
	cd api && ./hack/update-codegen.sh

proto:
	goproto-gen -p github.com/olive-io/olive/api/types/meta/v1
	goproto-gen --metadata-packages github.com/olive-io/olive/api/types/meta/v1 -p github.com/olive-io/olive/api/types/core/v1
	deepcopy-gen --bounding-dirs github.com/olive-io/olive/api/types/meta/v1,github.com/olive-io/olive/api/types/core/v1

	cd $(GOPATH)/src && \
	protoc -I . -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative github.com/olive-io/olive/api/rpc/planepb/rpc.proto && \
	protoc -I . -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative github.com/olive-io/olive/api/rpc/runnerpb/rpc.proto

generate:
	cd $(GOPATH)/src && \
	protoc --go_out=. --go_opt=paths=source_relative github.com/olive-io/olive/api/errors/errors.proto && \
    protoc --go_out=. --go_opt=paths=source_relative github.com/olive-io/olive/api/types/internal.proto && \
    protoc --go_out=. --go_opt=paths=source_relative github.com/olive-io/olive/api/types/bpmn.proto && \
	protoc -I. -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative github.com/olive-io/olive/api/rpc/monpb/rpc.proto
#	protoc -I. -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative github.com/olive-io/olive/api/rpc/runnerpb/rpc.proto
#
#	goimports -w api/*/**.go
#	sed -i.bak 's/json:"ref,omitempty"/json:"$$ref,omitempty"/g' api/discoverypb/openapi.pb.go
#	sed -i.bak 's/json:"applicationJson,omitempty"/json:"application\/json,omitempty"/g' api/discoverypb/openapi.pb.go
#	sed -i.bak 's/json:"applicationXml,omitempty"/json:"application\/xml,omitempty"/g' api/discoverypb/openapi.pb.go
#	sed -i.bak 's/json:"applicationYaml,omitempty"/json:"application\/yaml,omitempty"/g' api/discoverypb/openapi.pb.go
#	rm -fr api/*/**swagger.json api/*/**.bak

docker:


vet:
	go vet ./...

test: vet
	go test -v ./...

clean:
	rm -fr ./_output

