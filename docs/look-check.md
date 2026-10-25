# Look check

Seven small apps in [`examples/apps`](../examples/apps) are the standing visual check for the look on macOS, Windows 11 and Linux GNOME. Each one is designed for macOS first, uses only the `gowt` facade and `look`, has static data and opens at 1280x800 with a fixed selection, so a shot is the same on every run.

| App | Widgets |
|---|---|
| `chat` | sidebar sections with avatars, message feed, reactions, links, thread inspector, glass composer capsule |
| `mail` | three panes, unread dots and dates in a Table, reader, toolbar, Split |
| `settings` | category sidebar, Sections with Text, Combo, Switch, Check, Radio, Scale, Spinner, primary and destructive buttons |
| `files` | Tree sidebar, sorted Table with icons, path bar with a Link, search field, status line |
| `tracker` | Tabs, filter bar, Table with status dots, inspector with badges and a multi-line Text |
| `editor` | Tree, closable CTabs, StyledText with highlighting in the mono font, line numbers, status bar |
| `dashboard` | cards, Progress, a Canvas bar chart, activity Table, links |

`examples/apps/internal/kit` holds the shared avatar, badge and colour helpers. `make examples-check` vets all apps for the three OSes.

## Capture

`tooling/lookshots/lookshots.sh <step>... [-- app...]` writes to `.work/look-apps/` (`OUT` overrides): binaries in `bin/`, shots in `shots/<platform>/<app>.png`, one sheet per app in `<app>.png` with the platforms side by side, scaled to one logical width.

| Step | What | Needs |
|---|---|---|
| `build` | cross-builds every app for darwin/arm64, windows/arm64, linux/arm64 | Go |
| `win` | copies the `.exe` files to the Windows 11 VM and runs them in the desktop session | `.work/winvm/vm.sh`, `VMTAG=lookapps` |
| `linux-vm` | the same on the Ubuntu GNOME VM (native Wayland) | `.work/ubvm/vm.sh` |
| `linux-docker` | the same in the Docker stand (Xvfb, no desktop theme) | Docker, `make linux-vnc` |
| `sheet` | makes the side-by-side sheets from whatever shots exist | Python 3 with Pillow |
| `remote` | `win linux-vm linux-docker sheet` | |
| `mac` | opens each app on this Mac and writes `shots/macos/<app>.png` | a person at the Mac |

```sh
tooling/lookshots/lookshots.sh remote            # Windows and both Linux stands, then the sheets
tooling/lookshots/lookshots.sh mac sheet         # on the Mac, about 30 seconds, windows open one after another
tooling/lookshots/lookshots.sh win sheet -- chat # one app, one platform
```

`mac` captures with `GOWT_SNAP_SCREEN=1` (the composited window, so the glass sidebar and toolbar show); macOS may ask for Screen Recording permission once. `MAC_SCREEN=0` uses the plain view capture instead. `DELAY` sets `GOWT_SNAP_DELAY` in milliseconds (default 1500). The capture mechanism is [`GOWT_SNAP`](screenshots.md).

The VM IPs change: the Ubuntu helper takes the first of 10.0.3.3 and 192.168.65.2 with an open ssh port, or `VMHOST`. The Ubuntu VM must have its desktop session logged in; at the GDM greeter there is no Wayland socket and the capture fails.
