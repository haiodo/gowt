package gowt

import "github.com/haiodo/gowt/swt"

// Fit makes a Table or Tree column as wide as its header and its widest cell, again after the rows
// change (Row, Remove, Clear, Rows, Nodes, expand, collapse), once per event-loop turn. The width given
// to Column is only the first one. Every row is measured: for big data use Column.Pack once instead.
func Fit() Option { return style{id: sFit} }

type fitState struct {
	cols    []*Column
	pending bool
}

var fits = map[*swt.Control]*fitState{}

// addFit registers c to be fitted with the content of owner.
func addFit(owner *swt.Control, c *Column) {
	f := fits[owner]
	if f == nil {
		f = &fitState{}
		fits[owner] = f
		owner.AddDisposeListener(disposeFunc(func() { delete(fits, owner) }))
	}
	f.cols = append(f.cols, c)
	refitColumns(owner)
}

// refitColumns packs the Fit columns of owner on the next event-loop turn, once for all changes before it.
func refitColumns(owner *swt.Control) {
	f := fits[owner]
	if f == nil || f.pending {
		return
	}
	a := appOf(owner)
	if a == nil {
		return
	}
	f.pending = true
	a.Async(func() {
		f.pending = false
		if owner.IsDisposed() {
			return
		}
		for _, c := range f.cols {
			packColumn(c.c)
		}
	})
}

// FitColumns resizes every column to its header and widest cell once, whether or not it has Fit.
func (t *TableView) FitColumns() {
	for _, c := range t.t.GetColumns() {
		packColumn(c)
	}
}

// FitColumns resizes every column to its header and widest cell once, whether or not it has Fit.
func (t *TreeView) FitColumns() {
	for _, c := range t.t.GetColumns() {
		packColumn(c)
	}
}

func hasStyle(opts []Option, id styleID) bool {
	for _, o := range opts {
		if s, ok := o.(style); ok && s.id == id {
			return true
		}
	}
	return false
}
