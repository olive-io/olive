#!/usr/bin/env bash

# This scripts build the etcd binaries
# To build the tools, run `build_tools.sh`

set -o errexit
set -o nounset
set -euo pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
ls ${SCRIPT_ROOT}/vendor &> /dev/null && rm -rf ${SCRIPT_ROOT}/vendor

source ./scripts/lib.sh
source ./scripts/build_lib.sh

# only build when called directly, not sourced
if echo "$0" | grep -E "build(.sh)?$" >/dev/null; then
  run_build olive_build
fi
