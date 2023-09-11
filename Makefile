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
	protoc -I . -I $(GOPATH)/src/github.com/olive-io/olive/api -I $(GOPATH)/src/github.com/gogo/protobuf --gogo_out=:. $(ROOT)/api/api.proto && \
	protoc -I . -I $(GOPATH)/src/github.com/olive-io/olive/api -I $(GOPATH)/src/github.com/gogo/protobuf --gogo_out=:. $(ROOT)/api/rpc.proto

docker:


vet:
	go vet ./...

test: vet
	go test -v ./...

clean:
	rm -fr ./_output

