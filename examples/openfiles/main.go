// Command openfiles lists the files the system asks it to open: Finder, Explorer or a file manager
// (double click, Open With), command-line arguments, and a second launch handed to the running one.
//
// OPENFILES_ID=org.example.OpenFiles makes it a single instance (default: not). OPENFILES_LOG=<file>
// also appends every path to a file. OPENFILES_EXIT_MS=<ms> quits after that time.
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/haiodo/gowt"
)

func main() {
	if id := os.Getenv("OPENFILES_ID"); id != "" {
		first, err := gowt.SingleInstance(id)
		if err != nil {
			log.Println(err) // not fatal: the program runs as an extra instance
		}
		if !first {
			return // the running instance got our files
		}
	}
	err := gowt.Run(func(app *gowt.App) {
		var list *gowt.TextView
		w := app.Window("Open files")
		w.Add(
			gowt.Label("Double click a file of an associated type, or run: openfiles file..."),
			gowt.Text(gowt.Multiline(), gowt.ReadOnly(), gowt.Grow(), gowt.Ref(&list)))
		w.SetSize(560, 300)
		var all []string
		app.OnOpenFiles(func(paths []string) {
			all = append(all, paths...)
			list.SetText(strings.Join(all, "\n"))
			log.Println("open:", paths)
			if p := os.Getenv("OPENFILES_LOG"); p != "" {
				if f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
					fmt.Fprintln(f, strings.Join(paths, "\n"))
					f.Close()
				}
			}
		})
		if ms, _ := strconv.Atoi(os.Getenv("OPENFILES_EXIT_MS")); ms > 0 {
			app.After(time.Duration(ms)*time.Millisecond, app.Quit)
		}
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
