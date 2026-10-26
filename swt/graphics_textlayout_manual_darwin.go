// TextLayout._getOffset (manual.txt): MOVEMENT_CLUSTER steps over a grapheme cluster, ZWJ emoji
// sequences included, as native text does; SWT steps over one surrogate pair.

package swt

import (
	"github.com/haiodo/gowt/internal/cocoa"
	"github.com/haiodo/gowt/internal/jrt"
)

func (this *TextLayout) _getOffset(offset int32, movement int32, forward bool) int32 {
	this.CheckLayout()
	this.ComputeRuns()
	var length int32 = jrt.StringLength(this.text)
	if !(0 <= offset && offset <= length) {
		Error(ERROR_INVALID_RANGE)
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
	switch movement {
	case MOVEMENT_CLUSTER:
		if forward {
			loc, n := cocoa.GraphemeRange(this.text, int(offset))
			offset = int32(loc + n)
		} else {
			loc, _ := cocoa.GraphemeRange(this.text, int(offset-1))
			offset = int32(loc)
		}
		break
	case MOVEMENT_WORD:
		{
			offset = this.TranslateOffset(offset)
			offset = int32(this.textStorage.NextWordFromIndex(int64(offset), forward))
			return this.UntranslateOffset(offset)
		}
	case MOVEMENT_WORD_END:
		{
			offset = this.TranslateOffset(offset)
			if forward {
				offset = int32(this.textStorage.NextWordFromIndex(int64(offset), true))
			} else {
				length = this.TranslateOffset(length)
				var result int32 = 0
				for result < length {
					var wordEnd int32 = int32(this.textStorage.NextWordFromIndex(int64(result), true))
					if wordEnd >= offset {
						offset = result
						break
					}
					result = wordEnd
				}
			}
			return this.UntranslateOffset(offset)
		}
	case MOVEMENT_WORD_START:
		{
			offset = this.TranslateOffset(offset)
			if forward {
				var result int32 = this.TranslateOffset(length)
				for result > 0 {
					var wordStart int32 = int32(this.textStorage.NextWordFromIndex(int64(result), false))
					if wordStart <= offset {
						offset = result
						break
					}
					result = wordStart
				}
			} else {
				offset = int32(this.textStorage.NextWordFromIndex(int64(offset), false))
			}
			return this.UntranslateOffset(offset)
		}
	}
	return offset
}
