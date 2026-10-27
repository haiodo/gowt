// Package demovideo records the frames of the README video: the QA build of examples/apps/monitor
// is shot window-only as fast as the app answers, so no desktop shows. tooling/demo-video.sh drives it.
package demovideo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/haiodo/gowt/qa/qatest"
)

func TestMain(m *testing.M) {
	bin := os.Getenv("GOWT_DEMO_BIN")
	if bin == "" {
		bin = "unset"
	}
	qatest.Main(m, bin)
}

func envSeconds(name string, def float64) float64 {
	if v := os.Getenv(name); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// TestRecord shoots the first window for GOWT_DEMO_SECONDS after GOWT_DEMO_WARMUP seconds into
// GOWT_DEMO_OUT/f-NNNNN.png and writes frames.txt for ffmpeg's concat demuxer; a frame lasts until
// the next one.
func TestRecord(t *testing.T) {
	out := os.Getenv("GOWT_DEMO_OUT")
	if out == "" || os.Getenv("GOWT_DEMO_BIN") == "" {
		t.Skip("set GOWT_DEMO_BIN and GOWT_DEMO_OUT (tooling/demo-video.sh record)")
	}
	out, err := filepath.Abs(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	app := qatest.App(t)
	// The charts show the last minute; warming up fills them so the video starts on a full chart.
	time.Sleep(time.Duration(envSeconds("GOWT_DEMO_WARMUP", 60) * float64(time.Second)))
	length := time.Duration(envSeconds("GOWT_DEMO_SECONDS", 14) * float64(time.Second))

	var names []string
	var at []time.Duration
	steps := map[string]float64{} // milliseconds per step of the shot command, summed
	var total time.Duration
	start := time.Now()
	for time.Since(start) < length {
		name := fmt.Sprintf("f-%05d.png", len(names))
		t0 := time.Since(start)
		raw, err := app.Call("shot", map[string]any{"path": filepath.Join(out, name), "fast": true, "timing": true})
		if err != nil {
			t.Fatalf("shot %d: %v", len(names), err)
		}
		var res struct{ MS map[string]float64 }
		if json.Unmarshal(raw, &res) == nil {
			for k, v := range res.MS {
				steps[k] += v
			}
		}
		total += time.Since(start) - t0
		names = append(names, name)
		at = append(at, (t0+time.Since(start))/2)
	}
	var b strings.Builder
	for i, n := range names {
		next := time.Since(start)
		if i+1 < len(at) {
			next = at[i+1]
		}
		fmt.Fprintf(&b, "file '%s'\nduration %.4f\n", n, (next - at[i]).Seconds())
	}
	fmt.Fprintf(&b, "file '%s'\n", names[len(names)-1])
	if err := os.WriteFile(filepath.Join(out, "frames.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	n := float64(len(names))
	t.Logf("per frame: round trip %.0f ms", total.Seconds()*1000/n)
	for _, k := range []string{"wait", "capture", "overlay", "encode", "popovers"} {
		if v, ok := steps[k]; ok {
			t.Logf("per frame: %-8s %6.1f ms", k, v/n)
		}
	}
	t.Logf("%d frames in %.1f s: %.1f fps", len(names), time.Since(start).Seconds(), float64(len(names))/time.Since(start).Seconds())
}
