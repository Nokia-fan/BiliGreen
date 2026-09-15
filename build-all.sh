#!/usr/bin/env sh
set -eu
mkdir -p build
for target in windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
  os=${target%/*}; arch=${target#*/}; ext=""; [ "$os" = windows ] && ext=".exe"
  echo "building $os/$arch"
  flags="-s -w"
  [ "$os" = windows ] && flags="-s -w -H=windowsgui"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags="$flags" -o "build/biligreen-${os}-${arch}${ext}" .
done
