# Screenshots of the examples

The pictures in `docs/img/` are made by the programs themselves, in-process, with no screen recording and no extra permission.

## GOWT_SNAP

Any program built on `gowt.Run` honours two environment variables:

| Variable | Meaning |
|---|---|
| `GOWT_SNAP=<file.png>` | after the event loop has run for a while, write the first window as a PNG, close it and let `Run` return |
| `GOWT_SNAP_DELAY=<ms>` | that wait, default 800; raise it for a window that loads something (the web view examples use 4000) |

```sh
GOWT_SNAP=/tmp/hello.png go run ./examples/hello
```

Without `GOWT_SNAP` nothing changes; there is no API to call. `Run` returns an error if the program has no window or the capture fails.

How the window is read per OS (`internal/shot`):

| OS | Capture |
|---|---|
| macOS | `cacheDisplayInRect:toBitmapImageRep:` on the window's view tree, so no Screen Recording permission is needed (the terminal does not have it) |
| Windows | `BitBlt` from the window DC (the window must be on top and on screen) |
| Linux | `gdk_pixbuf_get_from_window` on the window's `GdkWindow` (X server content, no window frame) |

The capture is the window's client area.

## Make targets

| Target | Where | What |
|---|---|---|
| `make docs-shots` | macOS | opens each example and each catalog sample in turn, writes `docs/img/*-macos.png` |
| `make win-docs-shots` | CrossOver bottle `gowt` | cross-builds the `.exe` files, runs them under Wine, writes `docs/img/*-windows.png` |
| `make linux-docs-shots` | Docker stand `gowt-linux` | builds and runs inside the container, writes `docs/img/*-linux.png` |

All three run `tooling/docs-shots.sh <os>`: it builds each program in `examples/` except `controlexample` and `catalog`, runs it with `GOWT_SNAP=docs/img/example-<name>-<os>.png`, then runs `examples/catalog <widget>` for every sample, writing `docs/img/<widget>-<os>.png` (the [widget catalog](widget-catalog.md)). Windows open one after another and close by themselves; a program that does not produce a file within 60 seconds is killed and reported as `FAILED`. The script exits non-zero if any shot failed. Use another stand container with `make linux-docs-shots LINUX_CTR=name LINUX_PORT=6090`.

Only the Linux pictures exist so far; the macOS and Windows runs have not been done. To add a widget to the catalog, add a line to `samples` in `examples/catalog/main.go`; the script reads the names from there.
