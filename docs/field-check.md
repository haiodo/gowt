# Field check: running gowt's tests on your machine

gowt is developed on one Mac, with Windows under Wine and Linux under Xvfb in Docker and in a GNOME Wayland virtual machine. Real machines (other OS versions, other monitors, Wayland, real Windows) are where the bugs are. `cmd/fieldcheck` runs the checks there with one command and writes one zip file that you attach to a GitHub issue.

## What you need

- A checkout of this repository: `git clone https://github.com/haiodo/gowt && cd gowt`.
- Go, the version in the `go` line of [go.mod](../go.mod) (1.27).
- `git`, so the report can name the commit. Without it the commit field is empty and the run still works.
- No C compiler, no Java, no `make`, no `bash`: the tool is a Go program and runs the same on every OS.
- Linux also needs GTK 3 and a desktop session (see below). For the web view tests, WebKitGTK 4.1. Windows: the WebView2 Runtime (preinstalled on Windows 11).

## Run it

```sh
go run ./cmd/fieldcheck
```

It prints the suite it is on and, at the end, the path of `gowt-fieldcheck-<os>-<arch>-<date>.zip` (in the current directory, or in the one given by `-out dir`) and a one-line summary. Then open a new issue with the [Field check](https://github.com/haiodo/gowt/issues/new?template=field-check.yml) template and drag that one file into it.

| Flag | Effect |
|---|---|
| `-quick` | Runs 4 widget classes of the SWT tests (Button, Text, Table, Shell) instead of all of them; the gate then checks only those. Everything else runs. |
| `-only suite[,suite]` | Runs the named suites only: `env`, `probe`, `swt`, `jface`, `gui`, `qa`, `webview`, `snapshots`. `-only env` opens no window and writes the environment only. |
| `-out dir` | Where the zip goes. |

## What it opens and for how long

The run is not timed on a real machine yet; the SWT suite is the long part (about 3900 tests, one short-lived window or more each), expect tens of minutes at least. The tool kills a suite that exceeds its limit (100 minutes for `swt`, 30 for `gui`, 20 for `jface`, 15 for `qa`, 10 for `webview`, 1 for each snapshot) and goes on with the next one; Ctrl+C stops the run and still writes the zip with what is done.

| Suite | What happens |
|---|---|
| `env` | Reads OS, CPU, Go and git facts. No window. |
| `probe` | Starts gowt without a window to read the theme (light/dark), the primary monitor size and its scale. |
| `swt` | `cmd/swttest` against `tests/expected[_<os>].txt`: the translated SWT tests, windows opening and closing by themselves, the screen may flicker. |
| `jface` | The same runner with `-tags jface` against `tests/expected_jface[_<os>].txt`. |
| `gui` | `GOWT_GUI_TEST=1 go test` of the facade, `look`, `icons` and `swt` packages. |
| `qa` | The QA driver tests (`qa/qatest`) and `examples/qa`. They build a fixture app with `-tags gowtqa` and drive it; on macOS and Windows its windows open without taking the focus ([qa.md](qa.md)). |
| `webview` | The web view test (`./webview`). |
| `snapshots` | Three demo programs (`cmd/gowtdemo`, `cmd/backdropdemo`, `cmd/themecheck`) each open one window for a few seconds and save it as a PNG (`GOWT_SNAP`, [screenshots.md](screenshots.md)). |

While it runs, do not use the mouse and keyboard and do not close the windows: a click on a test window can fail a test.

It changes no system setting and no tracked file of the checkout; it never updates `tests/expected*.txt`. Binaries are built in a temporary folder that is removed at the end. Be aware of two side effects of the tests themselves: the SWT clipboard tests replace the clipboard contents, and the tests that post input events can move the pointer.

## Per OS

- **macOS**: run it from Terminal.
- **Windows**: run it from PowerShell or `cmd` in the checkout. If the machine is a virtual machine, say so in the issue.
- **Linux, X11 or Xwayland**: run it inside the desktop session (not over plain SSH). With no `DISPLAY` and no `WAYLAND_DISPLAY` the tool skips the suites that need a window and says so. On a server, `xvfb-run go run ./cmd/fieldcheck` gives a headless result.
- **Linux, Wayland**: run it in the Wayland session. gowt uses GTK 3 and the report records `XDG_SESSION_TYPE`, `WAYLAND_DISPLAY`, `DISPLAY`, `GDK_BACKEND` and `XDG_CURRENT_DESKTOP`, so X11 and Wayland results can be told apart. The gate was recorded on GNOME Wayland (Ubuntu 26.04, in a virtual machine); other compositors, KDE and real hardware are new information, failures included. `GDK_BACKEND=x11` gives Xwayland.

## What is in the zip

| File | Content |
|---|---|
| `report.md` | A human summary: verdict, the suite table, the first failures with their messages, the environment. |
| `report.json` | The same for machines: `env` (OS, build, CPU, Go, git commit and dirty flag, theme, monitor, session type, WebView2 or WebKitGTK version), and per suite `status`, `counts`, `gate`, `expected`, `failures` (test name and reason). Up to 500 failures per suite; the log has the rest. |
| `logs/` | The full output of every suite. They can contain file paths of your machine. |
| `snapshots/` | The PNGs of the demo programs. |

A suite is `ok` when the program exited with 0. For `swt` and `jface` the `gate` is the verdict of `swttest -expected`: `ok` means no test that is expected to pass failed and no failure is unexplained in the expected file; it does not mean every test passed. The expected files of Windows and Linux were produced on Wine and in Docker, so on a real machine a handful of differences is normal and is exactly what the report shows.

## Compare reports (maintainer)

```sh
go run ./tooling/fieldreport gowt-fieldcheck-*.zip
```

prints one row per zip: OS, architecture, gowt commit, verdict, and for each suite the pass/fail/skip counts, the gate and, for `swt` and `jface`, the counts the expected file of that OS has.
