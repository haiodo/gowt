# Dialogs and menus

Example: [`examples/dialogs`](../examples/dialogs).

## Message boxes

Dialogs are methods on the parent `Window`. Settings are one struct, results are plain values.

```go
a := w.MessageBox(gowt.Message{
	Title: "Delete", Text: "Delete the file?",
	Icon: gowt.IconQuestion, Buttons: gowt.ButtonsYesNoCancel,
})
if a == gowt.AnswerYes { ... }
```

- Icons: `IconNone`, `IconInfo`, `IconWarning`, `IconError`, `IconQuestion`.
- Buttons: `ButtonsOK`, `ButtonsOKCancel`, `ButtonsYesNo`, `ButtonsYesNoCancel`, `ButtonsRetryCancel`, `ButtonsAbortRetryIgnore`.
- Answers: `AnswerOK`, `AnswerCancel`, `AnswerYes`, `AnswerNo`, `AnswerRetry`, `AnswerAbort`, `AnswerIgnore`. Closing the box with the window button gives `AnswerCancel` (or `AnswerNo`, `AnswerOK` when there is no Cancel).
- `w.Confirm(title, text) bool` is the short form for a yes/no question.

## File, folder, colour and font dialogs

| Call | Result |
|---|---|
| `w.FileDialog(gowt.FileDialog{Title, Dir, Name, Save, Multi, Filters})` | `[]string`; nil when cancelled |
| `w.DirDialog(title, dir)` | `(path, ok)` |
| `w.ColorDialog(initial RGB)` | `(RGB, ok)` |
| `w.FontDialog(initial Font)` | `(Font, ok)` |

`FileFilter{Name, Pattern}`: a pattern list is separated by `;`, e.g. `"*.png;*.jpg"`; `"*"` matches all. `Font` is `{Name, Size, Bold, Italic}`.

## Menus

```go
bar := w.MenuBar()           // on macOS this is the application menu bar
file := bar.Submenu("&File")
file.Item("&Open...", openFile)
file.Separator()
file.Item("&Quit", w.Close)
view.Item("Word wrap", nil, gowt.Check()).SetChecked(true)
```

- `Item(text, onClick, opts...)` appends an item; `Check()` or `Radio()` makes it a toggle. `MenuItem` has `SetText`, `SetEnabled`, `Checked`, `SetChecked`, `OnClick`.
- `&` in the text marks a mnemonic. Text after a tab character is shown as a shortcut hint, but it does not bind a key: handle the shortcut yourself.
- `Menu.OnShow(f)` runs just before the menu opens; refresh enabled states there.
- Context menu: `gowt.PopupMenu(widget)` returns the widget's menu; a right click opens it, `Show()` opens it by hand. It is a function, not a method.

## Tool bar and tray

`w.ToolBar()` with `Item(text, onClick)` and `Separator()`; items have `SetImage`, `SetTooltip`, `SetEnabled`, `SetChecked`.

`app.Tray(tooltip, onClick)` adds a tray icon and returns nil where the system has no tray, so check for nil. Use `SetImage`, `SetTooltip`, `Remove`.
