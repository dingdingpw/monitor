#!/usr/bin/env sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
RELEASE="$ROOT/release"
EMBED_BINS="$ROOT/internal/server/agent_bins"
WEB="$ROOT/web"
WEB_DIST="$WEB/dist"
EMBED_WEB_DIST="$ROOT/internal/server/web/dist"
mkdir -p "$RELEASE"
mkdir -p "$EMBED_BINS"

(cd "$WEB" && npm run build)
rm -rf "$EMBED_WEB_DIST"
mkdir -p "$(dirname "$EMBED_WEB_DIST")"
cp -R "$WEB_DIST" "$EMBED_WEB_DIST"

build_one() {
  cmd="$1"
  name="$2"
  goos="$3"
  goarch="$4"
  goarm="${5:-}"
  suffix="$goos-$goarch"
  ext=""
  if [ -n "$goarm" ]; then suffix="$suffix""v$goarm"; fi
  if [ "$goos" = "windows" ]; then ext=".exe"; fi
  out="$RELEASE/$name-$suffix$ext"
  echo "building $out"
  if [ -n "$goarm" ]; then
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOARM="$goarm" go build -p 1 -trimpath -ldflags "-s -w" -o "$out" "$cmd"
  else
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -p 1 -trimpath -ldflags "-s -w" -o "$out" "$cmd"
  fi
}

for pair in "linux amd64" "linux arm64" "linux arm 7" "linux 386" "windows amd64" "windows arm64" "windows 386"; do
  set -- $pair
  build_one ./cmd/vps-agent vps-agent "$1" "$2" "${3:-}"
done

cp "$RELEASE"/vps-agent-* "$EMBED_BINS"/

for pair in "linux amd64" "linux arm64" "linux arm 7" "linux 386" "windows amd64" "windows arm64" "windows 386"; do
  set -- $pair
  build_one ./cmd/vps-server vps-server "$1" "$2" "${3:-}"
done

cp "$ROOT/scripts/install-server-interactive-linux.sh" "$RELEASE/install-server-linux.sh"
cp "$ROOT/scripts/install-agent-interactive-linux.sh" "$RELEASE/install-agent-linux.sh"
cp "$ROOT/scripts/install-agent-interactive-windows.ps1" "$RELEASE/install-agent-windows.ps1"
cp "$ROOT/scripts/uninstall-server-linux.sh" "$RELEASE/uninstall-server-linux.sh"
cp "$ROOT/scripts/uninstall-agent-linux.sh" "$RELEASE/uninstall-agent-linux.sh"
cp "$ROOT/scripts/uninstall-agent-windows.ps1" "$RELEASE/uninstall-agent-windows.ps1"

echo "release files written to $RELEASE"
