# gowt documentation

gowt is a native GUI toolkit for Go without cgo. The windows, buttons, tables and menus on screen are the operating system's own (AppKit on macOS, Win32 on Windows, GTK 3 on Linux). The Go API is a thin layer over Eclipse SWT, translated from Java to Go; you do not need to know SWT or Java to use it.

This documentation covers the `gowt` facade package. Every page has a runnable program in [`examples/`](../examples).

## Getting started

```sh
go get github.com/haiodo/gowt@latest
```

Go 1.27 or newer, no C compiler. Per-OS requirements are in [install.md](install.md).

```go
package main

import (
	"log"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Hello")
		w.SetLayout(gowt.Row{Vertical: true, Margin: 12, Spacing: 8})
		w.Label("Hello, gowt")
		w.Button("Quit", app.Quit)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

This is [`examples/hello`](../examples/hello). Run it with `go run ./examples/hello` from a checkout of the repository.

Four ideas carry the whole API:

- `gowt.Run` owns the UI thread and the event loop. You build windows inside the setup function, and `Run` returns when the last window closes or `app.Quit()` is called. It also returns errors raised by the toolkit.
- A parent creates its children: `w.Button("Quit", onClick)`, `panel.Label("text")`. There is no `Parent` argument and no separate "add" step.
- Settings that must be known at creation (a check box, a password field, a border) are options: `w.Button("Remember", nil, gowt.Check())`.
- Callbacks are plain functions: `OnClick(func())`, `OnChange(func(text string))`. Touch widgets only on the UI thread; [events.md](events.md) shows how to come back from a goroutine.

## Guide

| Topic | Page | Example |
|---|---|---|
| Installation, per OS | [install.md](install.md) | - |
| Windows and layouts | [windows-layouts.md](windows-layouts.md) | [`examples/layouts`](../examples/layouts) |
| Widgets | [widgets.md](widgets.md) | [`examples/widgets`](../examples/widgets) |
| Events and the UI thread | [events.md](events.md) | [`examples/events`](../examples/events) |
| Dialogs and menus | [dialogs-menus.md](dialogs-menus.md) | [`examples/dialogs`](../examples/dialogs) |
| Graphics and images | [graphics.md](graphics.md) | [`examples/graphics`](../examples/graphics) |
| Theme and platform look | [theme-look.md](theme-look.md) | [`examples/theme`](../examples/theme) |
| Icons | [icons.md](icons.md) | [`examples/icons`](../examples/icons) |
| Web view and Browser | [webview-browser.md](webview-browser.md) | [`examples/webview`](../examples/webview), [`examples/browser`](../examples/browser) |
| Using swt directly | [swt-direct.md](swt-direct.md) | [`examples/swtdirect`](../examples/swtdirect) |
| Platform notes and known gaps | [platforms.md](platforms.md) | - |

Reference material:

- Package docs: `go doc github.com/haiodo/gowt` (every exported name has a doc comment).
- [facade.md](facade.md): why the API looks the way it does.
- [internals.md](internals.md): repository map, the Java-to-Go translator, how the ports are built. Only needed to work on gowt itself.

## Not in the facade yet

Clipboard, drag and drop, `StyledText` and `Accessible` are being integrated and are reachable only through package `swt` for now. Do not rely on them from `gowt` yet.

## Building the examples

`make examples-check` runs `go vet` on every example for darwin, windows and linux (no window is opened). The examples were type-checked this way; they were not all run on every OS.
