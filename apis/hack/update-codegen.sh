#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
ls ${SCRIPT_ROOT}/vendor &> /dev/null && rm -rf ${SCRIPT_ROOT}/vendor
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../hack)}

source "${CODEGEN_PKG}/kube_codegen.sh"

THIS_PKG="github.com/olive-io/olive"

kube::codegen::gen_helpers \
    --boilerplate "${SCRIPT_ROOT}/../hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}"

if [[ -n "${API_KNOWN_VIOLATIONS_DIR:-}" ]]; then
    report_filename="${API_KNOWN_VIOLATIONS_DIR}/olive_violation_exceptions.list"
    if [[ "${UPDATE_API_KNOWN_VIOLATIONS:-}" == "true" ]]; then
        update_report="--update-report"
    fi
fi

kube::codegen::gen_openapi \
    --output-dir "${SCRIPT_ROOT}/../client-go/generated/openapi" \
    --output-pkg "${THIS_PKG}/client-go/generated/openapi" \
    --report-filename "${report_filename:-"/dev/null"}" \
    ${update_report:+"${update_report}"} \
    --boilerplate "${SCRIPT_ROOT}/../hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}/"

kube::codegen::gen_client \
    --with-watch \
    --with-applyconfig \
    --output-dir "${SCRIPT_ROOT}/../client-go/generated" \
    --output-pkg "${THIS_PKG}/client-go/generated" \
    --boilerplate "${SCRIPT_ROOT}/../hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}/"

go mod tidy && go mod vendor
kube::codegen::gen_protobuf \
    --output-dir "$(echo $GOPATH)/src" \
    --boilerplate "${SCRIPT_ROOT}/../hack/boilerplate.go.txt" \
        "${SCRIPT_ROOT}/"