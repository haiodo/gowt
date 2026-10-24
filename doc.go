/*
Package gowt builds native desktop windows in Go without cgo. The windows, buttons,
tables and menus on screen are the operating system's own widgets: AppKit on macOS,
Win32 on Windows, GTK 3 on Linux. The package is a small idiomatic layer over a Go
translation of Eclipse SWT; no C compiler and no Java are needed, and the OS libraries
are loaded at run time.

# Hello

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

# Concepts

  - [Run] owns the UI thread and the event loop. The setup function builds the windows;
    Run returns when the last window closes or [App.Quit] is called, with the error
    raised in setup. Errors in callbacks go to [App.OnError].
  - Constructors such as [Label], [Button] and [Table] return an [Option], a description.
    A description becomes a widget when it is passed to its parent: [App.Window],
    a container, or [Panel.Add]. The nesting of the calls is the nesting on screen.
  - Containers are [VBox], [HBox], [Grid], [Flow], [Fill], [Form], [Pages], [Tabs],
    [Split] and [Scroll]. Placement is options: [GrowX], [Span], [Size], [Gap], [Padding].
  - Everything else about a widget is an option too: events ([OnClick], [OnChange]),
    initial values ([Value], [Checked]) and bindings to state ([Bind], [BindText]).
  - State shared between widgets lives in package rx (rx.New, rx.Compute); widgets
    bind to it and callbacks Set it.
  - Touch widgets only on the UI thread. [App.Async] and rx State.Post come back from
    a goroutine.
  - Unwrap on any widget returns the underlying swt object for what the facade does
    not wrap.

# Documentation

The guide, one page and one runnable program per topic, starts at
https://github.com/haiodo/gowt/blob/main/docs/index.md. The programs are in
https://github.com/haiodo/gowt/tree/main/examples. The idioms on one page, for people
and coding agents: https://github.com/haiodo/gowt/blob/main/docs/agents.md.

Related packages: look (window material, Liquid Glass, Mica, dark content), icons
(system icons by name), rx (reactive state), webview (the system web view next to native
widgets), qa and qa/qatest (driving an app from go test).

# Platforms and requirements

  - macOS 13 or later.
  - Windows 10 or later (the minimum is not verified); the manifest for visual styles and
    DPI awareness is linked by this package. The webview package needs the WebView2 Runtime.
  - Linux: glibc (musl distributions such as Alpine are not supported), GTK 3 and an X11
    or XWayland display; WebKitGTK 4.1 for webview.
  - Go 1.27 or newer. Cross-compiling works from any host: GOOS=windows go build.

# The SWT port

Packages swt, jface and browser are the low-level port of SWT, JFace and SWT's Browser
with their Java names, translated by a generator. They are for porting SWT code and for
what this package does not wrap; start new programs here, not there.
*/
package gowt
