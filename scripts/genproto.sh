#!/usr/bin/env bash
#
# Generate all olive protobuf bindings.
# Run from repository root directory named olive.
#
set -euo pipefail

shopt -s globstar

if ! [[ "$0" =~ scripts/genproto.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

# Set SED variable
if LANG=C sed --help 2>&1 | grep -q GNU; then
  SED="sed"
elif command -v gsed &>/dev/null; then
  SED="gsed"
else
  echo "Failed to find GNU sed as sed or gsed. If you are on Mac: brew install gnu-sed." >&2
  exit 1
fi

source ./scripts/lib.sh

#GOFAST_BIN=$(tool_get_bin github.com/gogo/protobuf/protoc-gen-gofast)
GRPC_GATEWAY_BIN=$(tool_get_bin github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway)
#OPENAPIV2_BIN=$(tool_get_bin github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2)
#GOGOPROTO_ROOT="$(tool_pkg_dir github.com/gogo/protobuf/proto)/.."
GRPC_GATEWAY_ROOT="$(tool_pkg_dir github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway)/.."
GOOGLEAPI_ROOT=$(mktemp -d -t 'googleapi.XXXXX')

readonly googleapi_commit=0adf469dcd7822bf5bc058a7b0217f5558a75643

function cleanup_googleapi() {
  rm -rf "${GOOGLEAPI_ROOT}"
}

trap cleanup_googleapi EXIT

function download_googleapi() {
  run pushd "${GOOGLEAPI_ROOT}"
  run git init
  run git remote add upstream https://github.com/googleapis/googleapis.git
  run git fetch upstream "${googleapi_commit}"
  run git reset --hard FETCH_HEAD
  run popd
}

download_googleapi

echo
echo "Resolved binary and packages versions:"
echo "  - protoc-gen-gofast:       ${GOFAST_BIN}"
echo "  - protoc-gen-grpc-gateway: ${GRPC_GATEWAY_BIN}"
echo "  - openapiv2:               ${OPENAPIV2_BIN}"
echo "  - gogoproto-root:          ${GOGOPROTO_ROOT}"
echo "  - grpc-gateway-root:       ${GRPC_GATEWAY_ROOT}"
GOGOPROTO_PATH="${GOGOPROTO_ROOT}:${GOGOPROTO_ROOT}/protobuf"

# directories containing protos to be built
DIRS="./server/storage/wal/walpb ./api/etcdserverpb ./server/etcdserver/api/snap/snappb ./api/mvccpb ./server/lease/leasepb ./api/authpb ./server/etcdserver/api/v3lock/v3lockpb ./server/etcdserver/api/v3election/v3electionpb ./api/membershippb ./api/versionpb"

log_callout -e "\\nRunning gofast (gogo) proto generation..."

for dir in ${DIRS}; do
  run pushd "${dir}"
    run protoc --gofast_out=plugins=grpc:. -I=".:${GOGOPROTO_PATH}:${ETCD_ROOT_DIR}/..:${RAFT_ROOT}:${ETCD_ROOT_DIR}:${GOOGLEAPI_ROOT}" \
      --gofast_opt=paths=source_relative,Mraftpb/raft.proto=go.etcd.io/raft/v3/raftpb,Mgoogle/protobuf/descriptor.proto=github.com/gogo/protobuf/protoc-gen-gogo/descriptor \
      -I"${GRPC_GATEWAY_ROOT}" \
      --plugin="${GOFAST_BIN}" ./**/*.proto

    run gofmt -s -w ./**/*.pb.go
    run_go_tool "golang.org/x/tools/cmd/goimports" -w ./**/*.pb.go
  run popd
done

# log_callout -e "\\nRunning swagger & grpc_gateway proto generation..."

log_success -e "\\n./genproto SUCCESS"
