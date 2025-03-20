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

ifeq ($(GOHOSTOS), windows)
        #the `find.exe` is different from `find` in bash/shell.
        #to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
        #changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
        #Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
        Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
        TYPES_PROTO_FILES=$(shell $(Git_Bash) -c "find api/types -name *.proto")
        RPC_PROTO_FILES=$(shell $(Git_Bash) -c "find api/rpc -name *.proto")
        OPENAPI_PROTO_FILES=$(shell $(Git_Bash) -c "find api/rpc/consolepb -name *.proto")
else
        TYPES_PROTO_FILES=$(shell find api/types -name *.proto)
        RPC_PROTO_FILES=$(shell find api/rpc -name *.proto)
        OPENAPI_PROTO_FILES=$(shell find api/rpc/consolepb -name *.proto)
endif



all: build

vendor:
	go mod vendor

test-coverage:
	go test ./... -bench=. -coverage

lint:
	golint -set_exit_status ./..

install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/srikrsna/protoc-gen-gotag@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest
	go install github.com/google/wire/cmd/wire@latest


apis:
	protoc --proto_path=./api --go_out=paths=source_relative:./api api/errors/errors.proto
	protoc --proto_path=./api \
			--proto_path=./third_party \
			--go_out=paths=source_relative:./api \
			$(TYPES_PROTO_FILES)
	protoc --proto_path=. \
    		--proto_path=./api \
    		--proto_path=./third_party \
    		--gotag_out=paths=source_relative:. \
    		$(TYPES_PROTO_FILES)
	protoc --proto_path=./api \
    		--proto_path=./third_party \
    		--go_out=paths=source_relative:./api \
    		--go-grpc_out=paths=source_relative:./api \
    		--validate_out=paths=source_relative,lang=go:./api \
    		--grpc-gateway_out=paths=source_relative:./api \
    		$(RPC_PROTO_FILES)
	protoc --proto_path=./api \
			--proto_path=./third_party \
			--openapi_out=fq_schema_naming=true,title="olive",description="olive OpenAPI3.0 Document",version=$(GIT_TAG),default_response=false:./console/docs \
			$(OPENAPI_PROTO_FILES)


docker:


vet:
	go vet ./...

test: vet
	go test -v ./...

clean:
	rm -fr ./_output

