#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

SDK=golang
LEGACY_REPO=bonzo-signalhouse/SignalHouseSDK-GO
MODULE=github.com/signalhouse-sms/sdk-go
TRIES=${PUSH_TRIES:-10}
fail() { echo "::error::$SDK: $1" >&2; exit 1; }
DRY=${PUSH_DRY_RUN:-}

[ -n "${VERSION:-}" ] || fail "VERSION unset"
V=${VERSION#v}

SUNSET=$(grep -oE '\*\*[0-9]{4}-[0-9]{2}-[0-9]{2}\*\*' LEGACY_NOTICE.md | head -1 | tr -d '*')
[ -n "$SUNSET" ] || fail "no **YYYY-MM-DD** sunset date in LEGACY_NOTICE.md"
if [[ "$(date -u +%F)" > "$SUNSET" ]]; then
	echo "::notice::$SDK: legacy line sunset on $SUNSET; not pushing v$V"
	exit 0
fi

export GOWORK=off
PIN=$(awk -v m="$MODULE" '$1 == "require" && $2 == m { print $3 }' legacy/go.mod)
if [ "$PIN" != "v$V" ]; then
	[ "$DRY" = 1 ] || fail "legacy/go.mod requires $MODULE $PIN, not v$V"
	echo "::warning::$SDK: (dry run) legacy/go.mod requires $PIN, not v$V; a real run fails here"
fi

T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
TREE=$T/tree
mkdir -p "$TREE"
cp -R legacy/. "$TREE/"
cp LEGACY_README.md "$TREE/README.md"

if [ "$DRY" = 1 ]; then
	echo "(dry run) would run go mod tidy against https://proxy.golang.org to write go.sum, then commit this tree on $LEGACY_REPO main and tag v$V:"
	(cd "$TREE" && find . -type f | sort | sed 's#^\./#  #')
	echo "  go.sum (generated)"
	echo "--- go.mod"; cat "$TREE/go.mod"
	echo "--- README.md (first 3 lines)"; head -3 "$TREE/README.md"
	exit 0
fi

# The real go.sum needs $MODULE@v$V on the proxy; retry while it propagates.
for ((i = 1; i <= TRIES; i++)); do
	if (cd "$TREE" && GOPROXY=https://proxy.golang.org go mod tidy); then break; fi
	[ "$i" -lt "$TRIES" ] || fail "go mod tidy: proxy never served $MODULE@v$V"
	echo "proxy has no $MODULE@v$V yet (try $i/$TRIES); retrying in 30s"
	sleep 30
done
[ -s "$TREE/go.sum" ] || fail "go mod tidy wrote no go.sum"
(cd "$TREE" && GOPROXY=https://proxy.golang.org go build ./... && go vet ./...) || fail "shim build against $MODULE@v$V"

[ -n "${LEGACY_TOKEN:-}" ] || fail "LEGACY_TOKEN unset"
# Auth by header so the token never lands in a URL or .git/config.
export GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=http.https://github.com/.extraheader
GIT_CONFIG_VALUE_0="AUTHORIZATION: basic $(printf 'x-access-token:%s' "$LEGACY_TOKEN" | base64 | tr -d '\n')"
export GIT_CONFIG_VALUE_0
export GIT_AUTHOR_NAME=${GIT_AUTHOR_NAME:-signalhouse-bot} GIT_AUTHOR_EMAIL=${GIT_AUTHOR_EMAIL:-signalhouse-bot[bot]@users.noreply.github.com}
export GIT_COMMITTER_NAME=${GIT_COMMITTER_NAME:-$GIT_AUTHOR_NAME} GIT_COMMITTER_EMAIL=${GIT_COMMITTER_EMAIL:-$GIT_AUTHOR_EMAIL}

# LEGACY_URL overrides the remote for local tests.
git clone --quiet --single-branch --branch main "${LEGACY_URL:-https://github.com/$LEGACY_REPO.git}" "$T/repo" || fail "clone $LEGACY_REPO"
cd "$T/repo"
git rm -rq --ignore-unmatch -- .
cp -R "$TREE/." .
git add -A
SHIM_TREE=$(git write-tree)

if git ls-remote --exit-code --tags origin "refs/tags/v$V" >/dev/null; then
	git fetch --quiet origin "refs/tags/v$V:refs/tags/v$V" || fail "fetch tag v$V"
	[ "$(git rev-parse "v$V^{tree}")" = "$SHIM_TREE" ] || fail "$LEGACY_REPO tag v$V exists on a different tree; not moving it"
	echo "$LEGACY_REPO v$V exists, skipping"
	exit 0
fi

if [ "$(git rev-parse 'HEAD^{tree}')" != "$SHIM_TREE" ]; then
	git commit --quiet -m "Release v$V: shim for $MODULE v$V" || fail "commit"
	git push --quiet origin HEAD:refs/heads/main || fail "push $LEGACY_REPO main (not a fast-forward? re-run)"
fi
git tag -a "v$V" -m "v$V" || fail "tag v$V"
git push --quiet origin "refs/tags/v$V" || fail "push tag v$V"
echo "$LEGACY_REPO v$V pushed"
