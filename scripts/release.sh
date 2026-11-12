#!/bin/sh
set -eu
cd "$(dirname "$0")/.."

tag="${1:-}"
case "$tag" in
	v[0-9]*.[0-9]*.[0-9]*) ;;
	*) echo "usage: scripts/release.sh v0.1.0" >&2 && exit 1 ;;
esac
[ -z "$(git status --porcelain)" ] || { echo "commit your changes first" >&2 && exit 1; }
gh auth status >/dev/null 2>&1 || { echo "log in first: gh auth login" >&2 && exit 1; }

go test ./...
VERSION="$tag" ./scripts/build.sh

previous="$(git describe --tags --abbrev=0 2>/dev/null || true)"
changes="$(git log --format='- %s' ${previous:+"$previous"..}HEAD)"
notes="$(cat <<NOTES
## Install

\`\`\`
curl -fsSL https://github.com/elliotnorth/Scimitar/releases/latest/download/install.sh | sh
\`\`\`

Already installed? Run \`sudo scimitar update\`.

## Changes

$changes
NOTES
)"

git tag -a "$tag" -m "Scimitar $tag"
git push origin "$tag"
gh release create "$tag" dist/* --title "Scimitar $tag" --notes "$notes"
