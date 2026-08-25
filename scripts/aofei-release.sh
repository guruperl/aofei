#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
DEFAULT_AOFEI_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd -P)

usage() {
  cat <<'EOF'
Usage:
  scripts/aofei-release.sh build --output ABSOLUTE_PATH \
    [--pzdesign-root ABSOLUTE_PATH] [--genelet-root ABSOLUTE_PATH]
  scripts/aofei-release.sh verify RELEASE_DIRECTORY

Build creates an immutable W8M HTTP-backend release bundle. All three source
repositories must be clean and exactly equal to their configured upstream
branches. Existing output paths are never overwritten.
EOF
}

fail() {
  printf 'aofei-release: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command is unavailable: $1"
}

canonical_directory() {
  local path=$1
  [[ $path == /* ]] || fail "directory must be absolute: $path"
  [[ -d $path ]] || fail "directory does not exist: $path"
  (cd -- "$path" && pwd -P)
}

assert_release_source() {
  local name=$1
  local root=$2
  local status upstream head upstream_head remote_name remote_branch remote_head

  git -C "$root" rev-parse --is-inside-work-tree >/dev/null 2>&1 ||
    fail "$name is not a Git worktree: $root"
  status=$(git -C "$root" status --porcelain=v1 --untracked-files=all)
  [[ -z $status ]] || fail "$name worktree is not clean: $root"
  upstream=$(git -C "$root" rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null) ||
    fail "$name branch has no upstream: $root"
  head=$(git -C "$root" rev-parse HEAD)
  upstream_head=$(git -C "$root" rev-parse '@{upstream}')
  [[ $head == "$upstream_head" ]] ||
    fail "$name HEAD $head is not exactly published at $upstream ($upstream_head)"
  remote_name=${upstream%%/*}
  remote_branch=${upstream#*/}
  remote_head=$(git -C "$root" ls-remote --exit-code "$remote_name" "refs/heads/$remote_branch" | awk 'NR == 1 {print $1}') ||
    fail "$name upstream cannot be read from $remote_name"
  [[ $head == "$remote_head" ]] ||
    fail "$name HEAD $head does not match remote $upstream ($remote_head)"
}

source_manifest_value() {
  local root=$1
  local field=$2
  local branch remote_name

  branch=$(git -C "$root" branch --show-current)
  case "$field" in
    commit) git -C "$root" rev-parse HEAD ;;
    branch) printf '%s\n' "$branch" ;;
    upstream) git -C "$root" rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' ;;
    remote)
      remote_name=$(git -C "$root" config --get "branch.$branch.remote")
      git -C "$root" remote get-url "$remote_name"
      ;;
    *) fail "unsupported source field: $field" ;;
  esac
}

run_source_tests() {
  local aofei_root=$1
  local pzdesign_root=$2
  local genelet_root=$3

  (cd -- "$aofei_root" && GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./...)
  (cd -- "$pzdesign_root" && GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./...)
  (cd -- "$genelet_root" && GOWORK=off GOTOOLCHAIN=go1.23.5 go test ./...)
}

