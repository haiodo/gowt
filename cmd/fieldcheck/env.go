package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Env is the machine and checkout the report was made on. Nothing here opens a window.
type Env struct {
	GOOS         string            `json:"goos"`
	GOARCH       string            `json:"goarch"`
	OS           map[string]string `json:"os"`
	CPU          string            `json:"cpu,omitempty"`
	NumCPU       int               `json:"num_cpu"`
	GoVersion    string            `json:"go_version"`
	Git          map[string]string `json:"git"`
	Session      map[string]string `json:"session,omitempty"` // Linux: XDG_SESSION_TYPE, WAYLAND_DISPLAY, ...
	WebView      map[string]string `json:"webview,omitempty"` // Windows: WebView2 runtime; Linux: WebKitGTK
	Display      *Probe            `json:"display,omitempty"` // filled by the probe suite
	DisplayError string            `json:"display_error,omitempty"`
}

// Probe is what a gowt app reads from the display: theme and monitor.
type Probe struct {
	Dark         bool `json:"dark"`
	ScreenWidth  int  `json:"screen_width"`
	ScreenHeight int  `json:"screen_height"`
	ZoomPercent  int  `json:"zoom_percent"`
}

// capture runs a short command and returns its trimmed stdout, or "" when it fails.
func capture(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &out
	if cmd.Run() != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

// parseKV reads KEY=value lines (os-release); values may be quoted.
func parseKV(s string) map[string]string {
	m := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		m[k] = strings.Trim(v, `"'`)
	}
	return m
}

// parseReg returns the data of one value from `reg query` output: "    Name    REG_SZ    data".
func parseReg(out, name string) string {
	re := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(name) + `\s+REG_\w+\s+(.*?)\s*$`)
	if m := re.FindStringSubmatch(strings.ReplaceAll(out, "\r", "")); m != nil {
		return m[1]
	}
	return ""
}

func collectEnv() Env {
	e := Env{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, NumCPU: runtime.NumCPU(), OS: map[string]string{}}
	e.GoVersion = capture("go", "version")
	e.Git = gitInfo()
	switch runtime.GOOS {
	case "darwin":
		e.OS["name"] = capture("sw_vers", "-productName")
		e.OS["version"] = capture("sw_vers", "-productVersion")
		e.OS["build"] = capture("sw_vers", "-buildVersion")
		e.OS["kernel"] = capture("uname", "-r")
		e.CPU = capture("sysctl", "-n", "machdep.cpu.brand_string")
		if m := capture("sysctl", "-n", "hw.model"); m != "" {
			e.OS["model"] = m
		}
	case "windows":
		windowsInfo(&e)
	default:
		if b, err := os.ReadFile("/etc/os-release"); err == nil {
			rel := parseKV(string(b))
			e.OS["name"] = rel["PRETTY_NAME"]
			e.OS["id"] = rel["ID"]
			e.OS["version"] = rel["VERSION_ID"]
		}
		e.OS["kernel"] = capture("uname", "-r")
		e.CPU = cpuinfoModel()
		e.Session = map[string]string{}
		for _, k := range []string{"XDG_SESSION_TYPE", "XDG_CURRENT_DESKTOP", "WAYLAND_DISPLAY", "DISPLAY", "GDK_BACKEND"} {
			e.Session[k] = os.Getenv(k)
		}
		if v := capture("pkg-config", "--modversion", "webkit2gtk-4.1"); v != "" {
			e.WebView = map[string]string{"webkit2gtk-4.1": v}
		}
		if v := capture("pkg-config", "--modversion", "gtk+-3.0"); v != "" {
			e.OS["gtk3"] = v
		}
	}
	return e
}

func gitInfo() map[string]string {
	g := map[string]string{}
	g["commit"] = capture("git", "rev-parse", "HEAD")
	g["describe"] = capture("git", "describe", "--tags", "--always")
	g["branch"] = capture("git", "rev-parse", "--abbrev-ref", "HEAD")
	// Untracked files do not count: the zip this tool writes is one.
	g["dirty"] = strconv.FormatBool(capture("git", "status", "--porcelain", "--untracked-files=no") != "")
	return g
}

func cpuinfoModel() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "model name" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

const (
	winCurrent = `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`
	wv2Client  = `\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
)

func windowsInfo(e *Env) {
	reg := capture("reg", "query", winCurrent)
	build, _ := strconv.Atoi(parseReg(reg, "CurrentBuild"))
	name := parseReg(reg, "ProductName")
	e.OS["registry_product_name"] = name
	// Windows 11 still reports "Windows 10 ..." in ProductName; the build number tells them apart.
	if build >= 22000 {
		name = strings.Replace(name, "Windows 10", "Windows 11", 1)
	}
	e.OS["name"] = name
	e.OS["version"] = parseReg(reg, "DisplayVersion")
	e.OS["build"] = parseReg(reg, "CurrentBuild")
	e.OS["ubr"] = parseDword(parseReg(reg, "UBR"))
	e.OS["ver"] = capture("cmd", "/c", "ver")
	e.CPU = os.Getenv("PROCESSOR_IDENTIFIER")
	for _, path := range []string{
		`HKLM\SOFTWARE\WOW6432Node` + wv2Client[len(`\SOFTWARE`):],
		`HKLM` + wv2Client,
		`HKCU` + wv2Client,
	} {
		if v := parseReg(capture("reg", "query", path, "/v", "pv"), "pv"); v != "" && v != "0.0.0.0" {
			e.WebView = map[string]string{"webview2_runtime": v}
			return
		}
	}
	e.WebView = map[string]string{"webview2_runtime": "not found"}
}

// parseDword turns reg's 0x... DWORD data into decimal; other text stays.
func parseDword(s string) string {
	if h, ok := strings.CutPrefix(s, "0x"); ok {
		if n, err := strconv.ParseInt(h, 16, 64); err == nil {
			return strconv.FormatInt(n, 10)
		}
	}
	return s
}
