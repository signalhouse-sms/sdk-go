#!/usr/bin/env bash
# Release reconciler (spec S8). Tags HEAD and creates its GitHub release; never deletes or moves a tag.
set -euo pipefail

fail() { echo "::error::release: $*" >&2; exit 1; }

V=$(git log -1 --format='%(trailers:key=Release-Version,valueonly)' HEAD | tr -d '[:space:]')
[ -n "$V" ] || fail "HEAD has no Release-Version trailer"
[[ "$V" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "Release-Version is not x.y.z: $V"
TAG="v$V"
HEAD_SHA=$(git rev-parse HEAD)

if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
  [ "$(git rev-parse "$TAG^{}")" = "$HEAD_SHA" ] || fail "tag $TAG exists and does not point at HEAD"
else
  git tag -a "$TAG" -m "$TAG"
  if ! git push origin "refs/tags/$TAG"; then
    git fetch --no-tags origin "refs/tags/$TAG"
    [ "$(git rev-parse 'FETCH_HEAD^{}')" = "$HEAD_SHA" ] || fail "tag $TAG on origin does not point at HEAD"
  fi
fi

if VIEW=$(gh release view "$TAG" --json tagName,isPrerelease --jq '.tagName + " " + (.isPrerelease | tostring)' 2>&1); then
  [ "$VIEW" = "$TAG false" ] || fail "release $TAG is wrong: $VIEW (want tag $TAG, not a pre-release)"
  echo "release $TAG exists"
  exit 0
fi
case "$VIEW" in
  *"not found"*) ;;
  *) fail "gh release view $TAG failed: $VIEW" ;;
esac

NOTES=$(mktemp)
trap 'rm -f "$NOTES"' EXIT
# The release notes are the sync commit's body, without its trailer paragraph.
BODY=$(git log -1 --format=%b HEAD | sed '/^Monorepo-Commit: /,$d')
[ -n "$BODY" ] || BODY=$TAG
printf '%s\n' "$BODY" >"$NOTES"
gh release create "$TAG" --title "$TAG" --notes-file "$NOTES" --verify-tag
