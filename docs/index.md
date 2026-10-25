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
		w := app.Window("Hello",
			gowt.Label("Hello, gowt"),
			gowt.Button("Quit", gowt.OnClick(app.Quit)))
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

This is [`examples/hello`](../examples/hello). Run it with `go run ./examples/hello` from a checkout of the repository.

Four ideas carry the whole API:

- `gowt.Run` owns the UI thread and the event loop. You build windows inside the setup function, and `Run` returns when the last window closes or `app.Quit()` is called. It returns an error raised while setup builds the window; errors in callbacks go to `app.OnError` and the app keeps running.
- Constructors return descriptions: `gowt.Button("Quit", gowt.OnClick(quit))`, `gowt.Label("text")`. A description becomes a widget when it is passed to its parent: to `app.Window(title, children...)`, to a container such as `gowt.VBox(children...)`, or to `panel.Add(children...)`. Data shared between widgets lives in `rx` state that the widgets bind to; a widget you need for imperative work is captured with `gowt.Ref(&x)`.
- Settings that must be known at creation (a check box, a password field, a border) are options: `gowt.Button("Remember", gowt.Check())`. So are events and initial values: `gowt.OnClick`, `gowt.OnChange`, `gowt.Checked`, `gowt.Value`.
- Callbacks are plain functions given as options: `gowt.OnClick(func())`, `gowt.OnChange(func(text string))`. Touch widgets only on the UI thread; [events.md](events.md) shows how to come back from a goroutine.

## Guide

| Topic | Page | Example |
|---|---|---|
| Installation, per OS | [install.md](install.md) | - |
| Windows and layouts | [windows-layouts.md](windows-layouts.md) | [`examples/layouts`](../examples/layouts) |
| Widgets | [widgets.md](widgets.md) | [`examples/widgets`](../examples/widgets) |
| Events, the UI thread and reactive state | [events.md](events.md) | [`examples/events`](../examples/events), [`examples/rows`](../examples/rows) |
| Dialogs and menus | [dialogs-menus.md](dialogs-menus.md) | [`examples/dialogs`](../examples/dialogs) |
| Graphics and images | [graphics.md](graphics.md) | [`examples/graphics`](../examples/graphics) |
| Theme and platform look | [theme-look.md](theme-look.md) | [`examples/theme`](../examples/theme) |
| One layout for three systems: type and spacing tokens, density | [cross-platform-look.md](cross-platform-look.md) | - |
| Icons | [icons.md](icons.md) | [`examples/icons`](../examples/icons) |
| Web view | [webview-browser.md](webview-browser.md) | [`examples/webview`](../examples/webview) |
| Widget catalog (all widgets, one line each) | [widget-catalog.md](widget-catalog.md) | [`examples/catalog`](../examples/catalog) |
| Packaging and distribution | [packaging.md](packaging.md) | - |
| Platform notes and known gaps | [platforms.md](platforms.md) | - |
| Testing an app with the QA driver | [qa.md](qa.md) | [`examples/qa`](../examples/qa) |

## SWT port

Packages translated from Eclipse with their Java names, for porting SWT and JFace code or for what the facade does not wrap. Apps and the examples above do not use them: a new screen is written with `gowt`, and `webview` for a web page.

| Topic | Page | Example |
|---|---|---|
| Using swt directly | [swt-direct.md](swt-direct.md) | [`examples/swtdirect`](../examples/swtdirect) |
| JFace factories (`jface`) | [jface.md](jface.md) | [`examples/jface`](../examples/jface) |
| SWT `Browser` (`browser`) | [webview-browser.md](webview-browser.md#browser) | [`examples/browser`](../examples/browser) |

Reference material:

- [agents.md](agents.md): the canonical idioms on one page and which option works on which widget; the page to give a coding agent.
- [screenshots.md](screenshots.md): how `docs/img` is produced (`GOWT_SNAP`, `make docs-shots`).
- Package docs: `go doc github.com/haiodo/gowt` (every exported name has a doc comment).
- [facade.md](facade.md): why the API looks the way it does.
- [CONTRIBUTING.md](../CONTRIBUTING.md): build, tests, generated code, DCO sign-off, commit style.
- [field-check.md](field-check.md): run the checks on your machine with one command and attach the result to an issue.
- [internals.md](internals.md): repository map, the Java-to-Go translator, how the ports are built. Only needed to work on gowt itself.

## Not in the facade yet

Clipboard, drag and drop and `StyledText` are in the facade (`App.Clipboard`, `DragFrom`/`DropOn`, `StyledText`). Not verified: drag and drop with a real mouse on any OS; cross-application clipboard transfer (the peer-process SWT tests are skipped); `StyledText` style rendering on macOS (upstream tests skipped). `Accessible` is a stub on all three OSes and has no facade wrapper.

## Building the examples

`make examples-check` runs `go vet` on every example for darwin, windows and linux (no window is opened). The examples were type-checked this way; they were not all run on every OS.
