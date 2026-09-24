#!/usr/bin/env bash
# Runs j2go over the stage-1 (swt/graphics value types) and stage-2/3 (internal/cocoa PI
# bindings, all of them) file lists. --out is the repo root: each file lands under swt/ or
# internal/cocoa/ per its own Java package (see Main.goPackageDir).
set -euo pipefail
cd "$(dirname "$0")/.."

SWT_REPO="${SWT_REPO:-/Users/haiodo/Develop/repos/eclipse.platform.swt}"
COCOA_DIR="$SWT_REPO/bundles/org.eclipse.swt/Eclipse SWT PI/cocoa/org/eclipse/swt/internal/cocoa"
EVENTS_DIR="$SWT_REPO/bundles/org.eclipse.swt/Eclipse SWT/common/org/eclipse/swt/events"

mvn -q -f tooling/j2go/pom.xml package

# Round 3: SWT.java/SWTException/SWTError, Listener/Event/EventTable/TypedListener and the whole
# events/ package now join stage-1's graphics types in one invocation - TypeModel is
# per-invocation (see README "Round 2"), and SWT.java references Point/Rectangle-adjacent
# constants while TypedListener.java references every events/* type directly.
mapfile -t EVENTS_FILES < <(find "$EVENTS_DIR" -maxdepth 1 -name '*.java' -exec basename {} \; | sort | sed 's#^#org/eclipse/swt/events/#')

# Checkpoint b: every Eclipse SWT PI/cocoa class in one j2go run (all of them share bindings -
# TypeModel is per-invocation, see README "Round 2"), plus C.java (PI/common, OS.java's own
# base for memmove/malloc/PTR_sizeof). Selector.java is deliberately excluded from the file
# list - not parsed as a translation target at all, only resolved via sourcepath for the
# structural sel_x.value -> OSSel_registerName(...) rewrite (see README "Selector enum elision").
# -printf is a GNU find extension, not on macOS's BSD find - list + basename instead.
mapfile -t COCOA_FILES < <(find "$COCOA_DIR" -maxdepth 1 -name '*.java' ! -name 'Selector.java' -exec basename {} \; | sort | sed 's#^#org/eclipse/swt/internal/cocoa/#')

