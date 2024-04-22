#!/usr/bin/env bash

# This scripts build the etcd binaries
# To build the tools, run `build_tools.sh`

set -euo pipefail

source ./scripts/lib.sh
source ./scripts/build_lib.sh

# only build when called directly, not sourced
if echo "$0" | grep -E "build(.sh)?$" >/dev/null; then
  run_build olive_build
fi
