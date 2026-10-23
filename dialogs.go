package gowt

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/haiodo/gowt/swt"
)

// Icon is the picture of a message box.
type Icon int

// Message box icons.
const (
	IconNone Icon = iota
	IconInfo
	IconWarning
	IconError
	IconQuestion
)

// Buttons selects the buttons of a message box.
type Buttons int

// Button sets of a message box.
const (
	ButtonsOK Buttons = iota
	ButtonsOKCancel
	ButtonsYesNo
	ButtonsYesNoCancel
	ButtonsRetryCancel
	ButtonsAbortRetryIgnore
)

// Answer is the button that closed a message box. Closing the box with the window button gives AnswerCancel
// (or AnswerNo, AnswerOK if there is no Cancel).
type Answer int

// Answers a message box returns.
const (
	AnswerOK Answer = iota
	AnswerCancel
	AnswerYes
	AnswerNo
	AnswerRetry
	AnswerAbort
	AnswerIgnore
)

// Message describes a message box.
type Message struct {
	Title, Text string
	Icon        Icon
	Buttons     Buttons
}

var (
	iconStyle   = [...]int32{0, swt.ICON_INFORMATION, swt.ICON_WARNING, swt.ICON_ERROR, swt.ICON_QUESTION}
	buttonStyle = [...]int32{swt.OK, swt.OK | swt.CANCEL, swt.YES | swt.NO, swt.YES | swt.NO | swt.CANCEL,
		swt.RETRY | swt.CANCEL, swt.ABORT | swt.RETRY | swt.IGNORE}
	answerOf = map[int32]Answer{swt.OK: AnswerOK, swt.CANCEL: AnswerCancel, swt.YES: AnswerYes,
		swt.NO: AnswerNo, swt.RETRY: AnswerRetry, swt.ABORT: AnswerAbort, swt.IGNORE: AnswerIgnore}
	// answerStyle and answerName are indexed by Answer.
	answerStyle = [...]int32{swt.OK, swt.CANCEL, swt.YES, swt.NO, swt.RETRY, swt.ABORT, swt.IGNORE}
	answerName  = [...]string{"OK", "Cancel", "Yes", "No", "Retry", "Abort", "Ignore"}
)

// dialogCall is a dialog as the QA driver sees it (qa.Dialog).
type dialogCall struct {
	Kind, Title, Message string
	Buttons              []string
	Dir, Name            string
	Save, Multi          bool
	Filters              []string
}

// Set by a QA build (qa_on.go). With the QA server running a dialog takes the test's scripted answer
// and is not shown: a native modal loop would block the UI thread.
var (
	qaDialog       func(dialogCall) (reply json.RawMessage, cancel, handled bool)
	qaDialogFailed func(msg string)
)

func scripted(c dialogCall) (reply json.RawMessage, cancel, handled bool) {
	if qaDialog == nil {
		return nil, false, false
	}
	return qaDialog(c)
}

// replyAs decodes a scripted reply into v; a reply of the wrong shape fails the test's action.
func replyAs(kind string, reply json.RawMessage, v any) bool {
	if err := json.Unmarshal(reply, v); err != nil {
		qaDialogFailed(fmt.Sprintf("the %s dialog got the reply %s: %v", kind, reply, err))
		return false
	}
	return true
}

// names lists the buttons of b as the QA driver names them.
func (b Buttons) names() []string {
	var out []string
	for a, st := range answerStyle {
		if buttonStyle[b]&st != 0 {
			out = append(out, answerName[a])
		}
	}
	return out
}

// scriptedAnswer maps the reply of the test to a button of b; a cancelled box answers as closing
// its window does.
func scriptedAnswer(b Buttons, reply json.RawMessage, cancel bool) Answer {
	st := buttonStyle[b]
	if !cancel {
		var name string
		if replyAs("message", reply, &name) {
			for a, n := range answerName {
				if strings.EqualFold(n, name) && st&answerStyle[a] != 0 {
					return Answer(a)
				}
			}
			qaDialogFailed(fmt.Sprintf("the message dialog got the reply %s, its buttons are %s", reply, strings.Join(b.names(), ", ")))
		}
	}
	switch {
	case st&swt.CANCEL != 0:
		return AnswerCancel
	case st&swt.NO != 0:
		return AnswerNo
	}
	return AnswerOK
}

