package main

import (
	"fmt"

	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/swt"
)

// snapshot renders the only Shell's window through PrintWindow.
func snapshot(path string) {
	if err := shot.WindowPNG(swt.DisplayGetCurrent().GetShells()[0].Handle, path); err != nil {
		panic(err)
	}
	fmt.Println("snapshot", path)
}
