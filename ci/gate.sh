#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

SDK=golang

# >>> version-files (S3; must match versionFiles("golang") in ci/sdk-mirror/set-version.mjs)
VERSION_FILES=(
	legacy/go.mod
)
# <<< version-files

fail() { echo "::error::$SDK: $1" >&2; exit 1; }
check() { local name=$1; shift; "$@" || fail "$name"; }

T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
export GOWORK=off

# Prints "<file> <field> <value>" per version field.
read_versions() {
	local f
	for f in "${VERSION_FILES[@]}"; do
		case $f in
			legacy/go.mod) awk -v f="$f" '$1 == "require" && $2 == "github.com/signalhouse-sms/sdk-go" { print f, "require[github.com/signalhouse-sms/sdk-go]", $3; n++ } END { if (n != 1) print f, "require[github.com/signalhouse-sms/sdk-go] <missing>" }' "$f" ;;
			*) fail "version-files: no reader for $f" ;;
		esac
	done
}

check_versions() {
	local lines want
	lines=$(read_versions) || return 1
	if [ -n "${EXPECTED_VERSION:-}" ]; then want=v$EXPECTED_VERSION; else want=$(head -1 <<<"$lines" | awk '{print $NF}'); fi
	echo "$lines" | awk -v want="$want" '$NF != want { print "  " $0 " != " want; bad=1 } END { exit bad }'
}

check_legacy_readme() {
	cmp -s LEGACY_README.md <(cat LEGACY_NOTICE.md README.md)
}

# go mod tidy must be a no-op; compared by copy so it also works outside a git checkout.
check_tidy() {
	local rc=0
	cp go.mod "$T/go.mod.orig"
	[ -e go.sum ] && cp go.sum "$T/go.sum.orig"
	go mod tidy || rc=1
	cmp -s go.mod "$T/go.mod.orig" || { diff -u "$T/go.mod.orig" go.mod >&2 || true; rc=1; }
	if [ -e "$T/go.sum.orig" ]; then cmp -s go.sum "$T/go.sum.orig" || rc=1; elif [ -e go.sum ]; then echo "  go mod tidy created go.sum" >&2; rm -f go.sum; rc=1; fi
	cp "$T/go.mod.orig" go.mod
	[ -e "$T/go.sum.orig" ] && cp "$T/go.sum.orig" go.sum
	return $rc
}

check_gofmt() {
	local out
	out=$(gofmt -l .) || return 1
	[ -z "$out" ] || { echo "$out" >&2; return 1; }
}

# The committed shim must equal a fresh generation.
check_shim_generated() {
	(cd legacy && go run ./internal/gen -out "$T/shim.go") || return 1
	cmp -s "$T/shim.go" legacy/signalhouse/shim.go || { diff -u legacy/signalhouse/shim.go "$T/shim.go" >&2 || true; return 1; }
}

# Builds the shim against this checkout through a temp modfile; legacy/go.mod stays untouched.
check_shim_build() {
	local m=$T/legacy-mod
	mkdir -p "$m"
	{ cat legacy/go.mod; echo; echo "replace github.com/signalhouse-sms/sdk-go => ../"; } >"$m/go.mod"
	(cd legacy && go build -modfile="$m/go.mod" ./... && go vet -modfile="$m/go.mod" ./... && go test -modfile="$m/go.mod" ./...)
}

echo "== $SDK: versions"; check versions check_versions
echo "== $SDK: legacy readme"; check legacy-readme check_legacy_readme
echo "== $SDK: gofmt"; check gofmt check_gofmt
echo "== $SDK: go vet"; check "go vet" go vet ./...
echo "== $SDK: go build"; check "go build" go build ./...
echo "== $SDK: go mod tidy"; check "go mod tidy" check_tidy
echo "== $SDK: tests"; check tests go test ./...
echo "== $SDK: examples"; check examples go build -o /dev/null ./examples/...
echo "== $SDK: shim generated"; check "shim generated" check_shim_generated
echo "== $SDK: shim build"; check "shim build" check_shim_build
echo "gate passed: $SDK"