// MessageBox shows m modal over w and returns the pressed button.
func (w *Window) MessageBox(m Message) Answer {
	if r, cancel, ok := scripted(dialogCall{Kind: "message", Title: m.Title, Message: m.Text, Buttons: m.Buttons.names()}); ok {
		return scriptedAnswer(m.Buttons, r, cancel)
	}
	b := swt.NewMessageBoxParentStyle(w.shell, iconStyle[m.Icon]|buttonStyle[m.Buttons])
	b.SetText(m.Title)
	b.SetMessage(m.Text)
	return answerOf[b.Open()]
}

// Confirm asks a Yes/No question and reports whether Yes was pressed.
func (w *Window) Confirm(title, text string) bool {
	if r, cancel, ok := scripted(dialogCall{Kind: "confirm", Title: title, Message: text}); ok {
		yes := false
		if !cancel {
			replyAs("confirm", r, &yes)
		}
		return yes
	}
	return w.MessageBox(Message{title, text, IconQuestion, ButtonsYesNo}) == AnswerYes
}

// FileFilter is one entry of the file type chooser; Pattern is "*.png;*.jpg", "*" matches all.
type FileFilter struct{ Name, Pattern string }

// FileDialog describes an open or save dialog.
type FileDialog struct {
	Title, Dir, Name string
	Save, Multi      bool
	Filters          []FileFilter
}

func (d FileDialog) call() dialogCall {
	c := dialogCall{Kind: "file", Title: d.Title, Dir: d.Dir, Name: d.Name, Save: d.Save, Multi: d.Multi && !d.Save}
	for _, f := range d.Filters {
		c.Filters = append(c.Filters, f.Name+"|"+f.Pattern)
	}
	return c
}

// scriptedPaths takes a path or a list of paths from the reply; an empty list is a cancel.
func scriptedPaths(d FileDialog, reply json.RawMessage, cancel bool) []string {
	if cancel {
		return nil
	}
	var paths []string
	if json.Unmarshal(reply, &paths) != nil {
		var p string
		if !replyAs("file", reply, &p) {
			return nil
		}
		paths = []string{p}
	}
	if len(paths) > 1 && (d.Save || !d.Multi) {
		qaDialogFailed(fmt.Sprintf("the file dialog is not Multi, the reply has %d paths", len(paths)))
		return nil
	}
	return paths
}

// FileDialog shows d modal over w and returns the chosen paths, nil if cancelled. Multi only applies to open dialogs.
func (w *Window) FileDialog(d FileDialog) []string {
	if r, cancel, ok := scripted(d.call()); ok {
		return scriptedPaths(d, r, cancel)
	}
	style := int32(swt.OPEN)
	if d.Save {
		style = swt.SAVE
	} else if d.Multi {
		style |= swt.MULTI
	}
	dlg := swt.NewFileDialogParentStyle(w.shell, style)
	dlg.SetText(d.Title)
	dlg.SetFilterPath(d.Dir)
	dlg.SetFileName(d.Name)
	if len(d.Filters) > 0 {
		names, pats := make([]string, len(d.Filters)), make([]string, len(d.Filters))
		for i, f := range d.Filters {
			names[i], pats[i] = f.Name, f.Pattern
		}
		dlg.SetFilterNames(names)
		dlg.SetFilterExtensions(pats)
	}
	path := dlg.Open()
	if path == "" {
		return nil
	}
	if !d.Multi || d.Save {
		return []string{path}
	}
	var all []string
	for _, n := range dlg.GetFileNames() {
		all = append(all, filepath.Join(dlg.GetFilterPath(), n))
	}
	return all
}

// DirDialog shows a folder chooser; ok is false if cancelled.
func (w *Window) DirDialog(title, dir string) (path string, ok bool) {
	if r, cancel, handled := scripted(dialogCall{Kind: "dir", Title: title, Dir: dir}); handled {
		if !cancel {
			replyAs("dir", r, &path)
		}
		return path, path != ""
	}
	dlg := swt.NewDirectoryDialog(w.shell)
	dlg.SetText(title)
	dlg.SetFilterPath(dir)
	path = dlg.Open()
	return path, path != ""
}

