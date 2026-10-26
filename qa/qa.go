// Package qa is the QA hook of a gowt app. A build with -tags gowtqa and GOWT_QA_SOCKET in the
// environment makes gowt.Run listen on that socket; package qa/qatest drives the app from go test.
// Without the tag the app has no server and no network code: Handle does nothing.
package qa

import (
	"encoding/json"
	"fmt"
)

// Environment of a QA run, set by qatest.
const (
	EnvSocket = "GOWT_QA_SOCKET" // unix socket path; a QA build listens only when it is set
	EnvData   = "GOWT_QA_DATA"   // a private data directory for the app to use instead of the user's
)

// Environment of the test process, read by qatest.
const (
	EnvArtifacts = "GOWT_QA_ARTIFACTS" // directory for the shots, dump and logs of failed tests
	EnvKeep      = "GOWT_QA_KEEP"      // set: keep the app's socket, data and log directory after the run
)

// Args are the key-value arguments a test passes to a handler.
type Args struct{ m map[string]json.RawMessage }

// NewArgs wraps the raw JSON object a request carried; an empty or null raw is no arguments.
func NewArgs(raw json.RawMessage) (Args, error) {
	var a Args
	if len(raw) == 0 || string(raw) == "null" {
		return a, nil
	}
	if err := json.Unmarshal(raw, &a.m); err != nil {
		return a, fmt.Errorf("arguments are not a JSON object: %w", err)
	}
	return a, nil
}

// Has reports whether the test passed key.
func (a Args) Has(key string) bool { _, ok := a.m[key]; return ok }

// String, Int and Bool give the zero value when key is absent or of another type.
func (a Args) String(key string) string { var v string; a.Decode(key, &v); return v }

func (a Args) Int(key string) int { var v int; a.Decode(key, &v); return v }

func (a Args) Bool(key string) bool { var v bool; a.Decode(key, &v); return v }

// Decode reads the value of key into v; an absent key leaves v alone and is no error.
func (a Args) Decode(key string, v any) error {
	raw, ok := a.m[key]
	if !ok {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// Selector picks controls; it mirrors internal/qatree.Selector, which the client cannot import
// without linking the toolkit. An empty field matches anything.
type Selector struct {
	ID     string    `json:"id,omitempty"`
	Class  string    `json:"class,omitempty"`
	Text   string    `json:"text,omitempty"`
	Within *Selector `json:"within,omitempty"`
}

// ID picks the control created with gowt.ID(id).
func ID(id string) Selector { return Selector{ID: id} }

// Class picks controls of an swt class: "Button", "Table", "TableItem", ...
func Class(class string) Selector { return Selector{Class: class} }

// Text picks controls showing text.
func Text(text string) Selector { return Selector{Text: text} }

// In narrows s to the subtree of the one control parent picks.
func (s Selector) In(parent Selector) Selector { s.Within = &parent; return s }

// Node is one control or item in a dump; it mirrors internal/qatree.Node.
type Node struct {
	ID       string `json:"id,omitempty"`
	Class    string `json:"class"`
	Text     string `json:"text,omitempty"`
	Enabled  bool   `json:"enabled"`
	Visible  bool   `json:"visible"`
	Focused  bool   `json:"focused,omitempty"`
	Bounds   [4]int `json:"bounds"` // x, y, width, height in the coordinates of its window
	Checked  *bool  `json:"checked,omitempty"`
	Selected []int  `json:"selected,omitempty"`
	Value    string `json:"value,omitempty"`
	Path     string `json:"path"`
	Children []Node `json:"children,omitempty"`
}

// ScrollEvent is one scroll event for Element.ScrollEvent, delivered to a gfx view's input router.
// X and Y are the point in the control's coordinates. Without Precise the deltas are mouse wheel lines;
// with it, points as a trackpad reports them. Phase and Momentum are "", "mayBegin", "began",
// "changed", "ended" or "cancelled"; Momentum is the phase of the coasting after the fingers left.
// Mods is "Shift", "Mod+Alt" and so on, as for Key without the key; Mod with a wheel zooms.
type ScrollEvent struct {
	X, Y            float64
	DX, DY          float64
	Precise         bool
	Phase, Momentum string
	Mods            string
}

// PinchEvent is one pinch step for Element.Pinch. Magnification is the change of the zoom: 0.1 makes
// it 10 percent larger, -0.1 smaller. X and Y are the point the zoom keeps in place.
type PinchEvent struct {
	X, Y          float64
	Magnification float64
	Phase         string // "", "began", "changed", "ended" or "cancelled"
	Mods          string
}

// PointerEvent is a mouse event for Element.Pointer, ClickAt, DoubleClickAt, RightClickAt and DragTo. X
// and Y are the point in the control's coordinates. A gfx view gets it through its input router, as
// the native events arrive; any other control gets the MouseDown, MouseUp, MouseMove (and
// MouseDoubleClick) events of its listeners, with no operating system event involved.
//
// Kind is "down", "move" or "up" for one event, or "click" (down, up), "doubleclick" (down, up, down
// with Count 2, up) and "drag" (down, Steps moves on the way to ToX, ToY, up) for a sequence. Button is
// SWT's numbering, 1 left (the default), 2 middle, 3 right. Count is the click count of a down, 1 by
// default. Mods is "Shift", "Mod+Alt" and so on, as for Key without the key. A move needs no Button:
// the view reports the button held.
type PointerEvent struct {
	Kind     string
	X, Y     float64
	Button   int
	Count    int
	Mods     string
	ToX, ToY float64 // drag: where the button is released
	Steps    int     // drag: moves before the release, 4 by default
}

// Job is one running Job or Track of the app, as the "jobs" command reports it; it mirrors gowt.JobInfo.
type Job struct {
	Name       string `json:"name"`
	Background bool   `json:"background,omitempty"`
	Track      bool   `json:"track,omitempty"`
	ElapsedMS  int64  `json:"elapsed_ms"`
	Done       int    `json:"done,omitempty"`
	Total      int    `json:"total,omitempty"`
	Text       string `json:"text,omitempty"`
}

// Dialog is a scripted answer for qatest's Answer, or a dialog the app asked for as Dialogs reports it.
// Kind is "message" (Window.MessageBox), "confirm" (Window.Confirm), "file" (FileDialog), "dir"
// (DirDialog), "color" (ColorDialog) or "font" (FontDialog).
//
// Reply by kind: a button name ("OK", "Cancel", "Yes", "No", "Retry", "Abort", "Ignore") for message;
// a bool for confirm; a path, or a list of paths for a Multi dialog, for file; a path for dir; "#rrggbb"
// for color; a Font object ({"Name": "Menlo", "Size": 12, "Bold": true}) for font. Cancel answers as
// the user who dismissed the dialog and needs no Reply.
//
// In what Dialogs returns, Reply and Cancel are what the app received, and Unanswered marks a dialog
// with no scripted answer, which got Cancel.
type Dialog struct {
	Kind       string   `json:"kind"`
	Reply      any      `json:"reply,omitempty"`
	Cancel     bool     `json:"cancel,omitempty"`
	Title      string   `json:"title,omitempty"`
	Message    string   `json:"message,omitempty"`
	Buttons    []string `json:"buttons,omitempty"` // message: the buttons it offers
	Dir        string   `json:"dir,omitempty"`     // file, dir: where it opens
	Name       string   `json:"name,omitempty"`    // file: the proposed file name
	Save       bool     `json:"save,omitempty"`
	Multi      bool     `json:"multi,omitempty"`
	Filters    []string `json:"filters,omitempty"` // file: "Name|pattern"
	Unanswered bool     `json:"unanswered,omitempty"`
}
