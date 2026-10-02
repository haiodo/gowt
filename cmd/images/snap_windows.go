package main

import (
	"fmt"

	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/swt"
)

// snapshot copies the only Shell's window from its window DC.
func snapshot(path string) {
	if err := shot.WindowPNG(swt.DisplayGetCurrent().GetShells()[0].Handle, path); err != nil {
		panic(err)
	}
	fmt.Println("snapshot", path)
}
