package main

import (
	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/cocoa"
)

func dumpViews(w *gowt.Window) string { return cocoa.DumpViews(w.Unwrap().View, 4) }
