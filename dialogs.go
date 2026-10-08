package gowt

import (
	"path/filepath"

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
)

// MessageBox shows m modal over w and returns the pressed button.
func (w *Window) MessageBox(m Message) Answer {
	b := swt.NewMessageBoxParentStyle(w.shell, iconStyle[m.Icon]|buttonStyle[m.Buttons])
	b.SetText(m.Title)
	b.SetMessage(m.Text)
	return answerOf[b.Open()]
}

// Confirm asks a Yes/No question and reports whether Yes was pressed.
func (w *Window) Confirm(title, text string) bool {
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

// FileDialog shows d modal over w and returns the chosen paths, nil if cancelled. Multi only applies to open dialogs.
func (w *Window) FileDialog(d FileDialog) []string {
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
	dlg := swt.NewDirectoryDialog(w.shell)
	dlg.SetText(title)
	dlg.SetFilterPath(dir)
	path = dlg.Open()
	return path, path != ""
}

// ColorDialog shows a color chooser starting at initial; ok is false if cancelled.
func (w *Window) ColorDialog(initial RGB) (c RGB, ok bool) {
	dlg := swt.NewColorDialog(w.shell)
	dlg.SetRGB(swt.NewRGB(int32(initial.R), int32(initial.G), int32(initial.B)))
	r := dlg.Open()
	if r == nil {
		return initial, false
	}
	return rgbOf(r), true
}

// Font describes a typeface choice; Size is in points.
type Font struct {
	Name         string
	Size         int
	Bold, Italic bool
}

// FontDialog shows a font chooser starting at initial; ok is false if cancelled.
func (w *Window) FontDialog(initial Font) (f Font, ok bool) {
	dlg := swt.NewFontDialog(w.shell)
	if initial.Name != "" {
		fd := swt.NewFontData()
		fd.SetName(initial.Name)
		fd.SetHeight(int32(initial.Size))
		var st int32
		if initial.Bold {
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
	return Font{r.GetName(), int(r.GetHeight()), st&swt.BOLD != 0, st&swt.ITALIC != 0}, true
}
