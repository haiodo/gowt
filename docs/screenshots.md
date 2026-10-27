# Screenshots of the examples

The pictures in `docs/img/` are made by the programs themselves, in-process, with no screen recording and no extra permission.

## GOWT_SNAP

Any program built on `gowt.Run` honours two environment variables:

| Variable | Meaning |
|---|---|
| `GOWT_SNAP=<file.png>` | after the event loop has run for a while, write the first window as a PNG, close it and let `Run` return |
| `GOWT_SNAP_LAYERS=1` | macOS: render the window's layer tree (`CALayer renderInContext:`) instead of `cacheDisplayInRect:`, so layer-only content such as `Section` cards shows; glass and blurred backdrops stay empty. The reference images use the default |
| `GOWT_SNAP_SCREEN=1` | macOS: capture the window as the window server composites it (`CGWindowListCreateImage`), so glass, backdrops and the toolbar show; only this window, no desktop behind it. It may need Screen Recording permission |
| `GOWT_SNAP_DELAY=<ms>` | that wait, default 800. With a web view in the program the capture also waits until its last navigation has finished (plus 2 s for the first paint), at most 30 s more. A gfx view in the window must also have presented a frame after that wait, with no tile queued, for at most 3 s more. On macOS the tiles of a gfx view are drawn into the capture from the compositor's images (not with `GOWT_SNAP_SCREEN=1`, which has them from the window server) |

```sh
GOWT_SNAP=/tmp/hello.png go run ./examples/hello
```

Without `GOWT_SNAP` nothing changes; there is no API to call. `Run` returns an error if the program has no window or the capture fails.

How the window is read per OS (`internal/shot`):

| OS | Capture |
|---|---|
| macOS | `cacheDisplayInRect:toBitmapImageRep:` on the window's frame view (title bar and traffic lights, no shadow; `GOWT_SNAP_SCREEN=1` drops the shadow too), so no Screen Recording permission is needed (the terminal does not have it) |
| Windows | the window rectangle without the invisible resize border (`DWMWA_EXTENDED_FRAME_BOUNDS`), copied from the screen with `BitBlt`, so the title bar, the DWM frame and WebView2 children are in; the rounded corners are made transparent. The window must be on top and on screen. `GOWT_SNAP_FULL=1` uses `PrintWindow(PW_RENDERFULLCONTENT)` on the window instead |
| Linux | `gdk_pixbuf_get_from_window` on the window's `GdkWindow`: with client-side decorations (GNOME Wayland, GTK on X11) the title bar is in the surface and the transparent shadow margin is trimmed; a window-manager frame (`gdk_window_get_frame_extents` larger than the window) is read from the root window (not tried on a real X11 window manager) |

The capture is the whole window with its frame. `GOWT_SNAP_SIZE=<w>x<h>` (set by `examples/catalog`) names the window's outer size in points; the PNG is then exported at exactly `2w x 2h` pixels: a capture at scale 1 or 1.5 is scaled up (CatmullRom), a capture within 3% of the size is stretched to it. `GOWT_SNAP` prints the capture scale on stderr (`gowt: snap capture scale 1.5, exported at 2x as 720x520, ...`).

## Uniform catalog shots

The images of one catalog sample have the same pixel size on macOS, Windows and Linux, title bar included, so they can sit side by side. The size table is `sizes` in `examples/catalog/main.go` (outer size in points, `defaultSize` 320x200); `frameSlack` adds what a platform's window rect hides (Windows invisible border, Adwaita's smaller GTK window). `make shot-sizes` fails if any `<widget>-{macos,windows,linux}.png` differs from `2 x` that table; `go test ./examples/catalog` runs the same check for Windows and Linux only, until macOS is shot again.

| OS | Display scale of the shots |
|---|---|
| macOS | 2 (Retina), no change |
| Windows VM | 1.5: the 1728x1044 display follows the UTM window, and Windows takes 150% at most there; `tooling/vm-scale.sh windows 1.5` sets it with `tooling/win-dpi.ps1`, the script sets it back afterwards |
| Linux VM | 2: `tooling/vm-scale.sh linux 2` (Mutter `ApplyMonitorsConfig`, temporary); web view examples run through Xwayland with `GDK_SCALE=2` |

`docs/img/source-<os>.txt` records the scale the programs saw.