copy_pzdesign_assets() {
  local pzdesign_root=$1
  local target=$2
  local component relative

  if find "$pzdesign_root/tmpls" "$pzdesign_root/www" -type l -print -quit | grep -q .; then
    fail "pzdesign templates/static assets contain a symlink"
  fi

  mkdir -p -- "$target/summer" "$target/tmpls" "$target/www"
  while IFS= read -r -d '' component; do
    relative=${component#"$pzdesign_root/"}
    install -d -m 0755 -- "$target/$(dirname -- "$relative")"
    install -m 0644 -- "$component" "$target/$relative"
  done < <(find "$pzdesign_root/summer" -mindepth 2 -maxdepth 2 -type f -name component.json -print0 | LC_ALL=C sort -z)
  rsync -a --delete -- "$pzdesign_root/tmpls/" "$target/tmpls/"
  rsync -a --delete -- "$pzdesign_root/www/" "$target/www/"
}

normalize_release_modes() {
  local stage=$1

  find "$stage/assets" -type d -exec chmod 0755 {} +
  find "$stage/assets" -type f -exec chmod 0644 {} +
  find "$stage/bin" -type d -exec chmod 0755 {} +
  find "$stage/bin" -type f -exec chmod 0755 {} +
}

write_manifest() {
  local stage=$1
  local aofei_root=$2
  local pzdesign_root=$3
  local genelet_root=$4
  local aofei_commit pzdesign_commit genelet_commit release_id component_count artifact_count

  aofei_commit=$(source_manifest_value "$aofei_root" commit)
  pzdesign_commit=$(source_manifest_value "$pzdesign_root" commit)
  genelet_commit=$(source_manifest_value "$genelet_root" commit)
  release_id="aofei-${aofei_commit:0:12}_pzdesign-${pzdesign_commit:0:12}_genelet-${genelet_commit:0:12}"
  component_count=$(find "$stage/assets/pzdesign/summer" -mindepth 2 -maxdepth 2 -type f -name component.json | wc -l)
  artifact_count=$(find "$stage/bin" "$stage/assets" -type f | wc -l)

  jq -n \
    --arg release_id "$release_id" \
    --arg built_at "$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
    --arg go_version "$(go version)" \
    --arg aofei_commit "$aofei_commit" \
    --arg aofei_branch "$(source_manifest_value "$aofei_root" branch)" \
    --arg aofei_upstream "$(source_manifest_value "$aofei_root" upstream)" \
    --arg aofei_remote "$(source_manifest_value "$aofei_root" remote)" \
    --arg pzdesign_commit "$pzdesign_commit" \
    --arg pzdesign_branch "$(source_manifest_value "$pzdesign_root" branch)" \
    --arg pzdesign_upstream "$(source_manifest_value "$pzdesign_root" upstream)" \
    --arg pzdesign_remote "$(source_manifest_value "$pzdesign_root" remote)" \
    --arg genelet_commit "$genelet_commit" \
    --arg genelet_branch "$(source_manifest_value "$genelet_root" branch)" \
    --arg genelet_upstream "$(source_manifest_value "$genelet_root" upstream)" \
    --arg genelet_remote "$(source_manifest_value "$genelet_root" remote)" \
    --argjson component_count "$component_count" \
    --argjson artifact_count "$artifact_count" \
    '{
      schema_version: 1,
      kind: "w8m-http-backend",
      release_id: $release_id,
      built_at: $built_at,
      go_version: $go_version,
      sources: {
        aofei: {commit: $aofei_commit, branch: $aofei_branch, upstream: $aofei_upstream, remote: $aofei_remote},
        pzdesign: {commit: $pzdesign_commit, branch: $pzdesign_branch, upstream: $pzdesign_upstream, remote: $pzdesign_remote},
        genelet: {commit: $genelet_commit, branch: $genelet_branch, upstream: $genelet_upstream, remote: $genelet_remote}
      },
      contracts: {
        database: {tables: 96, views: 0, routines: 6, triggers: 65},
        accounting_version: "usd-cpm-impression-v3"
      },
      assets: {component_count: $component_count},
      artifact_count: $artifact_count
    }' >"$stage/manifest.json"
  printf '%s\n' "$release_id" >"$stage/RELEASE_ID"

  (
    cd -- "$stage"
    {
      find bin assets -type f -print0 | LC_ALL=C sort -z | xargs -0 sha256sum
      sha256sum manifest.json RELEASE_ID
    } >checksums.sha256
  )
}

verify_release() {
  local release=$1
  local expected_aofei expected_pzdesign actual_component_count expected_component_count
  local actual_artifact_count expected_artifact_count checksum_count

  release=$(canonical_directory "$release")
  [[ -f $release/manifest.json ]] || fail "manifest.json is missing"
  [[ -f $release/checksums.sha256 ]] || fail "checksums.sha256 is missing"
  [[ -f $release/RELEASE_ID ]] || fail "RELEASE_ID is missing"
  [[ -x $release/bin/unify ]] || fail "bin/unify is missing or not executable"
  [[ -x $release/bin/config-preflight ]] || fail "bin/config-preflight is missing or not executable"
  [[ -d $release/assets/pzdesign/tmpls ]] || fail "templates are missing"
  [[ -d $release/assets/pzdesign/www ]] || fail "static assets are missing"
  [[ -z $(find "$release" -type l -print -quit) ]] || fail "release contains a symlink"
  [[ -z $(find "$release" \( -perm -0002 -o -perm -0020 \) -print -quit) ]] ||
    fail "release contains a group/world-writable path"
  awk '{path=$2; if (path ~ /^\// || path ~ /(^|\/)\.\.(\/|$)/) exit 1}' "$release/checksums.sha256" ||
    fail "checksum manifest contains an unsafe path"
  (cd -- "$release" && sha256sum --check --strict checksums.sha256 >/dev/null) ||
    fail "release checksum verification failed"
  jq -e '
    .schema_version == 1 and
    .kind == "w8m-http-backend" and
    (.release_id | test("^aofei-[0-9a-f]{12}_pzdesign-[0-9a-f]{12}_genelet-[0-9a-f]{12}$")) and
    .contracts.database == {tables:96, views:0, routines:6, triggers:65} and
    .contracts.accounting_version == "usd-cpm-impression-v3"
  ' "$release/manifest.json" >/dev/null || fail "release manifest contract is invalid"
  jq -e '.sources | all(.[]; (.commit | test("^[0-9a-f]{40}$")))' "$release/manifest.json" >/dev/null ||
    fail "release source provenance is invalid"
  [[ $(cat "$release/RELEASE_ID") == "$(jq -r '.release_id' "$release/manifest.json")" ]] ||
    fail "RELEASE_ID does not match the manifest"
  expected_aofei=$(jq -r '.sources.aofei.commit' "$release/manifest.json")
  expected_pzdesign=$(jq -r '.sources.pzdesign.commit' "$release/manifest.json")
  go version -m "$release/bin/config-preflight" | grep -F "vcs.revision=$expected_aofei" >/dev/null ||
    fail "config-preflight build provenance does not match the manifest"
  go version -m "$release/bin/unify" | grep -F "vcs.revision=$expected_pzdesign" >/dev/null ||
    fail "unify build provenance does not match the manifest"
  actual_component_count=$(find "$release/assets/pzdesign/summer" -mindepth 2 -maxdepth 2 -type f -name component.json | wc -l)
  expected_component_count=$(jq -r '.assets.component_count' "$release/manifest.json")
  [[ $actual_component_count == "$expected_component_count" ]] || fail "component inventory does not match the manifest"
  actual_artifact_count=$(find "$release/bin" "$release/assets" -type f | wc -l)
  expected_artifact_count=$(jq -r '.artifact_count' "$release/manifest.json")
  [[ $actual_artifact_count == "$expected_artifact_count" ]] || fail "artifact inventory does not match the manifest"
  checksum_count=$(wc -l <"$release/checksums.sha256")
  [[ $checksum_count -eq $((actual_artifact_count + 2)) ]] || fail "checksum inventory is incomplete"
  [[ -f $release/assets/pzdesign/summer/publishercredential/component.json ]] ||
    fail "publishercredential component metadata is missing"
  printf 'release_verification=passed release_id=%s\n' "$(jq -r '.release_id' "$release/manifest.json")"
}

