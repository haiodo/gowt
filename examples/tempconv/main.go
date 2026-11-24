// Command tempconv is the 7GUIs temperature converter task: two fields convert into each other.
package main

import (
	"log"
	"strconv"

	"github.com/haiodo/gowt"
)

func tempConv() gowt.Option {
	var celsius, fahrenheit *gowt.TextView

	setFahrenheit := func(s string) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		fahrenheit.SetText(strconv.FormatFloat(v*9/5+32, 'f', -1, 64))
	}
	setCelsius := func(s string) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		celsius.SetText(strconv.FormatFloat((v-32)*5/9, 'f', -1, 64))
	}
	return gowt.Fields(gowt.GrowX(),
		gowt.Field("Celsius",
			gowt.Text(gowt.Value("0"), gowt.Size(140, 0), gowt.GrowX(), gowt.Ref(&celsius), gowt.OnEdit(setFahrenheit))),
		gowt.Field("Fahrenheit",
			gowt.Text(gowt.Value("32"), gowt.Size(140, 0), gowt.GrowX(), gowt.Ref(&fahrenheit), gowt.OnEdit(setCelsius))))
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("Temperature", tempConv())
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
