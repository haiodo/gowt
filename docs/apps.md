# Demo apps

Eight small apps in [`examples/apps`](../examples/apps) use only the `gowt` facade and `look`, with static data and a fixed selection. They are designed for macOS first and are the standing check of the look on macOS, Windows 11 and Linux GNOME, see [look-check.md](look-check.md). Each shot below is the whole window with its frame, light appearance, at the same size on every OS (`kit.SnapWidth` x `kit.SnapHeight` points, exported at 2 px per point), so the three pictures of an app compare side by side. How they are made: [screenshots.md](screenshots.md#app-shots). Single widgets: [components.md](components.md).

## chat

[`examples/apps/chat`](../examples/apps/chat): sidebar sections with avatars, message feed, reactions, links, thread inspector, glass composer capsule.

| macOS | Windows | Linux |
|---|---|---|
| ![chat, macOS](img/app-chat-macos.png) | ![chat, Windows](img/app-chat-windows.png) | ![chat, Linux](img/app-chat-linux.png) |

## dashboard

[`examples/apps/dashboard`](../examples/apps/dashboard): cards, Progress, a Canvas bar chart, activity Table, links.

| macOS | Windows | Linux |
|---|---|---|
| ![dashboard, macOS](img/app-dashboard-macos.png) | ![dashboard, Windows](img/app-dashboard-windows.png) | ![dashboard, Linux](img/app-dashboard-linux.png) |

## editor

[`examples/apps/editor`](../examples/apps/editor): Tree, closable CTabs, StyledText with highlighting in the mono font, line numbers, status bar.

| macOS | Windows | Linux |
|---|---|---|
| ![editor, macOS](img/app-editor-macos.png) | ![editor, Windows](img/app-editor-windows.png) | ![editor, Linux](img/app-editor-linux.png) |

## files

[`examples/apps/files`](../examples/apps/files): Tree sidebar, sorted Table with icons, path bar with a Link, search field, status line.

| macOS | Windows | Linux |
|---|---|---|
| ![files, macOS](img/app-files-macos.png) | ![files, Windows](img/app-files-windows.png) | ![files, Linux](img/app-files-linux.png) |

## mail

[`examples/apps/mail`](../examples/apps/mail): three panes, unread dots and dates in a Table, reader, toolbar, Split.

| macOS | Windows | Linux |
|---|---|---|
| ![mail, macOS](img/app-mail-macos.png) | ![mail, Windows](img/app-mail-windows.png) | ![mail, Linux](img/app-mail-linux.png) |

## settings

[`examples/apps/settings`](../examples/apps/settings): category sidebar, Sections with Text, Combo, Switch, Check, Radio, Scale, Spinner, primary and destructive buttons.

| macOS | Windows | Linux |
|---|---|---|
| ![settings, macOS](img/app-settings-macos.png) | ![settings, Windows](img/app-settings-windows.png) | ![settings, Linux](img/app-settings-linux.png) |

## tracker

[`examples/apps/tracker`](../examples/apps/tracker): Tabs, filter bar, Table with status dots, inspector with badges and a multi-line Text.

| macOS | Windows | Linux |
|---|---|---|
| ![tracker, macOS](img/app-tracker-macos.png) | ![tracker, Windows](img/app-tracker-windows.png) | ![tracker, Linux](img/app-tracker-linux.png) |

## monitor

[`examples/apps/monitor`](../examples/apps/monitor): live host monitor, CPU, memory and network history charts drawn with `gfx`, per-core bars, native cards and a sortable process table over the real system counters.

| macOS | Windows | Linux |
|---|---|---|
| ![monitor, macOS](img/app-monitor-macos.png) | ![monitor, Windows](img/app-monitor-windows.png) | ![monitor, Linux](img/app-monitor-linux.png) |

Windows is a Windows 11 virtual machine, Linux an Ubuntu GNOME desktop on Wayland in a virtual machine (light theme); `docs/img/source-<os>.txt` records the stand.
