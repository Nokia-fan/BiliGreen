#!/usr/bin/env sh
set -eu

mkdir -p release
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

for arch in amd64 arm64; do
  mkdir -p "$stage/BiliGreen-Windows-$arch"
  cp "build/biligreen-windows-$arch.exe" "$stage/BiliGreen-Windows-$arch/BiliGreen.exe"
  cp README.md "$stage/BiliGreen-Windows-$arch/使用说明.md"
  cp README.en.md "$stage/BiliGreen-Windows-$arch/README.en.md"
  cp LICENSE "$stage/BiliGreen-Windows-$arch/LICENSE"
  (cd "$stage" && zip -qr "$OLDPWD/release/BiliGreen-Windows-$arch.zip" "BiliGreen-Windows-$arch")

  mkdir -p "$stage/BiliGreen-Linux-$arch"
  cp "build/biligreen-linux-$arch" "$stage/BiliGreen-Linux-$arch/BiliGreen"
  chmod +x "$stage/BiliGreen-Linux-$arch/BiliGreen"
  cp README.md "$stage/BiliGreen-Linux-$arch/使用说明.md"
  cp README.en.md "$stage/BiliGreen-Linux-$arch/README.en.md"
  cp LICENSE "$stage/BiliGreen-Linux-$arch/LICENSE"
  tar -C "$stage" -czf "release/BiliGreen-Linux-$arch.tar.gz" "BiliGreen-Linux-$arch"
done
