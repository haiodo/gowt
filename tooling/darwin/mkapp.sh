#!/usr/bin/env bash
# Wraps a release binary into a macOS .app so Finder launches it as an app, not in Terminal.
# Usage: mkapp.sh <binary> <out-dir> [display name]
# CLASSIC_LOOK=1 keeps the pre-macOS 26 look (UIDesignRequiresCompatibility); by default AppKit gives Liquid Glass.
# File association (docs/packaging.md): DOC_EXTS="tnb tenniarb" (extensions without dots) adds CFBundleDocumentTypes;
# DOC_NAME is the type's name (default "<name> Document"), DOC_ICON a .icns file for the document icon, DOC_UTI a
# reverse-DNS type id (then the type is also exported with UTExportedTypeDeclarations).
set -euo pipefail
bin=$1; out=$2; name=${3:-$(basename "$bin")}
exe=$(basename "$bin")
compat=""
[ "${CLASSIC_LOOK:-}" = 1 ] && compat="<key>UIDesignRequiresCompatibility</key><true/>"
app="$out/$name.app"
doc=""
if [ -n "${DOC_EXTS:-}" ]; then
	dname=${DOC_NAME:-"$name Document"}
	exts=""; for e in $DOC_EXTS; do exts="$exts<string>$e</string>"; done
	dicon=""; [ -n "${DOC_ICON:-}" ] && dicon="<key>CFBundleTypeIconFile</key><string>$(basename "$DOC_ICON")</string>"
	duti=""; [ -n "${DOC_UTI:-}" ] && duti="<key>LSItemContentTypes</key><array><string>$DOC_UTI</string></array>"
	doc="<key>CFBundleDocumentTypes</key><array><dict>
		<key>CFBundleTypeName</key><string>$dname</string>
		<key>CFBundleTypeExtensions</key><array>$exts</array>
		<key>CFBundleTypeRole</key><string>Editor</string>
		<key>LSHandlerRank</key><string>Owner</string>
		$dicon
		$duti
	</dict></array>"
	if [ -n "${DOC_UTI:-}" ]; then
		doc="$doc<key>UTExportedTypeDeclarations</key><array><dict>
		<key>UTTypeIdentifier</key><string>$DOC_UTI</string>
		<key>UTTypeDescription</key><string>$dname</string>
		<key>UTTypeConformsTo</key><array><string>public.data</string></array>
		<key>UTTypeTagSpecification</key><dict><key>public.filename-extension</key><array>$exts</array></dict>
	</dict></array>"
	fi
fi
rm -rf "$app"
mkdir -p "$app/Contents/MacOS"
cp "$bin" "$app/Contents/MacOS/$exe"
if [ -n "${DOC_ICON:-}" ]; then
	mkdir -p "$app/Contents/Resources"
	cp "$DOC_ICON" "$app/Contents/Resources/"
fi
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
	$doc
</dict>
</plist>
PLIST
plutil -lint "$app/Contents/Info.plist" >/dev/null
# Ad-hoc signature: arm64 refuses unsigned code; notarization needs a real identity (plan task 23).
codesign --force --sign - "$app" >/dev/null 2>&1
echo "$app"
