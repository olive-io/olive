#!/usr/bin/env bash

# Examples:

# Edit go.mod files such that all olive modules are pointing on given version:
#
# % DRY_RUN=false TARGET_VERSION="v1.0.2" ./scripts/release_mod.sh update_versions

# Tag latest commit with current version number for all the modules and push upstream:
#
# % DRY_RUN=false REMOTE_REPO="origin" ./scripts/release_mod.sh push_mod_tags

set -e

source ./scripts/lib.sh

DRY_RUN=${DRY_RUN:-true}

# _cmd prints help message
function _cmd() {
  log_error "Command required: ${0} [cmd]"
  log_info "Available commands:"
  log_info "  - update_versions  - Updates all cross-module versions to \${TARGET_VERSION} in the local client."
  log_info "  - push_mod_tags    - Tags HEAD with all modules versions tags and pushes it to \${REMOTE_REPO}."
}

# update_module_version 
#   Updates versions of cross-references in all internal references in current module.
function update_module_version() {
  # remove go.work
  rm -fr go.work go.work.sum
  local v1version="${1}"
  local modules
  run go mod tidy
  modules=$(run go list -mod=readonly -f '{{if not .Main}}{{if not .Indirect}}{{.Path}}{{end}}{{end}}' -m all)

  deps=$(echo "${modules}" | grep -E "${ROOT_MODULE}/.*")
  for dep in ${deps}; do
    run go mod edit -require "${dep}@${v1version}"
  done

  run go mod tidy
}

function mod_tidy_fix {
  run rm ./go.sum
  run go mod tidy || return 2
}

# Updates all cross-module versions to ${TARGET_VERSION} in local client.
function update_versions_cmd() {
  assert_no_git_modifications || return 2

  if [ -z "${TARGET_VERSION}" ]; then
    log_error "TARGET_VERSION environment variable not set. Set it to e.g. v1.2.9-alpha.0"
    return 2
  fi

  local v2version="${TARGET_VERSION}"
  local v1version
  # converts e.g. v2.5.0-alpha.0 --> v1.205.0-alpha.0
  # shellcheck disable=SC2001
  v1version="$(echo "${TARGET_VERSION}" | sed 's|^v2.\([0-9]*\).|v1.30\1.|g')"

  log_info "DRY_RUN       : ${DRY_RUN}"
  log_info "TARGET_VERSION: ${TARGET_VERSION}"
  log_info ""
  # log_info "v2version: ${v2version}"
  # log_info "v1version: ${v1version}"

  run_for_modules update_module_version "${v2version}" "${v1version}"
  run_for_modules mod_tidy_fix || exit 2
}

function get_gpg_key {
  gitemail=$(git config --get user.email)
  keyid=$(run gpg --list-keys --with-colons "${gitemail}" | awk -F: '/^pub:/ { print $5 }')
  if [[ -z "${keyid}" ]]; then
    log_error "Failed to load gpg key. Is gpg set up correctly for olive releases?"
    return 2
  fi
  echo "$keyid"
}

function push_mod_tags_cmd {
  rm -fr go.work*
  assert_no_git_modifications || return 2

  if [ -z "${REMOTE_REPO}" ]; then
    log_error "REMOTE_REPO environment variable not set"
    return 2
  fi
  log_info "REMOTE_REPO:  ${REMOTE_REPO}"

  # Any module ccan be used for this
  local main_version
  main_version=$(go list -f '{{.Version}}' -m "${ROOT_MODULE}/api")
  local tags=()

  #keyid=$(get_gpg_key) || return 2

  for module in $(modules); do
    local version
    version=$(go list -f '{{.Version}}' -m "${module}")
    local path
    path=$(go list -f '{{.Path}}' -m "${module}")
    local subdir="${path//${ROOT_MODULE}\//}"
    local tag
    if [ -z "${version}" ]; then
      tag="${main_version}"
      version="${main_version}"
    else
      tag="${subdir///v[12]/}/${version}"
    fi

    log_info "Tags for: ${module} version:${version} tag:${tag}"
    # The sleep is ugly hack that guarantees that 'git describe' will
    # consider main-module's tag as the latest.
    run sleep 2
    #run git tag --local-user "${keyid}" --sign "${tag}" --message "${version}"
    run git tag "${tag}"
    tags=("${tags[@]}" "${tag}")
  done
  maybe_run git push -f "${REMOTE_REPO}" "${tags[@]}"
  go_work
}

# only release_mod when called directly, not sourced
if echo "$0" | grep -E "release_mod.sh$" >/dev/null; then
  "${1}_cmd"

  if "${DRY_RUN}"; then
    log_info
    log_warning "WARNING: It was a DRY_RUN. No files were modified."
  fi
fi