# Round 4: Widget/Control/Scrollable join stage-1's swt-package file set. They call straight into
# internal/cocoa (OS.objc_msgSend, NSView, ...) - the whole cocoa file set is fed after "--" as
# reference-only (parsed and modeled for cross-package name resolution, not re-emitted here; the
# second invocation below is still what actually (re)generates internal/cocoa/*.go).
#
# Round 5: Layout/Item (common widgets) and FillLayout/FillData/RowLayout/RowData (layout) join
# next - Composite.layout is a Layout field, so Layout must resolve before it. Then the cocoa
# widget-hierarchy chain above Scrollable: Composite/Canvas/Decorations/Shell, plus Button (a
# leaf Control subclass, independent of that chain).
#
# Round 6: Display and what its construction/event loop needs for real - Device (Display's
# superclass), Resource/Font/Color (Device.init builds the system colors/font), DeviceData,
# Synchronizer/RunnableLock (asyncExec queue polled every readAndDispatch/sleep).
#
# Round 7: GridLayout/GridData/FormAttachment/FormLayout/FormData (common layout - pure
# computation, no cocoa calls) join FillLayout/RowLayout; Label/Menu/MenuItem/Text (cocoa widgets)
# join Button. Menu/MenuItem move here from the manual-stub list, so Display/Shell/Decorations/
# Control/Widget (which all construct or call into them) must be regenerated in this same
# invocation - see README "Round 7 widgets".
# Round 7 gfx: the paint path - GC/GCData and the graphics resources a GC draws with
# (Pattern/Transform/Path/Region/Image/Cursor/TextLayout) plus their common value types.
java -jar tooling/j2go/target/j2go.jar --swt "$SWT_REPO" --out . \
	org/eclipse/swt/graphics/Point.java \
	org/eclipse/swt/graphics/Rectangle.java \
	org/eclipse/swt/graphics/RGB.java \
	org/eclipse/swt/graphics/RGBA.java \
	org/eclipse/swt/SWT.java \
	org/eclipse/swt/SWTException.java \
	org/eclipse/swt/SWTError.java \
	org/eclipse/swt/widgets/Listener.java \
	org/eclipse/swt/widgets/Event.java \
	org/eclipse/swt/widgets/EventTable.java \
	org/eclipse/swt/widgets/TypedListener.java \
	"${EVENTS_FILES[@]}" \
	org/eclipse/swt/widgets/Widget.java \
	org/eclipse/swt/widgets/Control.java \
	org/eclipse/swt/widgets/Scrollable.java \
	org/eclipse/swt/widgets/Layout.java \
	org/eclipse/swt/widgets/Item.java \
	org/eclipse/swt/layout/FillLayout.java \
	org/eclipse/swt/layout/FillData.java \
	org/eclipse/swt/layout/RowLayout.java \
	org/eclipse/swt/layout/RowData.java \
	org/eclipse/swt/layout/GridLayout.java \
	org/eclipse/swt/layout/GridData.java \
	org/eclipse/swt/layout/FormAttachment.java \
	org/eclipse/swt/layout/FormLayout.java \
	org/eclipse/swt/layout/FormData.java \
	org/eclipse/swt/widgets/Composite.java \
	org/eclipse/swt/widgets/Canvas.java \
	org/eclipse/swt/widgets/Decorations.java \
	org/eclipse/swt/widgets/Shell.java \
	org/eclipse/swt/widgets/Button.java \
	org/eclipse/swt/widgets/Label.java \
	org/eclipse/swt/widgets/Menu.java \
	org/eclipse/swt/widgets/MenuItem.java \
	org/eclipse/swt/widgets/Text.java \
	org/eclipse/swt/widgets/Group.java \
	org/eclipse/swt/widgets/Sash.java \
	org/eclipse/swt/custom/StackLayout.java \
	org/eclipse/swt/custom/SashFormLayout.java \
	org/eclipse/swt/custom/SashFormData.java \
	org/eclipse/swt/custom/SashForm.java \
	org/eclipse/swt/graphics/Resource.java \
	org/eclipse/swt/graphics/Device.java \
	org/eclipse/swt/graphics/DeviceData.java \
	org/eclipse/swt/graphics/Font.java \
	org/eclipse/swt/graphics/FontData.java \
	org/eclipse/swt/graphics/Color.java \
	org/eclipse/swt/widgets/Synchronizer.java \
	org/eclipse/swt/widgets/RunnableLock.java \
	org/eclipse/swt/widgets/Monitor.java \
	org/eclipse/swt/widgets/TouchSource.java \
	org/eclipse/swt/widgets/Display.java \
	org/eclipse/swt/graphics/Drawable.java \
	org/eclipse/swt/graphics/GC.java \
	org/eclipse/swt/graphics/GCData.java \
	org/eclipse/swt/graphics/FontMetrics.java \
	org/eclipse/swt/graphics/LineAttributes.java \
	org/eclipse/swt/graphics/Pattern.java \
	org/eclipse/swt/graphics/Transform.java \
	org/eclipse/swt/graphics/Path.java \
	org/eclipse/swt/graphics/PathData.java \
	org/eclipse/swt/graphics/Region.java \
	org/eclipse/swt/graphics/Image.java \
	org/eclipse/swt/graphics/ImageData.java \
	org/eclipse/swt/graphics/PaletteData.java \
	org/eclipse/swt/graphics/ImageDataProvider.java \
	org/eclipse/swt/graphics/ImageFileNameProvider.java \
	org/eclipse/swt/graphics/ImageDataAtSizeProvider.java \
	org/eclipse/swt/graphics/ImageGcDrawer.java \
	org/eclipse/swt/graphics/Cursor.java \
	org/eclipse/swt/graphics/TextLayout.java \
	org/eclipse/swt/graphics/TextStyle.java \
	org/eclipse/swt/graphics/GlyphMetrics.java \
	-- \
	org/eclipse/swt/internal/C.java \
	"${COCOA_FILES[@]}"

java -jar tooling/j2go/target/j2go.jar --swt "$SWT_REPO" --out . \
	org/eclipse/swt/internal/C.java \
	"${COCOA_FILES[@]}"

gofmt -w swt/*.go internal/cocoa/*.go
