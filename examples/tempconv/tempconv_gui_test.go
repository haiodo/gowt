package main

import (
	"os"
	"testing"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/internal/qatree"
)

var mainq = make(chan func())

func TestMain(m *testing.M) {
	done := make(chan int)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainq:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

func TestTemperatureConversion(t *testing.T) {
	if os.Getenv("GOWT_GUI_TEST") == "" {
		t.Skip("set GOWT_GUI_TEST=1")
	}
	done := make(chan struct{})
	mainq <- func() {
		defer close(done)
		err := gowt.Run(func(app *gowt.App) {
			w := app.Window("Temperature", tempConv())
			w.Show()
			app.Async(func() {
				defer w.Close()
				steps := []struct{ from, input, want string }{
					{"0", "100", "212"},
					{"212", "32", "0"},
					{"0", "001.50", "34.7"},
					{"001.50", "invalid", "34.7"},
				}
				for _, step := range steps {
					sel := qatree.Selector{Class: "Text", Text: step.from}
					if err := qatree.Act(app.Unwrap(), sel, "type", []byte(`{"text":"`+step.input+`"}`)); err != nil {
						t.Error(err)
						return
					}
					source, err := qatree.Find(app.Unwrap(), qatree.Selector{Class: "Text", Text: step.input})
					if err != nil || len(source) != 1 {
						t.Errorf("source %q: found %d, error %v", step.input, len(source), err)
					}
					matches, err := qatree.Find(app.Unwrap(), qatree.Selector{Class: "Text", Text: step.want})
					if err != nil || len(matches) != 1 {
						t.Errorf("input %q: wanted %q, found %d, error %v", step.input, step.want, len(matches), err)
					}
				}
			})
		})
		if err != nil {
			t.Error(err)
		}
	}
	<-done
}
