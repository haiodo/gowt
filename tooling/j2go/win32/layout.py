#!/usr/bin/env python3
"""Writes tooling/j2go/win32_layout.txt: the C size of each Win32 PI struct and offset/size of each of its
Java fields, measured with mingw-w64 (x86_64-w64-mingw32-gcc) and run under Wine (CrossOver bottle "gowt").
Usage: layout.py <swt repo> <out file> <wine exe> [bottle]; the Java classes come from os_structs.h/com_structs.h."""
import os, re, subprocess, sys, tempfile

swt, out, wine = sys.argv[1], sys.argv[2], sys.argv[3]
bottle = sys.argv[4] if len(sys.argv) > 4 else "gowt"
base = swt + "/bundles/org.eclipse.swt/Eclipse SWT PI/win32/"
PI = base + "org/eclipse/swt/internal/"
pairs = {}
for h in ("os_structs.h", "com_structs.h", "osversion_structs.h"):
    for l in open(base + "library/" + h):
        m = re.match(r"#define (\w+)_sizeof\(\) sizeof\((.*)\)", l)
        if m:
            pairs[m.group(1)] = m.group(2)
# sizeof natives of structs that have no Java class in the PI sources.
pairs.update({"ELEMDESC": "ELEMDESC", "TYPEDESC": "TYPEDESC", "LOGPEN": "LOGPEN", "PROPVARIANT": "PROPVARIANT", "SCRIPT_STRING_ANALYSIS": "SCRIPT_STRING_ANALYSIS"})
# Bitfield structs are hand-packed (bitfields_manual.go), FLICK_* is absent from the mingw headers.
skip = {"FLICK_DATA", "FLICK_POINT", "MENUBARINFO", "SCRIPT_ANALYSIS", "SCRIPT_CONTROL", "SCRIPT_LOGATTR",
        "SCRIPT_PROPERTIES", "SCRIPT_STATE"}
hdr = """#include <windows.h>
#include <WindowsX.h>
#include <commctrl.h>
#include <commdlg.h>
#include <oaidl.h>
#include <shlobj.h>
#include <ole2.h>
#include <olectl.h>
#include <objbase.h>
#include <shlwapi.h>
#include <shellapi.h>
#include <wininet.h>
#include <mshtmhst.h>
#include <oleacc.h>
#include <usp10.h>
#include <uxtheme.h>
#include <msctf.h>
#include <stddef.h>
#include <stdio.h>
int main(){
"""
c = hdr + 'printf("NOTIFYICONDATA_V2_SIZE %d\\n", (int)NOTIFYICONDATA_V2_SIZE);\n'
for cls, ctype in sorted(pairs.items()):
    if cls in skip:
        continue
    path = None
    for d in ("win32/", "ole/win32/", "win32/version/"):
        if os.path.exists(PI + d + cls + ".java"):
            path = PI + d + cls + ".java"
    if not path:
        c += 'printf("%s %%d\\n", (int)sizeof(%s));\n' % (cls, ctype)
        continue
    c += 'printf("%s %%d\\n", (int)sizeof(%s));\n' % (cls, ctype)
    src = re.sub(r"(?m)^\s*//.*$", "", open(path).read())
    for m in re.finditer(r"(?:/\*\*(.*?)\*/\s*)?public\s+(?!static)([A-Za-z_0-9]+)\s*(\[\s*\])?\s+([A-Za-z_0-9]+)\s*(\[\s*\])?\s*(=[^;]*)?;", src, re.S):
        doc, typ, _, name, _, _ = m.groups()
        if typ == "class":
            continue
        acc = re.search(r"accessor=([^\s,*]+)", doc or "")
        expr = acc.group(1) if acc else name
        c += 'printf("%s.%s %%d %%d\\n", (int)offsetof(%s, %s), (int)sizeof(((%s*)0)->%s));\n' % (cls, name, ctype, expr, ctype, expr)
c += "return 0;}\n"
d = tempfile.mkdtemp()
open(d + "/l.c", "w").write(c)
subprocess.check_call(["x86_64-w64-mingw32-gcc", "-w", "-DUNICODE", "-D_UNICODE", "-D_WIN32_WINNT=0x0A00", "-DWINVER=0x0A00", "-o", d + "/l.exe", d + "/l.c"])
res = subprocess.check_output([wine, "--bottle", bottle, d + "/l.exe"]).decode().replace("\r", "")
# Gdip/ole helper structs are C++ or hand-declared: sizes from gdiplusgpstubs.h/gdiplustypes.h.
extra = """MENUBARINFO 48
SCRIPT_ANALYSIS 4
SCRIPT_CONTROL 4
SCRIPT_LOGATTR 1
SCRIPT_PROPERTIES 8
SCRIPT_STATE 2
GdiplusStartupInput 24
GdiplusStartupInput.GdiplusVersion 0 4
GdiplusStartupInput.DebugEventCallback 8 8
GdiplusStartupInput.SuppressBackgroundThread 16 4
GdiplusStartupInput.SuppressExternalCodecs 20 4
BitmapData 32
BitmapData.Width 0 4
BitmapData.Height 4 4
BitmapData.Stride 8 4
BitmapData.PixelFormat 12 4
BitmapData.Scan0 16 8
BitmapData.Reserved 24 8
ColorPalette 12
ColorPalette.Flags 0 4
ColorPalette.Count 4 4
ColorPalette.Entries 8 4
PointF 8
PointF.X 0 4
PointF.Y 4 4
Rect 16
Rect.X 0 4
Rect.Y 4 4
Rect.Width 8 4
Rect.Height 12 4
RectF 16
RectF.X 0 4
RectF.Y 4 4
RectF.Width 8 4
RectF.Height 12 4
"""
open(out, "w").write("# class size / class.field offset size, see layout.py\n" + res + extra)