build_release() {
  local output=""
  local aofei_root=$DEFAULT_AOFEI_ROOT
  local pzdesign_root="$DEFAULT_AOFEI_ROOT/../pzdesign"
  local genelet_root="$DEFAULT_AOFEI_ROOT/../genelet"
  local output_parent stage=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --output) [[ $# -ge 2 ]] || fail "--output requires a value"; output=$2; shift 2 ;;
      --pzdesign-root) [[ $# -ge 2 ]] || fail "--pzdesign-root requires a value"; pzdesign_root=$2; shift 2 ;;
      --genelet-root) [[ $# -ge 2 ]] || fail "--genelet-root requires a value"; genelet_root=$2; shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) fail "unknown build argument: $1" ;;
    esac
  done

  [[ $output == /* ]] || fail "--output must be an absolute path"
  [[ ! -e $output ]] || fail "output already exists: $output"
  output_parent=$(dirname -- "$output")
  [[ $output_parent != / ]] || fail "refusing a release directly below /"
  output_parent=$(canonical_directory "$output_parent")
  aofei_root=$(canonical_directory "$aofei_root")
  pzdesign_root=$(canonical_directory "$pzdesign_root")
  genelet_root=$(canonical_directory "$genelet_root")

  for command in git go jq rsync sha256sum find install mktemp; do
    require_command "$command"
  done
  assert_release_source aofei "$aofei_root"
  assert_release_source pzdesign "$pzdesign_root"
  assert_release_source genelet "$genelet_root"
  run_source_tests "$aofei_root" "$pzdesign_root" "$genelet_root"

  stage=$(mktemp -d "$output_parent/.aofei-release.XXXXXX")
  trap 'if [[ -n ${stage:-} && -d $stage ]]; then rm -rf -- "$stage"; fi' EXIT
  install -d -m 0755 -- "$stage/bin" "$stage/assets/pzdesign"
  (cd -- "$pzdesign_root" && GOWORK=off GOTOOLCHAIN=go1.23.5 go build -trimpath -o "$stage/bin/unify" ./cmd/unify)
  (cd -- "$aofei_root" && GOWORK=off GOTOOLCHAIN=go1.23.5 go build -trimpath -o "$stage/bin/config-preflight" ./cmd/config-preflight)
  copy_pzdesign_assets "$pzdesign_root" "$stage/assets/pzdesign"
  normalize_release_modes "$stage"
  write_manifest "$stage" "$aofei_root" "$pzdesign_root" "$genelet_root"
  verify_release "$stage"
  mv -- "$stage" "$output"
  stage=""
  trap - EXIT
  printf 'release_build=passed path=%s release_id=%s\n' "$output" "$(cat "$output/RELEASE_ID")"
}

main() {
  local command=${1:-}
  case "$command" in
    build) shift; build_release "$@" ;;
    verify) [[ $# -eq 2 ]] || fail "verify requires one release directory"; verify_release "$2" ;;
    -h|--help|help) usage ;;
    *) usage >&2; exit 2 ;;
  esac
}

main "$@"
