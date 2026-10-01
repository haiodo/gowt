//go:build windows

// The GDI+ natives of org.eclipse.swt.internal.gdip.Gdip: gdip.cpp wraps the C++ classes, here the flat API of
// gdiplus.dll is called instead. Handles are the flat API's own pointers, except FontFamily: SWT creates an empty
// FontFamily and fills it later (Font.getFamily), so its handle is a cell holding the GpFontFamily*.
package win32

import (
	"math"
	"syscall"
	"unsafe"
)

var gdipProcs = map[string]*Proc{}

func gdipAddr(name string) uintptr {
	p := gdipProcs[name]
	if p == nil {
		p = newProc(name)
		gdipProcs[name] = p
	}
	return p.addr()
}

//go:uintptrescapes
func gp(name string, args ...uintptr) int32 {
	r, _, _ := syscall.SyscallN(gdipAddr(name), args...)
	return int32(r)
}

// gpOut calls a flat function whose last argument is a pointer receiving a handle; 0 when it fails.
//
//go:uintptrescapes
func gpOut(name string, args ...uintptr) int64 {
	out := new(uintptr)
	args = append(args, uintptr(unsafe.Pointer(out)))
	r, _, _ := syscall.SyscallN(gdipAddr(name), args...)
	if r != 0 {
		return 0
	}
	return int64(*out)
}

// gpInt calls a flat getter that writes a 32-bit value through its last argument.
//
//go:uintptrescapes
func gpInt(name string, args ...uintptr) int32 {
	out := new(int32)
	args = append(args, uintptr(unsafe.Pointer(out)))
	syscall.SyscallN(gdipAddr(name), args...)
	return *out
}

func f32(v float32) uintptr { return uintptr(math.Float32bits(v)) }

func b2u(b bool) uintptr { return boolToUintptr(b) }

func gdipFree(h int64) { kernel32.NewProc("LocalFree").Call(uintptr(h)) }

func gdipAlloc(n uintptr) int64 {
	h, _, _ := kernel32.NewProc("LocalAlloc").Call(0x40, n)
	return int64(h)
}

func cell(h int64) uintptr { return *(*uintptr)(unsafe.Add(unsafe.Pointer(nil), uintptr(h))) }

func GdipGdiplusStartup(token []int64, input *GdiplusStartupInput, output int64) int32 {
	var b [4]uint64
	input.toC(unsafe.Pointer(&b))
	return gp("GdiplusStartup", uintptr(unsafe.Pointer(&token[0])), uintptr(unsafe.Pointer(&b)), uintptr(output))
}

func GdipGdiplusShutdown(token int64) { syscall.SyscallN(gdipAddr("GdiplusShutdown"), uintptr(token)) }

func GdipBitmap_new(hbm int64, hpal int64) int64 {
	return gpOut("GdipCreateBitmapFromHBITMAP", uintptr(hbm), uintptr(hpal))
}

func GdipBitmap_newHicon(hicon int64) int64 {
	return gpOut("GdipCreateBitmapFromHICON", uintptr(hicon))
}

func GdipBitmap_newWidthHeightStrideFormatScan0(width int32, height int32, stride int32, format int32, scan0 int64) int64 {
	return gpOut("GdipCreateBitmapFromScan0", uintptr(width), uintptr(height), uintptr(stride), uintptr(format), uintptr(scan0))
}

func GdipBitmap_newFilenameUseIcm(filename []uint16, useIcm bool) int64 {
	if useIcm {
		return gpOut("GdipCreateBitmapFromFileICM", uintptr(unsafe.Pointer(&filename[0])))
	}
	return gpOut("GdipCreateBitmapFromFile", uintptr(unsafe.Pointer(&filename[0])))
}

func GdipBitmap_delete(bitmap int64) { gp("GdipDisposeImage", uintptr(bitmap)) }

func GdipBitmap_GetHBITMAP(bitmap int64, colorBackground int32, hbmReturn []int64) int32 {
	return gp("GdipCreateHBITMAPFromBitmap", uintptr(bitmap), uintptr(unsafe.Pointer(&hbmReturn[0])), uintptr(uint32(colorBackground)))
}

func GdipBitmap_GetHICON(bitmap int64, hicon []int64) int32 {
	return gp("GdipCreateHICONFromBitmap", uintptr(bitmap), uintptr(unsafe.Pointer(&hicon[0])))
}

func GdipBitmapData_new() int64 { return gdipAlloc(32) }

func GdipBitmapData_delete(bitmapData int64) { gdipFree(bitmapData) }

func GdipBitmap_LockBits(bitmap int64, rect int64, flags int32, pixelFormat int32, lockedBitmapData int64) int32 {
	return gp("GdipBitmapLockBits", uintptr(bitmap), uintptr(rect), uintptr(flags), uintptr(pixelFormat), uintptr(lockedBitmapData))
}

func GdipBitmap_UnlockBits(bitmap int64, lockedBitmapData int64) int32 {
	return gp("GdipBitmapUnlockBits", uintptr(bitmap), uintptr(lockedBitmapData))
}

func GdipBrush_Clone(brush int64) int64 { return gpOut("GdipCloneBrush", uintptr(brush)) }

func GdipBrush_GetType(brush int64) int32 { return gpInt("GdipGetBrushType", uintptr(brush)) }

func GdipPrivateFontCollection_new() int64 { return gpOut("GdipNewPrivateFontCollection") }

func GdipPrivateFontCollection_delete(collection int64) {
	c := uintptr(collection)
	gp("GdipDeletePrivateFontCollection", uintptr(unsafe.Pointer(&c)))
}

func GdipPrivateFontCollection_AddFontFile(collection int64, filename []uint16) int32 {
	return gp("GdipPrivateAddFontFile", uintptr(collection), uintptr(unsafe.Pointer(&filename[0])))
}

func GdipFont_new(hdc int64, hfont int64) int64 {
	var lf [92]byte
	r, _, _ := syscall.SyscallN(newProc("GetObject").addr(), uintptr(hfont), uintptr(len(lf)), uintptr(unsafe.Pointer(&lf)))
	if r == 0 {
		return 0
	}
	return gpOut("GdipCreateFontFromLogfontW", uintptr(hdc), uintptr(unsafe.Pointer(&lf)))
}

