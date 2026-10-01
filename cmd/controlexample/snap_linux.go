package main

import (
	"fmt"
	"os"

	"github.com/haiodo/gowt/internal/shot"
	"github.com/haiodo/gowt/swt"
)

// snapAll runs the shared -snap driver; the window is read back from the X server through GDK.
func snapAll(display *swt.Display, shell *swt.Shell, folder *swt.TabFolder, dir string) {
	handle := shell.Handle
	snapRun(display, shell, folder, dir, snapHooks{
		meta: func(path string) {
			if err := os.WriteFile(path, []byte(shot.Meta(handle)), 0o644); err != nil {
				panic(err)
			}
		},
		// notify=true runs the same listeners as a click on the tab.
		selectTab: func(folder *swt.TabFolder, index int) { folder.SetSelectionIndexNotify(int32(index), true) },
		click: func(root *swt.Control, text string) {
			b := findButton(root, text)
			if b == nil {
				fmt.Println("button not found:", text)
				return
			}
			b.SetSelection(!b.GetSelection())
			b.NotifyListeners(swt.Selection, swt.NewEvent())
			fmt.Printf("clicked %s: selection=%v\n", text, b.GetSelection())
		},
		shot: func(path string) {
			if err := shot.WindowPNG(handle, path); err != nil {
				panic(err)
			}
			fmt.Println("snapshot", path)
		},
	})
}
