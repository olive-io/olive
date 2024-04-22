#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 VERSION" >&2
  exit 1
fi

VERSION=${1}
if [ -z "$VERSION" ]; then
  echo "Usage: ${0} VERSION" >&2
  exit 1
fi

ARCH=$(go env GOARCH)
VERSION="${VERSION}-${ARCH}"
DOCKERFILE="Dockerfile"

if [ -z "${BINARYDIR:-}" ]; then
  RELEASE="olive-${1}"-$(go env GOOS)-${ARCH}
  BINARYDIR="${RELEASE}"
  TARFILE="${RELEASE}.tar.gz"
  TARURL="https://github.com/olive-io/olive/releases/download/${1}/${TARFILE}"
  if ! curl -f -L -o "${TARFILE}" "${TARURL}" ; then
    echo "Failed to download ${TARURL}."
    exit 1
  fi
  tar -zvxf "${TARFILE}"
fi

BINARYDIR=${BINARYDIR:-.}
BUILDDIR=${BUILDDIR:-.}

IMAGEDIR=${BUILDDIR}/image-docker

mkdir -p "${IMAGEDIR}"/var/olive
mkdir -p "${IMAGEDIR}"/var/lib/olive
cp "${BINARYDIR}"/etcd "${BINARYDIR}"/olivectl "${BINARYDIR}"/oliveutl "${IMAGEDIR}"

cat ./"${DOCKERFILE}" > "${IMAGEDIR}"/Dockerfile

docker build -t "${TAG}:${VERSION}" "${IMAGEDIR}"