func GdipFont_newFamilyEmSizeStyleUnit(family int64, emSize float32, style int32, unit int32) int64 {
	return gpOut("GdipCreateFont", cell(family), f32(emSize), uintptr(style), uintptr(unit))
}

func GdipFont_newFamilyNameEmSizeStyleUnitFontCollection(familyName []uint16, emSize float32, style int32, unit int32, fontCollection int64) int64 {
	fam := gpOut("GdipCreateFontFamilyFromName", uintptr(unsafe.Pointer(&familyName[0])), uintptr(fontCollection))
	if fam == 0 {
		return 0
	}
	defer gp("GdipDeleteFontFamily", uintptr(fam))
	return gpOut("GdipCreateFont", uintptr(fam), f32(emSize), uintptr(style), uintptr(unit))
}

func GdipFont_delete(font int64) { gp("GdipDeleteFont", uintptr(font)) }

func GdipFont_GetFamily(font int64, family int64) int32 {
	fam := gpOut("GdipGetFamily", uintptr(font))
	*(*uintptr)(unsafe.Add(unsafe.Pointer(nil), uintptr(family))) = uintptr(fam)
	if fam == 0 {
		return 1
	}
	return 0
}

func GdipFont_GetSize(font int64) float32 {
	var v float32
	gp("GdipGetFontSize", uintptr(font), uintptr(unsafe.Pointer(&v)))
	return v
}

func GdipFont_GetStyle(font int64) int32 { return gpInt("GdipGetFontStyle", uintptr(font)) }

func GdipFont_GetLogFontW(font int64, g int64, logfontW int64) int32 {
	return gp("GdipGetLogFontW", uintptr(font), uintptr(g), uintptr(logfontW))
}

func GdipFont_IsAvailable(font int64) bool { return font != 0 }

func GdipFontFamily_new() int64 { return gdipAlloc(8) }

func GdipFontFamily_newNameFontCollection(name []uint16, fontCollection int64) int64 {
	fam := gpOut("GdipCreateFontFamilyFromName", uintptr(unsafe.Pointer(&name[0])), uintptr(fontCollection))
	c := gdipAlloc(8)
	*(*uintptr)(unsafe.Add(unsafe.Pointer(nil), uintptr(c))) = uintptr(fam)
	return c
}

func GdipFontFamily_delete(family int64) {
	if f := cell(family); f != 0 {
		gp("GdipDeleteFontFamily", f)
	}
	gdipFree(family)
}

func GdipFontFamily_GetFamilyName(family int64, name []uint16, language uint16) int32 {
	return gp("GdipGetFamilyName", cell(family), uintptr(unsafe.Pointer(&name[0])), uintptr(language))
}

func GdipFontFamily_IsAvailable(family int64) bool { return cell(family) != 0 }

func GdipGraphics_new(hdc int64) int64 { return gpOut("GdipCreateFromHDC", uintptr(hdc)) }

func GdipGraphics_delete(graphics int64) { gp("GdipDeleteGraphics", uintptr(graphics)) }

func GdipGraphics_DrawArc(graphics int64, pen int64, x int32, y int32, width int32, height int32, startAngle float32, sweepAngle float32) int32 {
	return gp("GdipDrawArcI", uintptr(graphics), uintptr(pen), uintptr(x), uintptr(y), uintptr(width), uintptr(height), f32(startAngle), f32(sweepAngle))
}

func GdipGraphics_DrawDriverString(graphics int64, text int64, length int32, font int64, brush int64, positions *PointF, flags int32, matrix int64) int32 {
	var b [2]float32
	b[0], b[1] = positions.X, positions.Y
	return gp("GdipDrawDriverString", uintptr(graphics), uintptr(text), uintptr(length), uintptr(font), uintptr(brush), uintptr(unsafe.Pointer(&b)), uintptr(flags), uintptr(matrix))
}

func GdipGraphics_DrawDriverStringGraphicsTextLengthFontBrushPositionsFlagsMatrix(graphics int64, text int64, length int32, font int64, brush int64, positions []float32, flags int32, matrix int64) int32 {
	return gp("GdipDrawDriverString", uintptr(graphics), uintptr(text), uintptr(length), uintptr(font), uintptr(brush), uintptr(unsafe.Pointer(&positions[0])), uintptr(flags), uintptr(matrix))
}

