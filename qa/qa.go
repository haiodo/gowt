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
