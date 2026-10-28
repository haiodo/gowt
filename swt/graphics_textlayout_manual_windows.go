// TextLayout._getOffset (manual.txt), from Eclipse SWT win32/org/eclipse/swt/graphics/TextLayout.java:
// MOVEMENT_CLUSTER also steps over a ZWJ emoji sequence (clusterJoined), as on macOS and GTK.
//
// Copyright (c) 2000, 2022 IBM Corporation and others.
//
// SPDX-License-Identifier: EPL-2.0

package swt

import (
	"fmt"
	"unicode"

	"github.com/haiodo/gowt/internal/jrt"
	"github.com/haiodo/gowt/internal/win32"
)

func (this *TextLayout) _getOffset(offset int32, movement int32, forward bool) int32 {
	this.ComputeRuns(nil)
	var length int32 = jrt.StringLength(this.text)
	if !(0 <= offset && offset <= length) {
		ErrorCodeThrowableDetail(ERROR_INVALID_RANGE, nil, fmt.Sprintf(" [offset value: %d]", offset))
	}
	if forward && offset == length {
		return length
	}
	if !forward && offset == 0 {
		return 0
	}
	var step int32
	if forward {
		step = 1
	} else {
		step = -1
	}
	if (movement & MOVEMENT_CHAR) != 0 {
		return offset + step
	}
	length = jrt.StringLength(this.segmentsText)
	offset = this.TranslateOffset(offset)
	var logAttr *win32.SCRIPT_LOGATTR = win32.NewSCRIPT_LOGATTR()
	var properties *win32.SCRIPT_PROPERTIES = win32.NewSCRIPT_PROPERTIES()
	var i int32
	if forward {
		i = 0
	} else {
		i = int32(len(this.allRuns)) - 1
	}
	offset = this.ValidadeOffset(offset, step)
	for {
		var run *TextLayout_StyleItem = this.allRuns[i]
		if run.start <= offset && offset < run.start+run.length {
			if run.lineBreak && !run.softBreak {
				return this.UntranslateOffset(run.start)
			}
			if run.tab {
				return this.UntranslateOffset(run.start)
			}
			win32.OSMoveMemoryOverload83(properties, this.device.scripts[run.analysis.EScript], win32.SCRIPT_PROPERTIESSizeof)
			var isComplex bool = properties.FNeedsCaretInfo || properties.FNeedsWordBreaking
			if isComplex {
				this.BreakRun(run)
			}
			for run.start <= offset && offset < run.start+run.length {
				if isComplex {
					win32.OSMoveMemoryOverload82(logAttr, run.psla+int64(((offset-run.start)*win32.SCRIPT_LOGATTRSizeof)), win32.SCRIPT_LOGATTRSizeof)
				}
				switch movement {
				case MOVEMENT_CLUSTER:
					{
						if !properties.FNeedsCaretInfo || (!logAttr.FInvalid && logAttr.FCharStop) {
							var ch uint16 = jrt.CharAt(this.segmentsText, offset)
							stop := offset
							if 0xDC00 <= int32(ch) && int32(ch) <= 0xDFFF {
								if offset > 0 {
									ch = jrt.CharAt(this.segmentsText, offset-1)
									if 0xD800 <= int32(ch) && int32(ch) <= 0xDBFF {
										stop += step
									}
								}
							}
							if !clusterJoined(this.segmentsText, stop) {
								return this.UntranslateOffset(stop)
							}
						}
						break
					}
				case MOVEMENT_WORD_START, MOVEMENT_WORD:
					{
						if properties.FNeedsWordBreaking {
							if !logAttr.FInvalid && logAttr.FWordStop {
								return this.UntranslateOffset(offset)
							}
						} else {
							if offset > 0 {
								var letterOrDigit bool = (unicode.IsLetter(rune(jrt.CharAt(this.segmentsText, offset))) || unicode.IsDigit(rune(jrt.CharAt(this.segmentsText, offset))))
								var previousLetterOrDigit bool = (unicode.IsLetter(rune(jrt.CharAt(this.segmentsText, offset-1))) || unicode.IsDigit(rune(jrt.CharAt(this.segmentsText, offset-1))))
								if letterOrDigit != previousLetterOrDigit || !letterOrDigit {
									if !unicode.IsSpace(rune(jrt.CharAt(this.segmentsText, offset))) {
										return this.UntranslateOffset(offset)
									}
								}
							}
						}
						break
					}
				case MOVEMENT_WORD_END:
					{
						if offset > 0 {
							var isLetterOrDigit bool = (unicode.IsLetter(rune(jrt.CharAt(this.segmentsText, offset))) || unicode.IsDigit(rune(jrt.CharAt(this.segmentsText, offset))))
							var previousLetterOrDigit bool = (unicode.IsLetter(rune(jrt.CharAt(this.segmentsText, offset-1))) || unicode.IsDigit(rune(jrt.CharAt(this.segmentsText, offset-1))))
							if !isLetterOrDigit && previousLetterOrDigit {
								return this.UntranslateOffset(offset)
							}
						}
						break
					}
				}
				offset = this.ValidadeOffset(offset, step)
			}
		}
		i += step
		if !(0 <= i && i < int32(len(this.allRuns))-1 && 0 <= offset && offset < length) {
			break
		}
	}
	var cond489 int32
	if forward {
		cond489 = jrt.StringLength(this.text)
	} else {
		cond489 = 0
	}
	return cond489
}
