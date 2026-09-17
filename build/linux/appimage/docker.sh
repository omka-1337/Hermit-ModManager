#!/usr/bin/env bash
# Builds the release AppImage in an Ubuntu 24.04 container from the committed
# sources, so local builds match CI.
#
#   build/linux/appimage/docker.sh 0.1.0
set -euo pipefail

VERSION="${1:?usage: docker.sh <version>}"
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
mkdir -p "$ROOT/bin"

git -C "$ROOT" archive --format=tar HEAD | docker run --rm -i \
  -v "$ROOT/bin:/out" -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
  ubuntu:24.04 bash -euc "
    mkdir /src && tar -x -C /src
    /src/build/linux/appimage/deps-ubuntu.sh
    /src/build/linux/appimage/build-release.sh '$VERSION' /out
    chown \"\$HOST_UID:\$HOST_GID\" /out/Hermit-*.AppImage
  "