// ColorDialog shows a color chooser starting at initial; ok is false if cancelled.
func (w *Window) ColorDialog(initial RGB) (c RGB, ok bool) {
	if r, cancel, handled := scripted(dialogCall{Kind: "color", Message: fmt.Sprintf("#%02x%02x%02x", initial.R, initial.G, initial.B)}); handled {
		var hex string
		if cancel || !replyAs("color", r, &hex) {
			return initial, false
		}
		digits := strings.TrimPrefix(hex, "#")
		v, err := strconv.ParseUint(digits, 16, 32)
		if err != nil || len(digits) != 6 {
			qaDialogFailed(fmt.Sprintf(`the color dialog got the reply %s, want "#rrggbb"`, r))
			return initial, false
		}
		return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, true
	}
	dlg := swt.NewColorDialog(w.shell)
	dlg.SetRGB(swt.NewRGB(int32(initial.R), int32(initial.G), int32(initial.B)))
	r := dlg.Open()
	if r == nil {
		return initial, false
	}
	return rgbOf(r), true
}

// Weight is a CSS-style font weight, 100 to 900. The zero value means unset: Bold decides.
type Weight int

const (
	Light    Weight = 300
	Regular  Weight = 400
	Medium   Weight = 500
	Semibold Weight = 600
	Bold     Weight = 700
)

// Font describes a typeface choice; Size is in points. Weight overrides Bold when set.
// SetFont keeps the control's current family for an empty Name, which for a default control is the
// system UI font (SF Pro, Segoe UI, the GTK UI font) at the given weight. Scale sizes the font
// relative to the system font when Size is 0: Font{Scale: 1.2, Weight: Semibold} is a heading that
// grows with the system font on every OS.
type Font struct {
	Name         string
	Size         int
	Scale        float64
	Weight       Weight
	Bold, Italic bool
}

// from fills an empty Name and a zero Size of f with those of cur, or the Size with the system font
// size of d times Scale. The result is the key under which equal fonts share one native font.
func (f Font) from(cur *swt.Font, d *swt.Display) Font {
	name, size := fontData(cur)
	_, sys := fontData(d.GetSystemFont())
	return f.resolve(name, size, sys)
}

func fontData(f *swt.Font) (name string, size int) {
	if fd := f.GetFontData(); len(fd) > 0 {
		return fd[0].GetName(), int(fd[0].GetHeight())
	}
	return "", 0
}

func (f Font) resolve(name string, size, system int) Font {
	if f.Name == "" {
		f.Name = name
	}
	if f.Size == 0 {
		f.Size = size
		if f.Scale != 0 {
			f.Size = max(1, int(math.Round(float64(system)*f.Scale)))
		}
	}
	f.Scale = 0
	if f.Weight != 0 {
		f.Bold = false
	}
	return f
}

// swtFont creates the native font for f on display d; the caller owns it.
func (f Font) swtFont(d swt.DeviceLike) *swt.Font {
	if f.Weight != 0 {
		return swt.NewFontDeviceNameHeightWeight(d, f.Name, int32(f.Size), int32(f.Weight), f.Italic)
	}
	var st int32
	if f.Bold {
		st |= swt.BOLD
	}
	if f.Italic {
		st |= swt.ITALIC
	}
	return swt.NewFontDeviceNameHeightStyle(d, f.Name, int32(f.Size), st)
}

// FontDialog shows a font chooser starting at initial; ok is false if cancelled. The dialog knows only
// regular and bold: an initial Weight of Semibold or more starts bold. The result has Weight 0 if
// initial.Weight was 0, else Bold or Regular, so a Medium or Semibold round trip is not preserved.
func (w *Window) FontDialog(initial Font) (f Font, ok bool) {
	if r, cancel, handled := scripted(dialogCall{Kind: "font", Message: initial.Name}); handled {
		if cancel || !replyAs("font", r, &f) {
			return initial, false
		}
		return f, true
	}
	dlg := swt.NewFontDialog(w.shell)
	if initial.Name != "" {
		fd := swt.NewFontData()
		fd.SetName(initial.Name)
		fd.SetHeight(int32(initial.Size))
		var st int32
		if initial.Bold || initial.Weight >= Semibold {
			st |= swt.BOLD
		}
		if initial.Italic {
			st |= swt.ITALIC
		}
		fd.SetStyle(st)
		dlg.SetFontData(fd)
	}
	r := dlg.Open()
	if r == nil {
		return initial, false
	}
	st := r.GetStyle()
	f = Font{Name: r.GetName(), Size: int(r.GetHeight()), Bold: st&swt.BOLD != 0, Italic: st&swt.ITALIC != 0}
	if initial.Weight != 0 {
		f.Weight = Regular
		if f.Bold {
			f.Weight = Bold
		}
	}
	return f, true
}
