#!/usr/bin/env bash

set -euo pipefail

source ./scripts/lib.sh

VER=${1:-}
REPOSITORY="${REPOSITORY:-https://github.com/olive-io/olive.git}"

if [ -z "$VER" ]; then
  echo "Usage: ${0} VERSION" >> /dev/stderr
  exit 255
fi

function setup_env {
  local ver=${1}
  local proj=${2}

  if [ ! -d "${proj}" ]; then
    run git clone "${REPOSITORY}"
  fi

  pushd "${proj}" >/dev/null
    run git fetch --all
    run git checkout "${ver}"
  popd >/dev/null
}


function package {
  local target=${1}
  local srcdir="${2}/bin"

  local ccdir="${srcdir}/${GOOS}_${GOARCH}"
  if [ -d "${ccdir}" ]; then
    srcdir="${ccdir}"
  fi
  local ext=""
  if [ "${GOOS}" == "windows" ]; then
    ext=".exe"
  fi
  for bin in olive-meta olive-runner olive-gateway; do
    cp "${srcdir}/${bin}" "${target}/${bin}${ext}"
  done

  cp olive/README.md "${target}"/README.md
  cp meta/README.md "${target}"/README-meta.md
  cp runner/README.md "${target}"/README-runner.md
  cp gateway/README.md "${target}"/README-gateway.md

  cp -R olive/Documentation "${target}"/Documentation
}

function main {
  local proj="olive"

  mkdir -p release
  cd release
  setup_env "${VER}" "${proj}"

  local tarcmd=tar
  if [[ $(go env GOOS) == "darwin" ]]; then
      echo "Please use linux machine for release builds."
    exit 1
  fi

  for os in darwin windows linux; do
    export GOOS=${os}
    TARGET_ARCHS=("amd64")

    if [ ${GOOS} == "linux" ]; then
      TARGET_ARCHS+=("arm64")
      TARGET_ARCHS+=("ppc64le")
      TARGET_ARCHS+=("s390x")
    fi

    if [ ${GOOS} == "darwin" ]; then
      TARGET_ARCHS+=("arm64")
    fi

    for TARGET_ARCH in "${TARGET_ARCHS[@]}"; do
      export GOARCH=${TARGET_ARCH}

      pushd olive >/dev/null
      GO_LDFLAGS="-s -w" ./scripts/build.sh
      popd >/dev/null

      TARGET="olive-${VER}-${GOOS}-${GOARCH}"
      mkdir "${TARGET}"
      package "${TARGET}" "${proj}"

      if [ ${GOOS} == "linux" ]; then
        ${tarcmd} cfz "${TARGET}.tar.gz" "${TARGET}"
        echo "Wrote release/${TARGET}.tar.gz"
      else
        zip -qr "${TARGET}.zip" "${TARGET}"
        echo "Wrote release/${TARGET}.zip"
      fi
    done
  done
}

main