func GdipGraphics_DrawEllipse(graphics int64, pen int64, x int32, y int32, width int32, height int32) int32 {
	return gp("GdipDrawEllipseI", uintptr(graphics), uintptr(pen), uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func GdipGraphics_DrawImage(graphics int64, image int64, x int32, y int32) int32 {
	return gp("GdipDrawImageI", uintptr(graphics), uintptr(image), uintptr(x), uintptr(y))
}

func GdipGraphics_DrawImageGraphicsImageDestRectSrcxSrcySrcwidthSrcheightSrcUnitImageAttributesCallbackCallbackData(graphics int64, image int64, destRect *Rect, srcx int32, srcy int32, srcwidth int32, srcheight int32, srcUnit int32, imageAttributes int64, callback int64, callbackData int64) int32 {
	return gp("GdipDrawImageRectRectI", uintptr(graphics), uintptr(image), uintptr(destRect.X), uintptr(destRect.Y), uintptr(destRect.Width), uintptr(destRect.Height),
		uintptr(srcx), uintptr(srcy), uintptr(srcwidth), uintptr(srcheight), uintptr(srcUnit), uintptr(imageAttributes), uintptr(callback), uintptr(callbackData))
}

func GdipGraphics_DrawLine(graphics int64, pen int64, x1 int32, y1 int32, x2 int32, y2 int32) int32 {
	return gp("GdipDrawLineI", uintptr(graphics), uintptr(pen), uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2))
}

func GdipGraphics_DrawLines(graphics int64, pen int64, points []int32, count int32) int32 {
	return gp("GdipDrawLinesI", uintptr(graphics), uintptr(pen), uintptr(unsafe.Pointer(&points[0])), uintptr(count))
}

func GdipGraphics_DrawPath(graphics int64, pen int64, path int64) int32 {
	return gp("GdipDrawPath", uintptr(graphics), uintptr(pen), uintptr(path))
}

func GdipGraphics_DrawPolygon(graphics int64, pen int64, points []int32, count int32) int32 {
	return gp("GdipDrawPolygonI", uintptr(graphics), uintptr(pen), uintptr(unsafe.Pointer(&points[0])), uintptr(count))
}

func GdipGraphics_DrawRectangle(graphics int64, pen int64, x int32, y int32, width int32, height int32) int32 {
	return gp("GdipDrawRectangleI", uintptr(graphics), uintptr(pen), uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

// layoutAt is the RectF GDI+ takes where the C++ API took a point: the origin with no size.
func layoutAt(o *PointF) [4]float32 { return [4]float32{o.X, o.Y, 0, 0} }

func GdipGraphics_DrawString(graphics int64, string_ []uint16, length int32, font int64, origin *PointF, brush int64) int32 {
	r := layoutAt(origin)
	return gp("GdipDrawString", uintptr(graphics), uintptr(unsafe.Pointer(&string_[0])), uintptr(length), uintptr(font), uintptr(unsafe.Pointer(&r)), 0, uintptr(brush))
}

func GdipGraphics_DrawStringGraphicsStringLengthFontOriginFormatBrush(graphics int64, string_ []uint16, length int32, font int64, origin *PointF, format int64, brush int64) int32 {
	r := layoutAt(origin)
	return gp("GdipDrawString", uintptr(graphics), uintptr(unsafe.Pointer(&string_[0])), uintptr(length), uintptr(font), uintptr(unsafe.Pointer(&r)), uintptr(format), uintptr(brush))
}

func GdipGraphics_FillEllipse(graphics int64, brush int64, x int32, y int32, width int32, height int32) int32 {
	return gp("GdipFillEllipseI", uintptr(graphics), uintptr(brush), uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func GdipGraphics_FillPath(graphics int64, brush int64, path int64) int32 {
	return gp("GdipFillPath", uintptr(graphics), uintptr(brush), uintptr(path))
}

func GdipGraphics_Flush(graphics int64, intention int32) {
	gp("GdipFlush", uintptr(graphics), uintptr(intention))
}

func GdipGraphics_FillPie(graphics int64, brush int64, x int32, y int32, width int32, height int32, startAngle float32, sweepAngle float32) int32 {
	return gp("GdipFillPieI", uintptr(graphics), uintptr(brush), uintptr(x), uintptr(y), uintptr(width), uintptr(height), f32(startAngle), f32(sweepAngle))
}

func GdipGraphics_FillPolygon(graphics int64, brush int64, points []int32, count int32, fillMode int32) int32 {
	return gp("GdipFillPolygonI", uintptr(graphics), uintptr(brush), uintptr(unsafe.Pointer(&points[0])), uintptr(count), uintptr(fillMode))
}

func GdipGraphics_FillRectangle(graphics int64, brush int64, x int32, y int32, width int32, height int32) int32 {
	return gp("GdipFillRectangleI", uintptr(graphics), uintptr(brush), uintptr(x), uintptr(y), uintptr(width), uintptr(height))
}

func GdipGraphics_GetClipBounds(graphics int64, rect *RectF) int32 {
	var b [4]float32
	r := gp("GdipGetClipBounds", uintptr(graphics), uintptr(unsafe.Pointer(&b)))
	rect.fromC(unsafe.Pointer(&b))
	return r
}

func GdipGraphics_GetClipBoundsGraphicsRect(graphics int64, rect *Rect) int32 {
	var b [4]int32
	r := gp("GdipGetClipBoundsI", uintptr(graphics), uintptr(unsafe.Pointer(&b)))
	rect.fromC(unsafe.Pointer(&b))
	return r
}

func GdipGraphics_GetClip(graphics int64, region int64) int32 {
	return gp("GdipGetClip", uintptr(graphics), uintptr(region))
}

func GdipGraphics_GetHDC(graphics int64) int64 { return gpOut("GdipGetDC", uintptr(graphics)) }

func GdipGraphics_ReleaseHDC(graphics int64, hdc int64) {
	gp("GdipReleaseDC", uintptr(graphics), uintptr(hdc))
}

func GdipGraphics_GetInterpolationMode(graphics int64) int32 {
	return gpInt("GdipGetInterpolationMode", uintptr(graphics))
}

func GdipGraphics_GetSmoothingMode(graphics int64) int32 {
	return gpInt("GdipGetSmoothingMode", uintptr(graphics))
}

func GdipGraphics_GetTextRenderingHint(graphics int64) int32 {
	return gpInt("GdipGetTextRenderingHint", uintptr(graphics))
}

func GdipGraphics_GetTransform(graphics int64, matrix int64) int32 {
	return gp("GdipGetWorldTransform", uintptr(graphics), uintptr(matrix))
}

func GdipGraphics_GetVisibleClipBounds(graphics int64, rect *Rect) int32 {
	var b [4]int32
	r := gp("GdipGetVisibleClipBoundsI", uintptr(graphics), uintptr(unsafe.Pointer(&b)))
	rect.fromC(unsafe.Pointer(&b))
	return r
}

func GdipGraphics_MeasureDriverString(graphics int64, text int64, length int32, font int64, positions []float32, flags int32, matrix int64, boundingBox *RectF) int32 {
	var b [4]float32
	r := gp("GdipMeasureDriverString", uintptr(graphics), uintptr(text), uintptr(length), uintptr(font), uintptr(unsafe.Pointer(&positions[0])), uintptr(flags), uintptr(matrix), uintptr(unsafe.Pointer(&b)))
	boundingBox.fromC(unsafe.Pointer(&b))
	return r
}

func measureString(graphics int64, string_ []uint16, length int32, font int64, origin *PointF, format int64, boundingBox *RectF) int32 {
	l := layoutAt(origin)
	var b [4]float32
	var fit, lines int32
	r := gp("GdipMeasureString", uintptr(graphics), uintptr(unsafe.Pointer(&string_[0])), uintptr(length), uintptr(font), uintptr(unsafe.Pointer(&l)), uintptr(format),
		uintptr(unsafe.Pointer(&b)), uintptr(unsafe.Pointer(&fit)), uintptr(unsafe.Pointer(&lines)))
	boundingBox.fromC(unsafe.Pointer(&b))
	return r
}

func GdipGraphics_MeasureString(graphics int64, string_ []uint16, length int32, font int64, origin *PointF, boundingBox *RectF) int32 {
	return measureString(graphics, string_, length, font, origin, 0, boundingBox)
}

func GdipGraphics_MeasureStringGraphicsStringLengthFontOriginFormatBoundingBox(graphics int64, string_ []uint16, length int32, font int64, origin *PointF, format int64, boundingBox *RectF) int32 {
	return measureString(graphics, string_, length, font, origin, format, boundingBox)
}

func GdipGraphics_ResetClip(graphics int64) int32 { return gp("GdipResetClip", uintptr(graphics)) }

func GdipGraphics_Restore(graphics int64, gstate int32) int32 {
	return gp("GdipRestoreGraphics", uintptr(graphics), uintptr(gstate))
}

func GdipGraphics_Save(graphics int64) int32 { return gpInt("GdipSaveGraphics", uintptr(graphics)) }

func GdipGraphics_ScaleTransform(graphics int64, sx float32, sy float32, order int32) int32 {
	return gp("GdipScaleWorldTransform", uintptr(graphics), f32(sx), f32(sy), uintptr(order))
}

func GdipGraphics_SetClip(graphics int64, hrgn int64, combineMode int32) int32 {
	return gp("GdipSetClipHrgn", uintptr(graphics), uintptr(hrgn), uintptr(combineMode))
}

func GdipGraphics_SetClipGraphicsRectCombineMode(graphics int64, rect *Rect, combineMode int32) int32 {
	return gp("GdipSetClipRectI", uintptr(graphics), uintptr(rect.X), uintptr(rect.Y), uintptr(rect.Width), uintptr(rect.Height), uintptr(combineMode))
}

func GdipGraphics_SetClipPath(graphics int64, path int64) int32 {
	return gp("GdipSetClipPath", uintptr(graphics), uintptr(path), 0)
}

func GdipGraphics_SetClipPathGraphicsPathCombineMode(graphics int64, path int64, combineMode int32) int32 {
	return gp("GdipSetClipPath", uintptr(graphics), uintptr(path), uintptr(combineMode))
}

func GdipGraphics_SetCompositingQuality(graphics int64, compositingQuality int32) int32 {
	return gp("GdipSetCompositingQuality", uintptr(graphics), uintptr(compositingQuality))
}

func GdipGraphics_SetPageUnit(graphics int64, unit int32) int32 {
	return gp("GdipSetPageUnit", uintptr(graphics), uintptr(unit))
}

func GdipGraphics_SetPixelOffsetMode(graphics int64, pixelOffsetMode int32) int32 {
	return gp("GdipSetPixelOffsetMode", uintptr(graphics), uintptr(pixelOffsetMode))
}

func GdipGraphics_SetSmoothingMode(graphics int64, smoothingMode int32) int32 {
	return gp("GdipSetSmoothingMode", uintptr(graphics), uintptr(smoothingMode))
}

func GdipGraphics_SetTransform(graphics int64, matrix int64) int32 {
	return gp("GdipSetWorldTransform", uintptr(graphics), uintptr(matrix))
}

func GdipGraphics_SetInterpolationMode(graphics int64, mode int32) int32 {
	return gp("GdipSetInterpolationMode", uintptr(graphics), uintptr(mode))
}

func GdipGraphics_SetTextRenderingHint(graphics int64, mode int32) int32 {
	return gp("GdipSetTextRenderingHint", uintptr(graphics), uintptr(mode))
}

func GdipGraphics_TranslateTransform(graphics int64, dx float32, dy float32, order int32) int32 {
	return gp("GdipTranslateWorldTransform", uintptr(graphics), f32(dx), f32(dy), uintptr(order))
}

func GdipGraphicsPath_new(fillMode int32) int64 { return gpOut("GdipCreatePath", uintptr(fillMode)) }

func GdipGraphicsPath_newPointsTypesCountFillMode(points []int32, types []int8, count int32, fillMode int32) int64 {
	return gpOut("GdipCreatePath2I", uintptr(unsafe.Pointer(&points[0])), uintptr(unsafe.Pointer(&types[0])), uintptr(count), uintptr(fillMode))
}

func GdipGraphicsPath_delete(path int64) { gp("GdipDeletePath", uintptr(path)) }

func GdipGraphicsPath_AddArc(path int64, x float32, y float32, width float32, height float32, startAngle float32, sweepAngle float32) int32 {
	return gp("GdipAddPathArc", uintptr(path), f32(x), f32(y), f32(width), f32(height), f32(startAngle), f32(sweepAngle))
}

func GdipGraphicsPath_AddBezier(path int64, x1 float32, y1 float32, x2 float32, y2 float32, x3 float32, y3 float32, x4 float32, y4 float32) int32 {
	return gp("GdipAddPathBezier", uintptr(path), f32(x1), f32(y1), f32(x2), f32(y2), f32(x3), f32(y3), f32(x4), f32(y4))
}

func GdipGraphicsPath_AddLine(path int64, x1 float32, y1 float32, x2 float32, y2 float32) int32 {
	return gp("GdipAddPathLine", uintptr(path), f32(x1), f32(y1), f32(x2), f32(y2))
}

func GdipGraphicsPath_AddPath(path int64, addingPath int64, connect bool) int32 {
	return gp("GdipAddPathPath", uintptr(path), uintptr(addingPath), b2u(connect))
}

func GdipGraphicsPath_AddRectangle(path int64, rect *RectF) int32 {
	return gp("GdipAddPathRectangle", uintptr(path), f32(rect.X), f32(rect.Y), f32(rect.Width), f32(rect.Height))
}

func GdipGraphicsPath_AddString(path int64, string_ []uint16, length int32, family int64, style int32, emSize float32, origin *PointF, format int64) int32 {
	l := layoutAt(origin)
	return gp("GdipAddPathString", uintptr(path), uintptr(unsafe.Pointer(&string_[0])), uintptr(length), cell(family), uintptr(style), f32(emSize), uintptr(unsafe.Pointer(&l)), uintptr(format))
}

func GdipGraphicsPath_CloseFigure(path int64) int32 { return gp("GdipClosePathFigure", uintptr(path)) }

func GdipGraphicsPath_Clone(path int64) int64 { return gpOut("GdipClonePath", uintptr(path)) }

func GdipGraphicsPath_Flatten(path int64, matrix int64, flatness float32) int32 {
	return gp("GdipFlattenPath", uintptr(path), uintptr(matrix), f32(flatness))
}

func GdipGraphicsPath_GetBounds(path int64, bounds *RectF, matrix int64, pen int64) int32 {
	var b [4]float32
	r := gp("GdipGetPathWorldBounds", uintptr(path), uintptr(unsafe.Pointer(&b)), uintptr(matrix), uintptr(pen))
	bounds.fromC(unsafe.Pointer(&b))
	return r
}

func GdipGraphicsPath_GetLastPoint(path int64, lastPoint *PointF) int32 {
	var b [2]float32
	r := gp("GdipGetPathLastPoint", uintptr(path), uintptr(unsafe.Pointer(&b)))
	lastPoint.X, lastPoint.Y = b[0], b[1]
	return r
}

func GdipGraphicsPath_GetPathPoints(path int64, points []float32, count int32) int32 {
	return gp("GdipGetPathPoints", uintptr(path), uintptr(unsafe.Pointer(&points[0])), uintptr(count))
}

func GdipGraphicsPath_GetPathTypes(path int64, types []int8, count int32) int32 {
	return gp("GdipGetPathTypes", uintptr(path), uintptr(unsafe.Pointer(&types[0])), uintptr(count))
}

func GdipGraphicsPath_GetPointCount(path int64) int32 {
	return gpInt("GdipGetPointCount", uintptr(path))
}

func GdipGraphicsPath_IsOutlineVisible(path int64, x float32, y float32, pen int64, g int64) bool {
	return gpInt("GdipIsOutlineVisiblePathPoint", uintptr(path), f32(x), f32(y), uintptr(pen), uintptr(g)) != 0
}

func GdipGraphicsPath_IsVisible(path int64, x float32, y float32, g int64) bool {
	return gpInt("GdipIsVisiblePathPoint", uintptr(path), f32(x), f32(y), uintptr(g)) != 0
}

func GdipGraphicsPath_SetFillMode(path int64, fillmode int32) int32 {
	return gp("GdipSetPathFillMode", uintptr(path), uintptr(fillmode))
}

func GdipGraphicsPath_StartFigure(path int64) int32 { return gp("GdipStartPathFigure", uintptr(path)) }

func GdipGraphicsPath_Transform(path int64, matrix int64) int32 {
	return gp("GdipTransformPath", uintptr(path), uintptr(matrix))
}

func GdipHatchBrush_new(hatchStyle int32, foreColor int32, backColor int32) int64 {
	return gpOut("GdipCreateHatchBrush", uintptr(hatchStyle), uintptr(uint32(foreColor)), uintptr(uint32(backColor)))
}

func GdipHatchBrush_delete(brush int64) { gp("GdipDeleteBrush", uintptr(brush)) }

func GdipImage_delete(image int64) { gp("GdipDisposeImage", uintptr(image)) }

func GdipImage_Clone(image int64) int64 { return gpOut("GdipCloneImage", uintptr(image)) }

// A failing constructor returned handle 0 already; the C++ object's lastStatus has no flat counterpart.
func GdipImage_GetLastStatus(image int64) int32 { return 0 }

func GdipImage_GetPixelFormat(image int64) int32 {
	return gpInt("GdipGetImagePixelFormat", uintptr(image))
}

func GdipImage_GetWidth(image int64) int32 { return gpInt("GdipGetImageWidth", uintptr(image)) }

func GdipImage_GetHeight(image int64) int32 { return gpInt("GdipGetImageHeight", uintptr(image)) }

func GdipImage_GetPalette(image int64, palette int64, size int32) int32 {
	return gp("GdipGetImagePalette", uintptr(image), uintptr(palette), uintptr(size))
}

func GdipImage_GetPaletteSize(image int64) int32 {
	return gpInt("GdipGetImagePaletteSize", uintptr(image))
}

func GdipImageAttributes_new() int64 { return gpOut("GdipCreateImageAttributes") }

func GdipImageAttributes_delete(attrib int64) { gp("GdipDisposeImageAttributes", uintptr(attrib)) }

func GdipImageAttributes_SetWrapMode(attrib int64, wrap int32) int32 {
	return gp("GdipSetImageAttributesWrapMode", uintptr(attrib), uintptr(wrap), 0, 0)
}

func GdipImageAttributes_SetColorMatrix(attrib int64, matrix []float32, mode int32, type_ int32) int32 {
	return gp("GdipSetImageAttributesColorMatrix", uintptr(attrib), uintptr(type_), 1, uintptr(unsafe.Pointer(&matrix[0])), 0, uintptr(mode))
}

func GdipLinearGradientBrush_new(point1 *PointF, point2 *PointF, color1 int32, color2 int32) int64 {
	p1 := [2]float32{point1.X, point1.Y}
	p2 := [2]float32{point2.X, point2.Y}
	return gpOut("GdipCreateLineBrush", uintptr(unsafe.Pointer(&p1)), uintptr(unsafe.Pointer(&p2)), uintptr(uint32(color1)), uintptr(uint32(color2)), 0)
}

func GdipLinearGradientBrush_delete(brush int64) { gp("GdipDeleteBrush", uintptr(brush)) }

func GdipLinearGradientBrush_SetInterpolationColors(brush int64, presetColors []int32, blendPositions []float32, count int32) int32 {
	return gp("GdipSetLinePresetBlend", uintptr(brush), uintptr(unsafe.Pointer(&presetColors[0])), uintptr(unsafe.Pointer(&blendPositions[0])), uintptr(count))
}

func GdipLinearGradientBrush_SetWrapMode(brush int64, wrapMode int32) int32 {
	return gp("GdipSetLineWrapMode", uintptr(brush), uintptr(wrapMode))
}

func GdipLinearGradientBrush_ResetTransform(brush int64) int32 {
	return gp("GdipResetLineTransform", uintptr(brush))
}

func GdipLinearGradientBrush_ScaleTransform(brush int64, sx float32, sy float32, order int32) int32 {
	return gp("GdipScaleLineTransform", uintptr(brush), f32(sx), f32(sy), uintptr(order))
}

func GdipLinearGradientBrush_TranslateTransform(brush int64, dx float32, dy float32, order int32) int32 {
	return gp("GdipTranslateLineTransform", uintptr(brush), f32(dx), f32(dy), uintptr(order))
}

func GdipMatrix_new(m11 float32, m12 float32, m21 float32, m22 float32, dx float32, dy float32) int64 {
	return gpOut("GdipCreateMatrix2", f32(m11), f32(m12), f32(m21), f32(m22), f32(dx), f32(dy))
}

func GdipMatrix_delete(matrix int64) { gp("GdipDeleteMatrix", uintptr(matrix)) }

func GdipMatrix_GetElements(matrix int64, m []float32) int32 {
	return gp("GdipGetMatrixElements", uintptr(matrix), uintptr(unsafe.Pointer(&m[0])))
}

func GdipMatrix_Invert(matrix int64) int32 { return gp("GdipInvertMatrix", uintptr(matrix)) }

func GdipMatrix_IsIdentity(matrix int64) bool {
	return gpInt("GdipIsMatrixIdentity", uintptr(matrix)) != 0
}

func GdipMatrix_Multiply(matrix int64, matrix1 int64, order int32) int32 {
	return gp("GdipMultiplyMatrix", uintptr(matrix), uintptr(matrix1), uintptr(order))
}

func GdipMatrix_Rotate(matrix int64, angle float32, order int32) int32 {
	return gp("GdipRotateMatrix", uintptr(matrix), f32(angle), uintptr(order))
}

func GdipMatrix_Scale(matrix int64, scaleX float32, scaleY float32, order int32) int32 {
	return gp("GdipScaleMatrix", uintptr(matrix), f32(scaleX), f32(scaleY), uintptr(order))
}

func GdipMatrix_Shear(matrix int64, shearX float32, shearY float32, order int32) int32 {
	return gp("GdipShearMatrix", uintptr(matrix), f32(shearX), f32(shearY), uintptr(order))
}

func GdipMatrix_TransformPoints(matrix int64, pts *PointF, count int32) int32 {
	b := [2]float32{pts.X, pts.Y}
	r := gp("GdipTransformMatrixPoints", uintptr(matrix), uintptr(unsafe.Pointer(&b)), uintptr(count))
	pts.X, pts.Y = b[0], b[1]
	return r
}

func GdipMatrix_TransformPointsMatrixPtsCount(matrix int64, pts []float32, count int32) int32 {
	return gp("GdipTransformMatrixPoints", uintptr(matrix), uintptr(unsafe.Pointer(&pts[0])), uintptr(count))
}

func GdipMatrix_TransformVectors(matrix int64, pts *PointF, count int32) int32 {
	b := [2]float32{pts.X, pts.Y}
	r := gp("GdipVectorTransformMatrixPoints", uintptr(matrix), uintptr(unsafe.Pointer(&b)), uintptr(count))
	pts.X, pts.Y = b[0], b[1]
	return r
}

func GdipMatrix_Translate(matrix int64, offsetX float32, offsetY float32, order int32) int32 {
	return gp("GdipTranslateMatrix", uintptr(matrix), f32(offsetX), f32(offsetY), uintptr(order))
}

func GdipMatrix_SetElements(matrix int64, m11 float32, m12 float32, m21 float32, m22 float32, dx float32, dy float32) int32 {
	return gp("GdipSetMatrixElements", uintptr(matrix), f32(m11), f32(m12), f32(m21), f32(m22), f32(dx), f32(dy))
}

// GdipMoveMemory copies a native ColorPalette (Flags, Count, Entries...) into the Java-shaped struct.
func GdipMoveMemory(Destination *ColorPalette, SourcePtr int64, Length int32) {
	p := unsafe.Add(unsafe.Pointer(nil), uintptr(SourcePtr))
	Destination.Flags = *(*int32)(p)
	Destination.Count = *(*int32)(unsafe.Add(p, 4))
	for i := 0; i < len(Destination.Entries) && int32(i) < Destination.Count; i++ {
		Destination.Entries[i] = *(*int32)(unsafe.Add(p, 8+4*i))
	}
}

func GdipMoveMemoryDestinationSourcePtr(Destination *BitmapData, SourcePtr int64) {
	Destination.fromC(unsafe.Add(unsafe.Pointer(nil), uintptr(SourcePtr)))
}

func GdipPathGradientBrush_new(path int64) int64 {
	return gpOut("GdipCreatePathGradientFromPath", uintptr(path))
}

func GdipPathGradientBrush_delete(brush int64) { gp("GdipDeleteBrush", uintptr(brush)) }

func GdipPathGradientBrush_SetCenterColor(brush int64, color int32) int32 {
	return gp("GdipSetPathGradientCenterColor", uintptr(brush), uintptr(uint32(color)))
}

func GdipPathGradientBrush_SetCenterPoint(brush int64, pt *PointF) int32 {
	b := [2]float32{pt.X, pt.Y}
	return gp("GdipSetPathGradientCenterPoint", uintptr(brush), uintptr(unsafe.Pointer(&b)))
}

func GdipPathGradientBrush_SetInterpolationColors(brush int64, presetColors []int32, blendPositions []float32, count int32) int32 {
	return gp("GdipSetPathGradientPresetBlend", uintptr(brush), uintptr(unsafe.Pointer(&presetColors[0])), uintptr(unsafe.Pointer(&blendPositions[0])), uintptr(count))
}

func GdipPathGradientBrush_SetSurroundColors(brush int64, colors []int32, count []int32) int32 {
	return gp("GdipSetPathGradientSurroundColorsWithCount", uintptr(brush), uintptr(unsafe.Pointer(&colors[0])), uintptr(unsafe.Pointer(&count[0])))
}

func GdipPathGradientBrush_SetGraphicsPath(brush int64, path int64) int32 {
	return gp("GdipSetPathGradientPath", uintptr(brush), uintptr(path))
}

func GdipPathGradientBrush_SetWrapMode(brush int64, wrapMode int32) int32 {
	return gp("GdipSetPathGradientWrapMode", uintptr(brush), uintptr(wrapMode))
}

func GdipPen_new(brush int64, width float32) int64 {
	return gpOut("GdipCreatePen2", uintptr(brush), f32(width), 0)
}

func GdipPen_delete(pen int64) { gp("GdipDeletePen", uintptr(pen)) }

func GdipPen_GetBrush(pen int64) int64 { return gpOut("GdipGetPenBrushFill", uintptr(pen)) }

func GdipPen_SetBrush(pen int64, brush int64) int32 {
	return gp("GdipSetPenBrushFill", uintptr(pen), uintptr(brush))
}

func GdipPen_SetDashOffset(pen int64, dashOffset float32) int32 {
	return gp("GdipSetPenDashOffset", uintptr(pen), f32(dashOffset))
}

func GdipPen_SetDashPattern(pen int64, dashArray []float32, count int32) int32 {
	return gp("GdipSetPenDashArray", uintptr(pen), uintptr(unsafe.Pointer(&dashArray[0])), uintptr(count))
}

func GdipPen_SetDashStyle(pen int64, dashStyle int32) int32 {
	return gp("GdipSetPenDashStyle", uintptr(pen), uintptr(dashStyle))
}

func GdipPen_SetLineCap(pen int64, startCap int32, endCap int32, dashCap int32) int32 {
	return gp("GdipSetPenLineCap197819", uintptr(pen), uintptr(startCap), uintptr(endCap), uintptr(dashCap))
}

func GdipPen_SetLineJoin(pen int64, lineJoin int32) int32 {
	return gp("GdipSetPenLineJoin", uintptr(pen), uintptr(lineJoin))
}

func GdipPen_SetMiterLimit(pen int64, miterLimit float32) int32 {
	return gp("GdipSetPenMiterLimit", uintptr(pen), f32(miterLimit))
}

func GdipPen_SetWidth(pen int64, width float32) int32 {
	return gp("GdipSetPenWidth", uintptr(pen), f32(width))
}

func GdipPoint_new(x int32, y int32) int64 {
	h := gdipAlloc(8)
	*(*[2]int32)(unsafe.Add(unsafe.Pointer(nil), uintptr(h))) = [2]int32{x, y}
	return h
}

func GdipPoint_delete(point int64) { gdipFree(point) }

func GdipRegion_new(hRgn int64) int64 { return gpOut("GdipCreateRegionHrgn", uintptr(hRgn)) }

func GdipRegion_newGraphicsPath(path int64) int64 {
	return gpOut("GdipCreateRegionPath", uintptr(path))
}

func GdipRegion_new0() int64 { return gpOut("GdipCreateRegion") }

func GdipRegion_delete(region int64) { gp("GdipDeleteRegion", uintptr(region)) }

func GdipRegion_GetHRGN(region int64, graphics int64) int64 {
	return gpOut("GdipGetRegionHRgn", uintptr(region), uintptr(graphics))
}

func GdipRegion_IsInfinite(region int64, graphics int64) bool {
	return gpInt("GdipIsInfiniteRegion", uintptr(region), uintptr(graphics)) != 0
}

func GdipSolidBrush_new(color int32) int64 {
	return gpOut("GdipCreateSolidFill", uintptr(uint32(color)))
}

func GdipSolidBrush_delete(brush int64) { gp("GdipDeleteBrush", uintptr(brush)) }

func GdipStringFormat_delete(format int64) { gp("GdipDeleteStringFormat", uintptr(format)) }

func GdipStringFormat_Clone(format int64) int64 {
	return gpOut("GdipCloneStringFormat", uintptr(format))
}

func GdipStringFormat_GenericDefault() int64 { return gpOut("GdipStringFormatGetGenericDefault") }

func GdipStringFormat_GenericTypographic() int64 {
	return gpOut("GdipStringFormatGetGenericTypographic")
}

func GdipStringFormat_GetFormatFlags(format int64) int32 {
	return gpInt("GdipGetStringFormatFlags", uintptr(format))
}

func GdipStringFormat_SetHotkeyPrefix(format int64, hotkeyPrefix int32) int32 {
	return gp("GdipSetStringFormatHotkeyPrefix", uintptr(format), uintptr(hotkeyPrefix))
}

func GdipStringFormat_SetFormatFlags(format int64, flags int32) int32 {
	return gp("GdipSetStringFormatFlags", uintptr(format), uintptr(flags))
}

func GdipStringFormat_SetTabStops(format int64, firstTabOffset float32, count int32, tabStops []float32) int32 {
	return gp("GdipSetStringFormatTabStops", uintptr(format), f32(firstTabOffset), uintptr(count), uintptr(unsafe.Pointer(&tabStops[0])))
}

func GdipTextureBrush_new(image int64, wrapMode int32, dstX float32, dstY float32, dstWidth float32, dstHeight float32) int64 {
	return gpOut("GdipCreateTexture2", uintptr(image), uintptr(wrapMode), f32(dstX), f32(dstY), f32(dstWidth), f32(dstHeight))
}

func GdipTextureBrush_newImageRectAttribs(image int64, rect *Rect, attribs int64) int64 {
	return gpOut("GdipCreateTextureIAI", uintptr(image), uintptr(attribs), uintptr(rect.X), uintptr(rect.Y), uintptr(rect.Width), uintptr(rect.Height))
}

func GdipTextureBrush_delete(brush int64) { gp("GdipDeleteBrush", uintptr(brush)) }

func GdipTextureBrush_SetTransform(brush int64, matrix int64) int32 {
	return gp("GdipSetTextureTransform", uintptr(brush), uintptr(matrix))
}

func GdipTextureBrush_ResetTransform(brush int64) int32 {
	return gp("GdipResetTextureTransform", uintptr(brush))
}

func GdipTextureBrush_ScaleTransform(brush int64, sx float32, sy float32, order int32) int32 {
	return gp("GdipScaleTextureTransform", uintptr(brush), f32(sx), f32(sy), uintptr(order))
}

func GdipTextureBrush_TranslateTransform(brush int64, dx float32, dy float32, order int32) int32 {
	return gp("GdipTranslateTextureTransform", uintptr(brush), f32(dx), f32(dy), uintptr(order))
}

func GdipTextureBrush_GetImage(brush int64) int64 {
	return gpOut("GdipGetTextureImage", uintptr(brush))
}

// gdipNames lists the flat functions above, for Probe.
var gdipNames = []string{
	"GdipAddPathArc",
	"GdipAddPathBezier",
	"GdipAddPathLine",
	"GdipAddPathPath",
	"GdipAddPathRectangle",
	"GdipAddPathString",
	"GdipBitmapLockBits",
	"GdipBitmapUnlockBits",
	"GdipCloneBrush",
	"GdipCloneImage",
	"GdipClonePath",
	"GdipCloneStringFormat",
	"GdipClosePathFigure",
	"GdipCreateBitmapFromFile",
	"GdipCreateBitmapFromFileICM",
	"GdipCreateBitmapFromHBITMAP",
	"GdipCreateBitmapFromHICON",
	"GdipCreateBitmapFromScan0",
	"GdipCreateFont",
	"GdipCreateFontFamilyFromName",
	"GdipCreateFontFromLogfontW",
	"GdipCreateFromHDC",
	"GdipCreateHBITMAPFromBitmap",
	"GdipCreateHICONFromBitmap",
	"GdipCreateHatchBrush",
	"GdipCreateImageAttributes",
	"GdipCreateLineBrush",
	"GdipCreateMatrix2",
	"GdipCreatePath",
	"GdipCreatePath2I",
	"GdipCreatePathGradientFromPath",
	"GdipCreatePen2",
	"GdipCreateRegion",
	"GdipCreateRegionHrgn",
	"GdipCreateRegionPath",
	"GdipCreateSolidFill",
	"GdipCreateTexture2",
	"GdipCreateTextureIAI",
	"GdipDeleteBrush",
	"GdipDeleteFont",
	"GdipDeleteFontFamily",
	"GdipDeleteGraphics",
	"GdipDeleteMatrix",
	"GdipDeletePath",
	"GdipDeletePen",
	"GdipDeletePrivateFontCollection",
	"GdipDeleteRegion",
	"GdipDeleteStringFormat",
	"GdipDisposeImage",
	"GdipDisposeImageAttributes",
	"GdipDrawArcI",
	"GdipDrawDriverString",
	"GdipDrawEllipseI",
	"GdipDrawImageI",
	"GdipDrawImageRectRectI",
	"GdipDrawLineI",
	"GdipDrawLinesI",
	"GdipDrawPath",
	"GdipDrawPolygonI",
	"GdipDrawRectangleI",
	"GdipDrawString",
	"GdipFillEllipseI",
	"GdipFillPath",
	"GdipFillPieI",
	"GdipFillPolygonI",
	"GdipFillRectangleI",
	"GdipFlattenPath",
	"GdipFlush",
	"GdipGetBrushType",
	"GdipGetClip",
	"GdipGetClipBounds",
	"GdipGetClipBoundsI",
	"GdipGetDC",
	"GdipGetFamily",
	"GdipGetFamilyName",
	"GdipGetFontSize",
	"GdipGetFontStyle",
	"GdipGetImageHeight",
	"GdipGetImagePalette",
	"GdipGetImagePaletteSize",
	"GdipGetImagePixelFormat",
	"GdipGetImageWidth",
	"GdipGetInterpolationMode",
	"GdipGetLogFontW",
	"GdipGetMatrixElements",
	"GdipGetPathLastPoint",
	"GdipGetPathPoints",
	"GdipGetPathTypes",
	"GdipGetPathWorldBounds",
	"GdipGetPenBrushFill",
	"GdipGetPointCount",
	"GdipGetRegionHRgn",
	"GdipGetSmoothingMode",
	"GdipGetStringFormatFlags",
	"GdipGetTextRenderingHint",
	"GdipGetTextureImage",
	"GdipGetVisibleClipBoundsI",
	"GdipGetWorldTransform",
	"GdipInvertMatrix",
	"GdipIsInfiniteRegion",
	"GdipIsMatrixIdentity",
	"GdipIsOutlineVisiblePathPoint",
	"GdipIsVisiblePathPoint",
	"GdipMeasureDriverString",
	"GdipMeasureString",
	"GdipMultiplyMatrix",
	"GdipNewPrivateFontCollection",
	"GdipPrivateAddFontFile",
	"GdipReleaseDC",
	"GdipResetClip",
	"GdipResetLineTransform",
	"GdipResetTextureTransform",
	"GdipRestoreGraphics",
	"GdipRotateMatrix",
	"GdipSaveGraphics",
	"GdipScaleLineTransform",
	"GdipScaleMatrix",
	"GdipScaleTextureTransform",
	"GdipScaleWorldTransform",
	"GdipSetClipHrgn",
	"GdipSetClipPath",
	"GdipSetClipRectI",
	"GdipSetCompositingQuality",
	"GdipSetImageAttributesColorMatrix",
	"GdipSetImageAttributesWrapMode",
	"GdipSetInterpolationMode",
	"GdipSetLinePresetBlend",
	"GdipSetLineWrapMode",
	"GdipSetMatrixElements",
	"GdipSetPageUnit",
	"GdipSetPathFillMode",
	"GdipSetPathGradientCenterColor",
	"GdipSetPathGradientCenterPoint",
	"GdipSetPathGradientPath",
	"GdipSetPathGradientPresetBlend",
	"GdipSetPathGradientSurroundColorsWithCount",
	"GdipSetPathGradientWrapMode",
	"GdipSetPenBrushFill",
	"GdipSetPenDashArray",
	"GdipSetPenDashOffset",
	"GdipSetPenDashStyle",
	"GdipSetPenLineCap197819",
	"GdipSetPenLineJoin",
	"GdipSetPenMiterLimit",
	"GdipSetPenWidth",
	"GdipSetPixelOffsetMode",
	"GdipSetSmoothingMode",
	"GdipSetStringFormatFlags",
	"GdipSetStringFormatHotkeyPrefix",
	"GdipSetStringFormatTabStops",
	"GdipSetTextRenderingHint",
	"GdipSetTextureTransform",
	"GdipSetWorldTransform",
	"GdipShearMatrix",
	"GdipStartPathFigure",
	"GdipStringFormatGetGenericDefault",
	"GdipStringFormatGetGenericTypographic",
	"GdipTransformMatrixPoints",
	"GdipTransformPath",
	"GdipTranslateLineTransform",
	"GdipTranslateMatrix",
	"GdipTranslateTextureTransform",
	"GdipTranslateWorldTransform",
	"GdipVectorTransformMatrixPoints",
	"GdiplusShutdown",
	"GdiplusStartup",
}
