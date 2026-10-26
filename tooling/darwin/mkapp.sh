#!/usr/bin/env bash
# Wraps a release binary into a macOS .app so Finder launches it as an app, not in Terminal.
# Usage: mkapp.sh <binary> <out-dir> [display name]
# CLASSIC_LOOK=1 keeps the pre-macOS 26 look (UIDesignRequiresCompatibility); by default AppKit gives Liquid Glass.
# Bundle: BUNDLE_ID (default com.github.haiodo.gowt.<executable>), VERSION (CFBundleShortVersionString, default 0.1),
# BUILD (CFBundleVersion, default 1), ICON (the app icon: an .icns file, or a square PNG of 1024 or more that is
# converted with sips and iconutil; written as Resources/AppIcon.icns with CFBundleIconFile).
# File association (docs/packaging.md): DOC_EXTS="tnb tenniarb" (extensions without dots) adds CFBundleDocumentTypes;
# DOC_NAME is the type's name (default "<name> Document"), DOC_ICON a .icns file for the document icon, DOC_UTI a
# reverse-DNS type id (then the type is also exported with UTExportedTypeDeclarations); DOC_UTI_CONFORMS lists the
# types it conforms to, separated by spaces (default public.data; public.json public.text for a text format).
# DOC_ICON may also be a PNG.
set -euo pipefail
bin=$1; out=$2; name=${3:-$(basename "$bin")}
exe=$(basename "$bin")
bundle=${BUNDLE_ID:-com.github.haiodo.gowt.$exe}
version=${VERSION:-0.1}
build=${BUILD:-1}
compat=""
[ "${CLASSIC_LOOK:-}" = 1 ] && compat="<key>UIDesignRequiresCompatibility</key><true/>"
app="$out/$name.app"
# icns FILE: prints the path of an .icns file for an .icns or PNG input (PNG goes through an iconset).
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
icns() {
	case "$1" in
	*.icns) echo "$1" ;;
	*.png)
		local set="$tmp/$(basename "$1" .png).iconset"; mkdir -p "$set"
		for n in 16 32 128 256 512; do
			sips -z $n $n "$1" --out "$set/icon_${n}x${n}.png" >/dev/null
			sips -z $((n * 2)) $((n * 2)) "$1" --out "$set/icon_${n}x${n}@2x.png" >/dev/null
		done
		iconutil -c icns "$set" -o "$tmp/$(basename "$1" .png).icns"
		echo "$tmp/$(basename "$1" .png).icns" ;;
	*) echo "mkapp: icon $1 is neither .icns nor .png" >&2; exit 1 ;;
	esac
}
doc=""
if [ -n "${DOC_EXTS:-}" ]; then
	dname=${DOC_NAME:-"$name Document"}
	exts=""; for e in $DOC_EXTS; do exts="$exts<string>$e</string>"; done
	dicon=""
	if [ -n "${DOC_ICON:-}" ]; then
		docicns=$(icns "$DOC_ICON")
		dicon="<key>CFBundleTypeIconFile</key><string>$(basename "$docicns")</string>"
	fi
	conforms=""; for c in ${DOC_UTI_CONFORMS:-public.data}; do conforms="$conforms<string>$c</string>"; done
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
		<key>UTTypeConformsTo</key><array>$conforms</array>
		<key>UTTypeTagSpecification</key><dict><key>public.filename-extension</key><array>$exts</array></dict>
	</dict></array>"
	fi
fi
rm -rf "$app"
mkdir -p "$app/Contents/MacOS"
cp "$bin" "$app/Contents/MacOS/$exe"
mkdir -p "$app/Contents/Resources"
[ -n "${DOC_ICON:-}" ] && cp "$docicns" "$app/Contents/Resources/"
appicon=""
if [ -n "${ICON:-}" ]; then
	cp "$(icns "$ICON")" "$app/Contents/Resources/AppIcon.icns"
	appicon="<key>CFBundleIconFile</key><string>AppIcon</string>"
fi
cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key><string>$exe</string>
	<key>CFBundleIdentifier</key><string>$bundle</string>
	<key>CFBundleName</key><string>$name</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>$version</string>
	<key>CFBundleVersion</key><string>$build</string>
	<key>LSMinimumSystemVersion</key><string>13.0</string>
	<key>NSHighResolutionCapable</key><true/>
	<key>NSPrincipalClass</key><string>NSApplication</string>
	$appicon
	$compat
	$doc
</dict>
</plist>
PLIST
plutil -lint "$app/Contents/Info.plist" >/dev/null
# Ad-hoc signature: arm64 refuses unsigned code; notarization needs a real identity (plan task 23).
codesign --force --sign - "$app" >/dev/null 2>&1
echo "$app"
