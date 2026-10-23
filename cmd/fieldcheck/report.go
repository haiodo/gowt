package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Report is report.json.
type Report struct {
	Schema      int           `json:"schema"`
	Created     string        `json:"created"`
	Args        []string      `json:"args"`
	Interrupted bool          `json:"interrupted,omitempty"`
	Verdict     string        `json:"verdict"` // ok when every suite that ran is ok or skipped
	Env         Env           `json:"env"`
	Suites      []SuiteResult `json:"suites"`
}

func (r *Report) Finish() {
	r.Verdict = "ok"
	for _, s := range r.Suites {
		if s.Status != "ok" && s.Status != "skipped" {
			r.Verdict = "fail"
		}
	}
	if r.Interrupted {
		r.Verdict = "fail"
	}
}

func (r *Report) osLabel() string {
	e := r.Env
	label := strings.TrimSpace(e.OS["name"] + " " + e.OS["version"])
	if b := e.OS["build"]; b != "" {
		label += " (" + b + ")"
	}
	return label + " " + e.GOARCH
}

// Summary is the one line printed at the end.
func (r *Report) Summary() string {
	var parts []string
	for _, s := range r.Suites {
		p := s.Name + " " + s.Status
		if s.Counts != nil {
			p += fmt.Sprintf(" %d/%d/%d", s.Counts.Passed, s.Counts.Failed, s.Counts.Skipped)
		}
		if s.Gate != "" {
			p += " gate " + s.Gate
		}
		parts = append(parts, p)
	}
	return fmt.Sprintf("%s: %s; %s", r.Verdict, r.osLabel(), strings.Join(parts, "; "))
}

func (r *Report) Markdown() string {
	var b strings.Builder
	e := r.Env
	fmt.Fprintf(&b, "# gowt field check: %s\n\n**Verdict: %s.** %s\n\n", r.osLabel(), r.Verdict, r.Summary())
	if r.Interrupted {
		b.WriteString("The run was interrupted; the suites after the last one listed did not run.\n\n")
	}
	b.WriteString("## Suites\n\n| suite | status | pass | fail | skip | gate | expected (pass/fail/skip) | time |\n|---|---|---|---|---|---|---|---|\n")
	for _, s := range r.Suites {
		c := [3]string{"-", "-", "-"}
		if s.Counts != nil {
			c = [3]string{fmt.Sprint(s.Counts.Passed), fmt.Sprint(s.Counts.Failed), fmt.Sprint(s.Counts.Skipped)}
		}
		exp, gate := "-", "-"
		if s.Expected != nil {
			exp = fmt.Sprintf("%d/%d/%d", s.Expected.Pass, s.Expected.Fail, s.Expected.Skip)
		}
		if s.Gate != "" {
			gate = s.Gate
		}
		note := s.Status
		if s.Note != "" {
			note += " (" + s.Note + ")"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %.0fs |\n", s.Name, note, c[0], c[1], c[2], gate, exp, s.DurationS)
	}
	for _, s := range r.Suites {
		if len(s.Failures) == 0 && len(s.GateLines) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s: failures\n\n", s.Name)
		for _, g := range firstN(s.GateLines, 30) {
			fmt.Fprintf(&b, "- %s\n", g)
		}
		for _, f := range firstN(s.Failures, 30) {
			fmt.Fprintf(&b, "- `%s` %s\n", f.Test, f.Reason)
		}
		if n := len(s.Failures) - 30; n > 0 {
			fmt.Fprintf(&b, "- ... %d more in report.json\n", n)
		}
	}
	b.WriteString("\n## Environment\n\n")
	fmt.Fprintf(&b, "- OS: %s %s, build %s, kernel %s\n", e.OS["name"], e.OS["version"], e.OS["build"], e.OS["kernel"])
	fmt.Fprintf(&b, "- CPU: %s, %d logical, %s\n", e.CPU, e.NumCPU, e.GOARCH)
	fmt.Fprintf(&b, "- Go: %s\n", e.GoVersion)
	fmt.Fprintf(&b, "- gowt: %s (%s), dirty: %s\n", e.Git["commit"], e.Git["describe"], e.Git["dirty"])
	if d := e.Display; d != nil {
		theme := map[bool]string{true: "dark", false: "light"}[d.Dark]
		fmt.Fprintf(&b, "- Display: %dx%d points, zoom %d%%, theme %s\n", d.ScreenWidth, d.ScreenHeight, d.ZoomPercent, theme)
	} else if e.DisplayError != "" {
		fmt.Fprintf(&b, "- Display: %s\n", e.DisplayError)
	}
	for _, k := range sortedKeys(e.Session) {
		fmt.Fprintf(&b, "- %s: %q\n", k, e.Session[k])
	}
	for _, k := range sortedKeys(e.WebView) {
		fmt.Fprintf(&b, "- %s: %s\n", k, e.WebView[k])
	}
	b.WriteString("\nFull logs are in logs/, window pictures in snapshots/, all numbers in report.json.\n")
	return b.String()
}

func firstN[T any](s []T, n int) []T {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func sortedKeys(m map[string]string) []string {
	k := make([]string, 0, len(m))
	for key := range m {
		k = append(k, key)
	}
	sort.Strings(k)
	return k
}

// writeZip stores report.json and report.md first, then the logs and snapshots; bin/ (the test binaries) is left out.
func writeZip(zipPath, dir string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	add := func(rel string) error {
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		src, err := os.Open(filepath.Join(dir, rel))
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(w, src)
		return err
	}
	for _, rel := range []string{"report.md", "report.json"} {
		if err := add(rel); err != nil {
			f.Close()
			return err
		}
	}
	for _, sub := range []string{"logs", "snapshots"} {
		err := filepath.WalkDir(filepath.Join(dir, sub), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(dir, p)
			return add(filepath.ToSlash(rel))
		})
		if err != nil {
			f.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
