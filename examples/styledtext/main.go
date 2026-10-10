// Command styledtext highlights words in an editor. Offsets are runes, not bytes.
package main

import (
	"log"
	"strings"
	"unicode/utf8"

	"github.com/haiodo/gowt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("StyledText")
		w.SetLayout(gowt.Fill{})
		ed := w.StyledText(gowt.Scrollbars(), gowt.Border())
		ed.SetText("Styled text: ключевые слова func and return are colored per line.")

		blue := gowt.RGB{R: 30, G: 80, B: 200}
		// The callback gets one line; Start counts runes from its beginning.
		ed.OnStyle(func(line string) []gowt.StyleSpan {
			var spans []gowt.StyleSpan
			for _, kw := range []string{"func", "return"} {
				if i := strings.Index(line, kw); i >= 0 {
					spans = append(spans, gowt.StyleSpan{
						Start: utf8.RuneCountInString(line[:i]), Length: len(kw),
						Style: gowt.TextStyle{Foreground: &blue, Bold: true},
					})
				}
			}
			return spans
		})
		w.SetSize(480, 160)
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
