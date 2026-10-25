package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One sample has one docs image size on every OS: 2 px per point of its size in sizes.
// GOWT_SHOT_OS names the platforms; the default leaves out macos until it is re-shot (make shot-sizes checks all).
func TestShotSizes(t *testing.T) {
	oses := strings.Fields(os.Getenv("GOWT_SHOT_OS"))
	if len(oses) == 0 {
		oses = []string{"windows", "linux"}
	}
	for _, s := range samples {
		size, ok := sizes[s.name]
		if !ok {
			size = defaultSize
		}
		for _, o := range oses {
			path := filepath.Join("..", "..", "docs", "img", fmt.Sprintf("%s-%s.png", s.name, o))
			f, err := os.Open(path)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			cfg, err := png.DecodeConfig(f)
			f.Close()
			if err != nil {
				t.Errorf("%s: %v", path, err)
			} else if cfg.Width != 2*size[0] || cfg.Height != 2*size[1] {
				t.Errorf("%s is %dx%d, want %dx%d", path, cfg.Width, cfg.Height, 2*size[0], 2*size[1])
			}
		}
	}
}
