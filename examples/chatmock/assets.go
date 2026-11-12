package main

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/haiodo/gowt"
)

//go:embed assets/*.png
var assetFS embed.FS

// photoCache keeps the decoded avatar photos; one entry per person.
var photoCache = map[string]*gowt.Image{}

// photo returns the avatar photo of p as a round image, size points wide, or nil when the
// person has no photo asset.
func photo(app *gowt.App, p person, size int) *gowt.Image {
	if p.photo == "" {
		return nil
	}
	img, ok := photoCache[p.photo]
	if !ok {
		b, err := assetFS.ReadFile("assets/" + p.photo)
		if err != nil {
			panic(fmt.Sprintf("chatmock: avatar asset %q: %v", p.photo, err))
		}
		img, err = app.ImageFrom(bytes.NewReader(b))
		if err != nil {
			panic(fmt.Sprintf("chatmock: avatar asset %q: %v", p.photo, err))
		}
		photoCache[p.photo] = img
	}
	return app.CircleImage(img, size)
}
