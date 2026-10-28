#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

SDK=golang
MODULE=github.com/signalhouse-sms/sdk-go
TRIES=${PUBLISH_TRIES:-10}
fail() { echo "::error::$SDK: $1" >&2; exit 1; }
run() { if [ "${PUBLISH_DRY_RUN:-}" = 1 ]; then printf '+ (dry run) would run:'; printf ' %q' "$@"; echo; else "$@"; fi; }
export GOWORK=off

# The release is the tag; a dry run outside a tag falls back to the legacy pin.
tag_version() {
	local tags
	tags=$(git tag --points-at HEAD --list 'v*' 2>/dev/null || true)
	if [ "$(grep -c . <<<"$tags")" -eq 1 ]; then echo "${tags#v}"; return; fi
	[ -z "$tags" ] || fail "several v* tags on HEAD: $tags"
	[ "${PUBLISH_DRY_RUN:-}" = 1 ] || fail "HEAD has no v* tag"
	awk -v m="$MODULE" '$1 == "require" && $2 == m { sub(/^v/, "", $3); print $3 }' legacy/go.mod
	echo "(dry run) no v* tag on HEAD; using the legacy/go.mod pin" >&2
}

# Exit 0 = the proxy serves it, 1 = absent; anything else fails.
published() {
	local code
	code=$(curl -sS -o /dev/null -w '%{http_code}' --retry 3 "https://proxy.golang.org/$MODULE/@v/v$1.info") || fail "proxy check v$1"
	case $code in
		200) return 0 ;;
		404 | 410) return 1 ;;
		*) fail "proxy check v$1: HTTP $code" ;;
	esac
}

# Asks the proxy to fetch the tag, retrying while GitHub propagates it.
warm_proxy() {
	local scratch i
	scratch=$(mktemp -d)
	(cd "$scratch" && go mod init scratch >/dev/null 2>&1)
	for ((i = 1; i <= TRIES; i++)); do
		if (cd "$scratch" && GOPROXY=https://proxy.golang.org GOFLAGS=-mod=mod go list -m "$MODULE@v$V"); then rm -rf "$scratch"; return 0; fi
		[ "$i" -lt "$TRIES" ] && { echo "proxy has no v$V yet (try $i/$TRIES); retrying in 30s"; sleep 30; }
	done
	rm -rf "$scratch"
	return 1
}

V=$(tag_version)
[ -n "$V" ] || fail "read version"
if published "$V"; then echo "$MODULE@v$V exists, skipping"; exit 0; fi
echo "polling GOPROXY=https://proxy.golang.org go list -m $MODULE@v$V (up to $TRIES tries, 30s apart)"
run warm_proxy || fail "proxy did not serve $MODULE@v$V"
run curl -fsS --retry 5 "https://sum.golang.org/lookup/$MODULE@v$V" || fail "sum.golang.org lookup"
