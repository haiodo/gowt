// Command booker is the 7GUIs flight booker task: one-way or return dates are checked before booking.
package main

import (
	"log"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

const (
	oneWay = iota
	returnFlight
)

var flightKinds = []string{"one-way flight", "return flight"}

func parseDate(s string) (time.Time, error) {
	return time.Parse("02.01.2006", s)
}

func bookable(mode int, first, second time.Time) bool {
	return mode != returnFlight || !first.After(second)
}

func booker() gowt.Option {
	mode := rx.New(oneWay)
	first := rx.New("27.03.2014")
	second := rx.New("27.03.2014")
	message := rx.New("")

	firstValid := rx.Compute(func() bool {
		_, err := parseDate(first.Get())
		return err == nil
	})
	secondValid := rx.Compute(func() bool {
		if mode.Get() == oneWay {
			return true
		}
		_, err := parseDate(second.Get())
		return err == nil
	})
	ready := rx.Compute(func() bool {
		a, err := parseDate(first.Get())
		if err != nil {
			return false
		}
		if mode.Get() == oneWay {
			return true
		}
		b, err := parseDate(second.Get())
		return err == nil && bookable(mode.Get(), a, b)
	})

	invalid := gowt.RGB{R: 255, G: 200, B: 200}
	markInvalid := func(t *gowt.TextView, valid *rx.Derived[bool]) {
		gowt.Watch(t, valid, func(ok bool) {
			if ok {
				t.SetBackground(nil)
			} else {
				t.SetBackground(invalid)
			}
		})
	}
	return gowt.VBox(
		gowt.Combo(flightKinds, gowt.ReadOnly(), gowt.Selected(oneWay), gowt.Size(180, 0), gowt.GrowX(),
			gowt.Bind(mode)),
		gowt.Text(gowt.Size(180, 0), gowt.GrowX(), gowt.Bind(first), gowt.Init(func(t *gowt.TextView) {
			markInvalid(t, firstValid)
		})),
		gowt.Text(gowt.Size(180, 0), gowt.GrowX(), gowt.Bind(second),
			gowt.BindEnabled(rx.Compute(func() bool { return mode.Get() == returnFlight })),
			gowt.Init(func(t *gowt.TextView) { markInvalid(t, secondValid) })),
		gowt.Button("Book", gowt.BindEnabled(ready), gowt.OnClick(func() {
			a, err := parseDate(first.Get())
			if err != nil {
				return
			}
			kind := flightKinds[mode.Get()]
			if mode.Get() == oneWay {
				message.Set("You have booked a " + kind + " on " + a.Format("02.01.2006") + ".")
				return
			}
			b, err := parseDate(second.Get())
			if err != nil {
				return
			}
			message.Set("You have booked a " + kind + " on " + a.Format("02.01.2006") +
				" and " + b.Format("02.01.2006") + ".")
		})),
		gowt.Label("", gowt.BindText(message)),
	)
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Flight booker", booker())
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
