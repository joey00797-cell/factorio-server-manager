#!/bin/bash
# Usage: tools/release.sh X.Y.Z [--prerelease] [--dry-run]
# Builds the archives, writes SHA256SUMS (and a minisign signature if
# FSM_MINISIGN_KEY points to a secret key), then tags and creates the release.
set -euo pipefail
REPO="joey00797-cell/factorio-server-manager"
VER="${1:?usage: tools/release.sh X.Y.Z [--prerelease] [--dry-run]}"
shift
TAG="v$VER"
PRE=""; DRY=0
for a in "$@"; do
    case "$a" in
        --prerelease) PRE="--prerelease";;
        --dry-run) DRY=1;;
        *) echo "Unknown option: $a"; exit 1;;
    esac
done
cd "$(dirname "$(readlink -f "$0")")/.."
have() { [ -n "$(command -v "$1")" ]; }
have gh || { echo "gh is required"; exit 1; }
have docker || { echo "docker is required"; exit 1; }

[ -z "$(git status --porcelain)" ] || { echo "Working tree is not clean"; exit 1; }
BR=$(git branch --show-current)
if [ -z "$PRE" ] && [ "$BR" != "master" ]; then
    echo "Stable releases are made from master (you are on $BR). Use --prerelease for other branches."; exit 1
fi
git fetch origin --tags
[ "$(git rev-parse HEAD)" = "$(git rev-parse "origin/$BR")" ] || { echo "$BR differs from origin/$BR; push or pull first"; exit 1; }
[ "$DRY" = "1" ] || [ -z "$(git tag -l "$TAG")" ] || { echo "Tag $TAG already exists locally"; exit 1; }
[ "$DRY" = "1" ] || [ -z "$(git ls-remote --tags origin "refs/tags/$TAG")" ] || { echo "Tag $TAG already exists on origin"; exit 1; }

DIST="/tmp/fsm-release-$TAG"
rm -rf "$DIST"; mkdir -p "$DIST/raw"
NOTES="$DIST/notes.md"
awk -v v="$VER" 'index($0, "## [" v "]") == 1 {f=1; next} /^## \[/ {f=0} f' CHANGELOG.md > "$NOTES"
[ -s "$NOTES" ] || { echo "No section [$VER] in CHANGELOG.md"; exit 1; }

echo "== Building"
docker build -f docker/Dockerfile-build --target build --build-arg "VERSION=$TAG" -t fsm-release-stage .
ID=$(docker create fsm-release-stage)
docker cp "$ID:/go/src/factorio-server-manager/build/." "$DIST/raw/"
docker rm "$ID"
for os in linux windows; do
    [ -f "$DIST/raw/factorio-server-manager-$os.zip" ] || { echo "Missing $os archive"; exit 1; }
    cp "$DIST/raw/factorio-server-manager-$os.zip" "$DIST/fsm-$os-$TAG.zip"
done
(cd "$DIST" && sha256sum fsm-*.zip > SHA256SUMS)
ASSETS=("$DIST"/fsm-*.zip "$DIST/SHA256SUMS")
if [ -n "${FSM_MINISIGN_KEY:-}" ] && have minisign; then
    minisign -Sm "$DIST/SHA256SUMS" -s "$FSM_MINISIGN_KEY"
    ASSETS+=("$DIST/SHA256SUMS.minisig")
fi

echo ""
echo "== Release $TAG ${PRE:+(pre-release)}"
cat "$NOTES"
echo ""
cat "$DIST/SHA256SUMS"
echo "Assets: ${ASSETS[*]}"
if [ "$DRY" = "1" ]; then echo "Dry run: nothing published. Files are in $DIST"; exit 0; fi
read -r -p "Create tag and release $TAG? [y/N] " a
[ "$a" = "y" ] || { echo "Cancelled."; exit 0; }
git tag "$TAG"
git push origin "$TAG"
gh release create "$TAG" --repo "$REPO" --title "$TAG" --notes-file "$NOTES" $PRE "${ASSETS[@]}"
echo "Released $TAG"
