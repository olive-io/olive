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


generate:
	cd $(GOPATH)/src && \
	protoc --go_out=. github.com/olive-io/olive/api/discoverypb/discovery.proto && \
	protoc --go_out=. github.com/olive-io/olive/api/discoverypb/openapi.proto && \
	protoc -I. -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --go-olive_out=. --go-olive_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative --openapiv2_out=. --openapiv2_opt use_go_templates=true github.com/olive-io/olive/api/gatewaypb/rpc.proto && \
	protoc -I. -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative github.com/olive-io/olive/api/olivepb/internal.proto && \
	protoc -I. -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative github.com/olive-io/olive/api/olivepb/raft.proto && \
	protoc -I. -I github.com/googleapis/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative github.com/olive-io/olive/api/olivepb/rpc.proto

	goimports -w api/*/**.go
	sed -i "" 's/json:"ref,omitempty"/json:"$$ref,omitempty"/g' api/discoverypb/openapi.pb.go
	sed -i "" 's/json:"applicationJson,omitempty"/json:"application\/json,omitempty"/g' api/discoverypb/openapi.pb.go
	sed -i "" 's/json:"applicationXml,omitempty"/json:"application\/xml,,omitempty"/g' api/discoverypb/openapi.pb.go
	sed -i "" 's/json:"applicationYaml,omitempty"/json:"application\/yaml,,omitempty"/g' api/discoverypb/openapi.pb.go
	rm -fr api/*/**swagger.json

docker:


vet:
	go vet ./...

test: vet
	go test -v ./...

clean:
	rm -fr ./_output

