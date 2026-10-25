package gowt

import "runtime"

// fieldOpt is one label | control row of Fields.
type fieldOpt struct {
	label   string
	control Option
	top     bool
	opts    []Option
}

func (fieldOpt) option() {}

type fieldTop struct{}

func (fieldTop) option() {}

// FieldTop, an option of Field, aligns the label with the top of a tall control (a multi-line Text, a
// List) instead of centring it on the control.
func FieldTop() Option { return fieldTop{} }

// Field is a row of Fields: a Label with the text label and the control beside it. opts go to the
// label (Foreground, Typeface, Tooltip), FieldTop moves it to the top of the row.
func Field(label string, control Option, opts ...Option) Option {
	f := fieldOpt{label: label, control: control}
	for _, o := range opts {
		if _, ok := o.(fieldTop); ok {
			f.top = true
		} else {
			f.opts = append(f.opts, o)
		}
	}
	return f
}

// fieldLabelAlign is the horizontal place of a form label: trailing, against the field, on macOS
// (System Settings, Finder dialogs); leading on Windows and GNOME.
func fieldLabelAlign() Align {
	if runtime.GOOS == "darwin" {
		return AlignEnd
	}
	return AlignStart
}

// Fields creates a form: a two-column Grid in which every Field is a label and its control. The labels
// are aligned the way the OS does it, trailing on macOS and leading on Windows and GNOME, and
// centred on the control's row. Give the controls GrowX() to fill the second column; other options
// are those of Grid (Gap, Padding, GrowX) and may be mixed with the Field rows.
// Options: Gap, Padding, Border, Field.
func Fields(opts ...Option) Option {
	var out []Option
	for _, o := range opts {
		f, ok := o.(fieldOpt)
		if !ok {
			out = append(out, o)
			continue
		}
		v := AlignCenter
		if f.top {
			v = AlignStart
		}
		label := append([]Option{Place(fieldLabelAlign(), v)}, f.opts...)
		out = append(out, Label(f.label, label...), f.control)
	}
	return Grid(2, out...)
}
