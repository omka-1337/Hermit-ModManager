#!/usr/bin/env bash
# Installs what build-release.sh needs on Ubuntu 24.04, the oldest release with
# WebKitGTK 6. Building there keeps the AppImage runnable on older distros.
set -euo pipefail

GO_VERSION="${GO_VERSION:-1.27.1}"
NODE_VERSION="${NODE_VERSION:-24.8.0}"
SUDO=""
[ "$(id -u)" -ne 0 ] && SUDO="sudo"

export DEBIAN_FRONTEND=noninteractive
$SUDO apt-get update
$SUDO apt-get install -y --no-install-recommends \
  ca-certificates curl wget file xz-utils git build-essential pkg-config \
  libgtk-4-dev libwebkitgtk-6.0-dev librsvg2-common desktop-file-utils

if ! command -v go >/dev/null || ! go version | grep -q "go${GO_VERSION}"; then
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | $SUDO tar -C /usr/local -xz
fi
if ! command -v node >/dev/null; then
  curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" \
    | $SUDO tar -C /usr/local --strip-components=1 -xJ
fi
