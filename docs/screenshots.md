# Screenshots of the examples

The pictures in `docs/img/` are made by the programs themselves, in-process, with no screen recording and no extra permission.

## GOWT_SNAP

Any program built on `gowt.Run` honours two environment variables:

| Variable | Meaning |
|---|---|
| `GOWT_SNAP=<file.png>` | after the event loop has run for a while, write the first window as a PNG, close it and let `Run` return |
| `GOWT_SNAP_LAYERS=1` | macOS: render the window's layer tree (`CALayer renderInContext:`) instead of `cacheDisplayInRect:`, so layer-only content such as `Section` cards shows; glass and blurred backdrops stay empty. The reference images use the default |
| `GOWT_SNAP_SCREEN=1` | macOS: capture the window as the window server composites it (`CGWindowListCreateImage`), so glass, backdrops and the toolbar show; only this window, no desktop behind it. It may need Screen Recording permission |
| `GOWT_SNAP_DELAY=<ms>` | that wait, default 800. With a web view in the program the capture also waits until its last navigation has finished (plus 2 s for the first paint), at most 30 s more |

```sh
GOWT_SNAP=/tmp/hello.png go run ./examples/hello
```

Without `GOWT_SNAP` nothing changes; there is no API to call. `Run` returns an error if the program has no window or the capture fails.

How the window is read per OS (`internal/shot`):

| OS | Capture |
|---|---|
| macOS | `cacheDisplayInRect:toBitmapImageRep:` on the window's view tree, so no Screen Recording permission is needed (the terminal does not have it) |
| Windows | `BitBlt` from the window DC (the window must be on top and on screen) |
| Linux | `gdk_pixbuf_get_from_window` on the window's `GdkWindow` (the window's own surface: X server content without frame, on native Wayland with the GTK frame and shadow) |

The capture is the window's client area.

## Make targets

| Target | Where | What |
|---|---|---|
| `make docs-shots` | macOS | opens each example and each catalog sample in turn, writes `docs/img/*-macos.png` |
| `make vm-docs-shots-windows` | Windows 11 arm64 UTM VM (`.work/winvm/vm.sh`) | cross-builds for arm64, runs on the logged-on desktop, writes `docs/img/*-windows.png` and `source-windows.txt` |
| `make vm-docs-shots-linux` | Ubuntu 26.04 GNOME UTM VM with a GPU (`.work/ubvm/vm.sh`) | native Wayland, `GTK_THEME=Adwaita:light`, writes `docs/img/*-linux.png` and `source-linux.txt` |
| `make win-docs-shots` | CrossOver bottle `gowt` | Wine render, not used for `docs/img` any more |
| `make linux-docs-shots` | Docker stand `gowt-linux` | Xvfb render, not used for `docs/img` any more |

`tooling/docs-shots-vm.sh windows|linux` (`ONLY="coolbar example-hello"` shoots a few) copies the programs to the VM, runs them one after another and fetches the PNGs; `source-<os>.txt` records the stand, commit and date. The Linux images include the GTK window frame and its shadow (transparent) at the VM's scale of 2, the web view examples are shot through Xwayland because native Wayland reads a WebKitGTK window back empty. The Windows images are the client area at the VM's scale. `GOWT_SNAP_FULL=1` (Windows) copies with `PrintWindow(PW_RENDERFULLCONTENT)` so WebView2 shows; `GOWT_SNAP_NOFOCUS=1` (macOS) takes the focus ring off before the capture, `tooling/docs-shots.sh` sets it.

All three run `tooling/docs-shots.sh <os>`: it builds each program in `examples/` except `controlexample` and `catalog`, runs it with `GOWT_SNAP=docs/img/example-<name>-<os>.png`, then runs `examples/catalog <widget>` for every sample, writing `docs/img/<widget>-<os>.png` (the [widget catalog](widget-catalog.md)). Windows open one after another and close by themselves; a program that does not produce a file within 60 seconds is killed and reported as `FAILED`. The script exits non-zero if any shot failed. Use another stand container with `make linux-docs-shots LINUX_CTR=name LINUX_PORT=6090`.

To add a widget to the catalog, add a line to `samples` in `examples/catalog/main.go`; the script reads the names from there.
