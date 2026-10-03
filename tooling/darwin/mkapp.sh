#!/usr/bin/env bash
# Wraps a release binary into a macOS .app so Finder launches it as an app, not in Terminal.
# Usage: mkapp.sh <binary> <out-dir> [display name]
# CLASSIC_LOOK=1 keeps the pre-macOS 26 look (UIDesignRequiresCompatibility); by default AppKit gives Liquid Glass.
set -euo pipefail
bin=$1; out=$2; name=${3:-$(basename "$bin")}
exe=$(basename "$bin")
compat=""
[ "${CLASSIC_LOOK:-}" = 1 ] && compat="<key>UIDesignRequiresCompatibility</key><true/>"
app="$out/$name.app"
rm -rf "$app"
mkdir -p "$app/Contents/MacOS"
cp "$bin" "$app/Contents/MacOS/$exe"
cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key><string>$exe</string>
	<key>CFBundleIdentifier</key><string>com.github.haiodo.gowt.$exe</string>
	<key>CFBundleName</key><string>$name</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>0.1</string>
	<key>CFBundleVersion</key><string>1</string>
	<key>LSMinimumSystemVersion</key><string>13.0</string>
	<key>NSHighResolutionCapable</key><true/>
	<key>NSPrincipalClass</key><string>NSApplication</string>
	$compat
</dict>
</plist>
PLIST
# Ad-hoc signature: arm64 refuses unsigned code; notarization needs a real identity (plan task 23).
codesign --force --sign - "$app" >/dev/null 2>&1
echo "$app"
