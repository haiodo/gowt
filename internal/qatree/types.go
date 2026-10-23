// Package qatree walks the controls of a running gowt app for the QA driver: a dump of the windows,
// selectors and semantic actions. Every function runs on the UI thread.
package qatree

// Node is one control or item in a dump.
type Node struct {
	ID       string `json:"id,omitempty"` // gowt.ID
	Class    string `json:"class"`        // swt class: Shell, Button, Text, Table, TableItem, ...
	Text     string `json:"text,omitempty"`
	Enabled  bool   `json:"enabled"`
	Visible  bool   `json:"visible"`
	Focused  bool   `json:"focused,omitempty"`
	Bounds   [4]int `json:"bounds"` // x, y, width, height in the coordinates of its window
	Checked  *bool  `json:"checked,omitempty"`
	Selected []int  `json:"selected,omitempty"`
	Value    string `json:"value,omitempty"`
	Path     string `json:"path"` // position in the tree, stable within one dump
	Children []Node `json:"children,omitempty"`
}

// Selector picks controls. An empty field matches anything; Within narrows the search to the
// subtree of the one control it picks.
type Selector struct {
	ID     string    `json:"id,omitempty"`
	Class  string    `json:"class,omitempty"`
	Text   string    `json:"text,omitempty"`
	Within *Selector `json:"within,omitempty"`
}
