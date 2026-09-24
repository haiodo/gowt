// Command tree shows a Tree (NSOutlineView) with three root items and their children.
package main

import (
	"fmt"
	"runtime"

	"github.com/haiodo/gowt/swt"
)

// AppKit must run on the process's main thread.
func init() { runtime.LockOSThread() }

func main() {
	display := swt.NewDisplay()
	shell := swt.NewShellDisplay(display)
	shell.SetText("Tree")
	shell.SetLayout(&swt.NewFillLayout().Layout)
	tree := swt.NewTree(shell, swt.BORDER)

	// Event.Item is the *Widget inside a TreeItem; map it back to reach TreeItem's methods.
	items := map[*swt.Widget]*swt.TreeItem{}
	children := map[string][]string{
		"Fruits":     {"Apple", "Banana", "Cherry"},
		"Vegetables": {"Carrot", "Potato"},
		"Grains":     {"Rice", "Wheat", "Oats"},
	}
	var roots []*swt.TreeItem
	for _, name := range []string{"Fruits", "Vegetables", "Grains"} {
		root := swt.NewTreeItem(tree, swt.NONE)
		root.SetText(name)
		items[root.AsWidget()] = root
		roots = append(roots, root)
		for _, c := range children[name] {
			child := swt.NewTreeItemParentItemStyle(root, swt.NONE)
			child.SetText(c)
			items[child.AsWidget()] = child
		}
	}
	roots[0].SetExpanded(true)

	tree.AddSelectionListener(swt.SelectionListenerWidgetSelectedAdapter(func(e *swt.SelectionEvent) {
		fmt.Println("Selected:", items[e.Item].GetText())
	}))
	tree.AddTreeListener(swt.TreeListenerTreeExpandedAdapter(func(e *swt.TreeEvent) {
		fmt.Println("Expanded:", items[e.Item].GetText())
	}))

	shell.SetSize(260, 300)
	shell.Open()
	for !shell.IsDisposed() {
		if !display.ReadAndDispatch() {
			display.Sleep()
		}
	}
	display.Dispose()
}
