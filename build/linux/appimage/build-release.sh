#!/usr/bin/env bash
# Builds Hermit-<version>-x86_64.AppImage. Run on Ubuntu 24.04 (see
# deps-ubuntu.sh and docker.sh); the AppImage bundles GTK 4 and WebKitGTK 6.
#
#   build/linux/appimage/build-release.sh 0.1.0 [output-dir]
set -euo pipefail

VERSION="${1:?usage: build-release.sh <version> [output-dir]}"
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUT="$(mkdir -p "${2:-$ROOT/bin}" && cd "${2:-$ROOT/bin}" && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
export PATH="/usr/local/go/bin:$PATH"
export APPIMAGE_EXTRACT_AND_RUN=1 # no FUSE in containers and CI

echo "==> Building Hermit $VERSION"
(cd "$ROOT/frontend" && npm ci --no-audit --no-fund && npm run build)
(cd "$ROOT" && go build -tags production -trimpath -buildvcs=false \
  -ldflags="-w -s -X hermit/internal/app.Version=$VERSION" -o "$WORK/hermit" .)

APPDIR="$WORK/Hermit.AppDir"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/apprun-hooks"
cp "$WORK/hermit" "$APPDIR/usr/bin/hermit"
# linuxdeploy accepts icons up to 512x512.
cp "$ROOT/build/linux/appimage/hermit.png" "$WORK/hermit.png"
cat > "$WORK/hermit.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Hermit
Comment=BepInEx mod manager for Linux
Exec=hermit
Icon=hermit
Categories=Game;Utility;
Terminal=false
StartupWMClass=org.wails.hermit
DESKTOP

# WebKit starts helper processes from a directory compiled into the library.
# Bundle them, and further down make that path relative to the AppImage.
WEBKIT_DIR="$(pkg-config --variable=libdir webkitgtk-6.0)/webkitgtk-6.0"
mkdir -p "$APPDIR$WEBKIT_DIR"
cp -r "$WEBKIT_DIR/." "$APPDIR$WEBKIT_DIR/"

echo "==> Deploying libraries"
cd "$WORK"
wget -q -O linuxdeploy "https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage"
chmod +x linuxdeploy
cp "$ROOT/build/linux/appimage/linuxdeploy-plugin-gtk.sh" .
DEPLOY_GTK_VERSION=4 NO_STRIP=1 ./linuxdeploy --appdir "$APPDIR" \
  --executable "$APPDIR/usr/bin/hermit" \
  --executable "$APPDIR$WEBKIT_DIR/WebKitWebProcess" \
  --executable "$APPDIR$WEBKIT_DIR/WebKitNetworkProcess" \
  --library "$APPDIR$WEBKIT_DIR/injected-bundle/libwebkitgtkinjectedbundle.so" \
  --desktop-file "$WORK/hermit.desktop" --icon-file "$WORK/hermit.png" \
  --plugin gtk

echo "==> Patching WebKit helper paths"
# linuxdeploy points the helpers at their own directory; their libraries are
# in usr/lib, two and three levels up.
patchelf --set-rpath '$ORIGIN/../..' "$APPDIR$WEBKIT_DIR/WebKitWebProcess" "$APPDIR$WEBKIT_DIR/WebKitNetworkProcess"
patchelf --set-rpath '$ORIGIN/../../..' "$APPDIR$WEBKIT_DIR/injected-bundle/libwebkitgtkinjectedbundle.so"

# "/usr/lib/..." becomes "././/lib/...": same length, resolved from $APPDIR/usr,
# which the last AppRun hook makes the working directory.
LIBWEBKIT="$(find "$APPDIR/usr/lib" -name 'libwebkitgtk-6.0.so*' -type f | head -n1)"
grep -q -a "$WEBKIT_DIR" "$LIBWEBKIT"
sed -i "s|$WEBKIT_DIR|././${WEBKIT_DIR#/usr}|g" "$LIBWEBKIT"
grep -q -a "././${WEBKIT_DIR#/usr}" "$LIBWEBKIT"

# Hooks run in name order: save the untouched environment first (the launch
# wrapper hands it to games), and finish in the AppDir with GTK's own backend
# choice restored (the GTK plugin forces X11). WebKit's bubblewrap sandbox
# cannot bind the helpers' relative paths inside the read-only image and
# aborts, so it is turned off; the webview only shows Hermit's own UI and
# Thunderstore package pages.
cat > "$APPDIR/apprun-hooks/00-hermit-environment.sh" <<'HOOK'
export HERMIT_ORIGINAL_ENV="$(env -0 | base64 -w0)"
if [ "${GDK_BACKEND+set}" = set ]; then export HERMIT_GDK_BACKEND="$GDK_BACKEND"; fi
HOOK
cat > "$APPDIR/apprun-hooks/zz-hermit-workdir.sh" <<'HOOK'
if [ "${HERMIT_GDK_BACKEND+set}" = set ]; then export GDK_BACKEND="$HERMIT_GDK_BACKEND"; else unset GDK_BACKEND; fi
unset HERMIT_GDK_BACKEND GTK_THEME
export WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS=1
cd "$APPDIR/usr"
HOOK

echo "==> Packaging"
OUTPUT="Hermit-$VERSION-x86_64.AppImage" ./linuxdeploy --appdir "$APPDIR" --output appimage
mv "Hermit-$VERSION-x86_64.AppImage" "$OUT/"
echo "==> $OUT/Hermit-$VERSION-x86_64.AppImage"