## Make targets

| Target | Where | What |
|---|---|---|
| `make docs-shots` | macOS | opens each example and each catalog sample in turn, writes `docs/img/*-macos.png` |
| `make vm-docs-shots-windows` | Windows 11 arm64 UTM VM (`.work/winvm/vm.sh`) | cross-builds for arm64, runs on the logged-on desktop, writes `docs/img/*-windows.png` and `source-windows.txt` |
| `make vm-docs-shots-linux` | Ubuntu 26.04 GNOME UTM VM with a GPU (`.work/ubvm/vm.sh`) | native Wayland, `GTK_THEME=Adwaita:light`, writes `docs/img/*-linux.png` and `source-linux.txt` |
| `make win-docs-shots` | CrossOver bottle `gowt` | Wine render, not used for `docs/img` any more |
| `make linux-docs-shots` | Docker stand `gowt-linux` | Xvfb render, not used for `docs/img` any more |

`tooling/docs-shots-vm.sh windows|linux` (`ONLY="coolbar example-hello"` shoots a few) copies the programs to the VM, runs them one after another and fetches the PNGs; `source-<os>.txt` records the stand, commit and date. The web view examples are shot through Xwayland because native Wayland reads a WebKitGTK window back empty; under Xwayland the window has no frame, so those two images lack the title bar. `GOWT_SNAP_FULL=1` (Windows) copies with `PrintWindow(PW_RENDERFULLCONTENT)` so WebView2 shows; `GOWT_SNAP_NOFOCUS=1` (macOS) takes the focus ring off before the capture, `tooling/docs-shots.sh` sets it.

All three run `tooling/docs-shots.sh <os>`: it builds each program in `examples/` except `controlexample` and `catalog`, runs it with `GOWT_SNAP=docs/img/example-<name>-<os>.png`, then runs `examples/catalog <widget>` for every sample, writing `docs/img/<widget>-<os>.png` (the [components](components.md)). Windows open one after another and close by themselves; a program that does not produce a file within 60 seconds is killed and reported as `FAILED`. The script exits non-zero if any shot failed. Use another stand container with `make linux-docs-shots LINUX_CTR=name LINUX_PORT=6090`.

To add a widget to the catalog, add a line to `samples` in `examples/catalog/main.go`; the script reads the names from there.

## App shots

`docs/img/app-<name>-<os>.png` for the eight apps of `examples/apps` (shown on [apps.md](apps.md)). Under `GOWT_SNAP` the apps open at `kit.SnapWidth` x `kit.SnapHeight` points (examples/apps/internal/kit) and set `GOWT_SNAP_SIZE`, so every shot is 2080x1200, light appearance (`GOWT_APPEARANCE=light`). `make shot-sizes` checks them with the catalog samples.

| OS | Command |
|---|---|
| macOS (opens a window per app, `GOWT_SNAP_SCREEN=1` so the glass shows) | `make docs-shots-apps` (= `ONLY=apps tooling/docs-shots.sh macos`) |
| Windows 11 VM | `ONLY=apps make vm-docs-shots-windows` |
| Ubuntu GNOME VM | `ONLY=apps make vm-docs-shots-linux` |

`WINWD` (single backslashes, e.g. `C:\gowt\orch\gallery`) moves the work folder on the Windows VM. The PNGs are not quantised, about 5 MB for the 14 VM shots.

## README video

The loop at the top of the README (`docs/img/monitor-macos.webp`, linked to `docs/video/monitor-macos.mp4`) is the monitor app shot window-only through the QA socket (`tooling/demovideo`), so no desktop shows. Re-record it on macOS, from the repository root, each release:

```
tooling/demo-video.sh macos
```

It takes about 80 s (60 s of warm-up so the charts are full, then 20 s of shots; about 5 GB of raw frames in `bin/demo/macos`, deleted after the assembly unless `KEEP=1`), opens one window and does not touch the mouse. `tooling/demo-video.sh record <label>` and `assemble <label>` run the two halves apart (the recording needs a window server, the assembly needs `ffmpeg` and `img2webp`). `LOOP`, `FPS`, `WIDTH` and `Q` tune the loop length (20 s, a hard cut), the frame rate, the WebP width and its quality; the script prints the sizes (aim for 2-4 MB for the WebP) and fails if the QA shots come in under 10 fps.
