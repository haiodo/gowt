package kit

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every docs/img/app-<name>-<os>.png is exactly 2 x the snap size, so an app's shots compare.
// GOWT_SHOT_OS lists the OSes to check (make shot-sizes: all three); the default has no macOS.
func TestAppShotSizes(t *testing.T) {
	oses := strings.Fields(os.Getenv("GOWT_SHOT_OS"))
	if len(oses) == 0 {
		oses = []string{"windows", "linux"}
	}
	dirs, err := os.ReadDir(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if !d.IsDir() || d.Name() == "internal" {
			continue
		}
		for _, o := range oses {
			path := filepath.Join("..", "..", "..", "..", "docs", "img", fmt.Sprintf("app-%s-%s.png", d.Name(), o))
			f, err := os.Open(path)
			if err != nil {
				t.Errorf("%s: %v", path, err)
				continue
			}
			cfg, err := png.DecodeConfig(f)
			f.Close()
			if err != nil {
				t.Errorf("%s: %v", path, err)
			} else if cfg.Width != 2*SnapWidth || cfg.Height != 2*SnapHeight {
				t.Errorf("%s is %dx%d, want %dx%d", path, cfg.Width, cfg.Height, 2*SnapWidth, 2*SnapHeight)
			}
		}
	}
}
