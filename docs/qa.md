# Testing an app with the QA driver

The QA driver lets a `go test` process use your app like a user: type into fields, press buttons, select rows, wait until the app is quiet, read what is on screen. The checks are ordinary Go tests.

## How it works

- You build the app a second time with `-tags gowtqa`. That build contains a small server. `gowt.Run` starts it when the environment variable `GOWT_QA_SOCKET` is set.
- The test process (package `qa/qatest`) starts the QA build, connects to its socket and sends JSON commands: dump, find, act, settle, shot, state, jobs. On Windows, where a unix socket may not be available, the app listens on `127.0.0.1` with a random token and writes the address next to the socket path.
- Input is semantic, not synthetic. `Click` runs the control's own selection event, `Type` sends key events and sets the text one character at a time. No mouse or keyboard event goes through the operating system, so the test does not move your cursor and does not need the app to have the focus.
- On macOS and Windows the QA build opens windows without activating them, so a test run does not steal the focus from what you are doing. On Linux windows open normally.
- The build without the tag has no server and no network code. `qa.Handle` is an empty function there. `make qa-check` ([tooling/qa-check.sh](../tooling/qa-check.sh)) builds a fixture both ways and fails if the server's symbols appear in the plain build, or if `qa/qatest` links a toolkit package.
- `qatest` imports no toolkit, so the test binary does not link AppKit, Win32 or GTK. The app process opens the windows.

## Quick start

The program is [`examples/qa`](../examples/qa).

**1. Name the controls** the test needs with `gowt.ID`. It has no effect on the control and is the same in every build.

```go
gowt.Text(gowt.ID("name"), gowt.OnChange(func(s string) { name.Set(s) })),
gowt.Button("Greet", gowt.ID("greet"), gowt.OnClick(func() { greeting.Set("Hello, " + name.Get()) })),
gowt.Label("", gowt.ID("greeting"), gowt.BindText(greeting)),
```

**2. Register state handlers** with package `qa`. A handler runs on the UI thread. The handler named `"reset"` is called before every test; make it return the app to its start state.

```go
qa.Handle("reset", func(qa.Args) error { greeting.Set(""); return nil })
qa.HandleValue("greeting", func(qa.Args) (any, error) { return greeting.Get(), nil })
```

Handlers read their arguments from `qa.Args`: `Has`, `String`, `Int`, `Bool` (zero value when the key is absent or of another type) and `Decode(key, &v)`.

**3. Write `TestMain` and a test.** `qatest.Main` starts the app on first use and stops it after the last test.

```go
func TestMain(m *testing.M) {
	// build the app with -tags gowtqa into bin, see examples/qa/qa_test.go
	qatest.Main(m, bin)
}

func TestGreet(t *testing.T) {
	app := qatest.App(t) // runs the "reset" handler first

	app.Find(qa.ID("name")).Type("Ada")
	app.Find(qa.ID("greet")).Click() // actions settle before they return

	if got := app.Find(qa.ID("greeting")).Node().Text; got != "Hello, Ada" {
		t.Fatalf("greeting is %q", got)
	}
}
```

Every `qatest` method fails the test with `t.Fatalf`; none returns an error. `Main` takes the binary path and extra arguments for the app.

**4. Build and run.** The example builds the app in `TestMain` with `go build -tags gowtqa`, and opens windows only when `GOWT_GUI_TEST` is set; its tests skip otherwise.

```sh
GOWT_GUI_TEST=1 go test ./examples/qa
```

`GOWT_GUI_TEST` is a convention of the example and of gowt's own tests, not something `qatest` reads. If your `TestMain` builds nothing and the binary comes from elsewhere, pass its path to `qatest.Main` as is.

### Environment

| Variable | Set by | Meaning |
|---|---|---|
| `GOWT_QA_SOCKET` | `qatest` | Socket path. The QA build listens only when it is set; without it the build behaves like a normal one. |
| `GOWT_QA_DATA` | `qatest` | A private empty directory for the app's files. The toolkit does not read it; your app must, via `os.Getenv(qa.EnvData)`, to keep tests away from the user's data. |
| `GOWT_QA_ARTIFACTS` | you | Directory for screenshots and failure files. Without it a temp directory is made per call. |
| `GOWT_QA_KEEP` | you | Any value keeps the temp directory of the run (socket, data, `app.log`) after the app stops. |
| `GOWT_QA_APP` | you | Not read by `qatest`. [qa/qatest/e2e_gui_test.go](../qa/qatest/e2e_gui_test.go) uses it to take a prebuilt binary instead of building one; do the same in your `TestMain` if you need it. |

## Selectors

A `qa.Selector` says which control. An empty field matches anything; all given fields must match.

| Constructor | Matches |
|---|---|
| `qa.ID("send")` | the control created with `gowt.ID("send")` |
| `qa.Class("Button")` | the swt class name from the dump: `Button`, `Text`, `Table`, `TableItem`, ... |
| `qa.Text("Save")` | text equal to the node's `Text` or `Value` |
| `qa.Text("~Sav")` | a text that starts with `~` matches a part of `Text` or `Value`. A literal text cannot start with `~`. |
| `sel.In(parent)` | narrows `sel` to the descendants of the one control `parent` picks |

```go
qa.Class("TableItem").In(qa.ID("contacts"))        // rows of one table
qa.Text("~Ada").In(qa.ID("contacts"))              // the row with "Ada" in its first cell or in the other cells
qa.ID("ok").In(qa.ID("settings"))                  // a control inside the one named "settings"
```

Matching is strict:

- An action needs exactly one match. Zero or several fail with the count and the path, class and text of each match: `qatree: click on {class=Button}: 2 controls match: 0/Composite.0/Button.0 (Button "Save"), ...`.
- `In` needs its parent to match exactly one control, otherwise the error says `within ... matches N controls`. The parent itself is not a candidate.
- `Element.Nodes()` returns an empty list for no match, `Exists()` reports it, and `Node()` fails unless there is exactly one.
- A selector is looked up again on every call, so an `Element` kept in a variable never goes stale.

### Reading the dump

`app.Dump()` returns one tree per open window; windows that are hidden or closed are not in it. When a selector does not match, print the dump (a failed test saves it as `dump.json`) and copy what you see.

| Field | Meaning |
|---|---|
| `ID` | the `gowt.ID`, also set on items that are widgets: table, tree and tab items, tool items, menu items |
| `Class` | swt class: `Shell`, `Button`, `Label`, `Text`, `Table`, `Tree`, ... |
| `Text` | the label or title; for a `Text` field its hint (the message), not its content |
| `Value` | the content of `Text`, `StyledText` and `Combo`; the number of `Scale`, `Slider`, `Spinner`, `ProgressBar`; the selected item of a `List`; the other cells of a table row, tab separated |
| `Enabled`, `Visible` | already combined with the parents, so false means the user cannot reach it. Controls on a TabFolder page that is not selected, and children of a collapsed tree item, are not visible. |
| `Focused` | this control has the focus |
| `Bounds` | `[x y w h]` in the coordinates of its window |
| `Checked` | set for check, radio and toggle buttons, check items of a table, tree, tool bar and menu |
| `Selected` | selected indices in a `List`, `Combo`, `Table`, `Tree` (position among all items in pre-order, collapsed ones too), `TabFolder` |
| `Path` | position in the tree such as `0/Composite.1/Button.0`; stable within one dump |

Items have their own classes:

| Class | Where | `Text` |
|---|---|---|
| `TableItem` | `Table` | first cell; the rest are in `Value` |
| `TreeItem` | `Tree`, nested | first cell |
| `ListItem`, `ComboItem` | `List`, `Combo`; swt has no item objects, so these have no `ID` | the entry |
| `TabItem`, `CTabItem` | `TabFolder`, `CTabFolder` | tab title |
| `ToolItem` | `ToolBar` | item text |
| `MenuItem` | the window's menu bar, nested for submenus | item text |

## Actions

```go
app.Find(qa.ID("composer")).Type("hi")   // replaces the text of a Text, StyledText or editable Combo
app.Find(qa.ID("send")).Click()          // button, check, radio, link, tool item, menu item, row, tab
app.Find(qa.ID("remember")).Check(true)  // check box, radio, toggle, check item of a table or tree row
app.Find(qa.ID("rooms")).Select(2)       // item by index in a list, combo, table, tree or tab folder
```

| Action | Notes |
|---|---|
| `Click` | A button gets its selection event. Check and radio buttons toggle (a radio button is selected). A link reports its `href`. A `TableItem`, `TreeItem`, `TabItem`, `CTabItem`, `ListItem` or `ComboItem` is selected. Click on a menu item that opens a submenu fails. |
| `Type(text)` | Sets the focus, clears the field, then for each character sends `KeyDown` (a listener may veto it), sets the text with the character added, sends `KeyUp`. So `OnChange` runs once per character. A read-only field fails. |
| `Check(on)` | Does nothing when the state already is `on`. A radio button cannot be cleared; select another one. A control without a checked state fails. |
| `Select(index)` | The index counts the items in the dump's order; for a tree, in pre-order. |

The `act` command also accepts per-call `settle` and `timeout_ms`; `qatest` has no method for them, use `app.Call("act", ...)`.

Before an action runs the driver checks that the user could do it. It refuses, with the reason in the error, when the control or an ancestor is not visible or is disabled, when its window is closed, or when an application-modal or system-modal window is open above it.

Every action settles before it returns (see Waiting), so the next line of the test sees the result.

More actions:

| Call | Does |
|---|---|
| `SelectText("b")` | Selects the item with this text in a list, combo, table, tree or tab folder. |
| `Activate()` | Double click or Enter: a List, Table or Tree row (selected first), the selected row of the control, a single-line Text, a push Button, a window (its default button). `OnActivate` runs once. |
| `Expand()`, `Collapse()` | A Tree row. The Expand event runs first, so a lazy tree builds its children; a lazy node needs a placeholder child. |
| `Focus()` | Gives the control focus (`OnFocus`). On macOS the app stays inactive and no window is brought to the front. |
| `Close()` | Closes a window like its close button; `OnClose` may veto. |
| `Key("Mod+K")`, `app.Key(...)` | A key or combo: `Enter`, `Esc`, `Shift+Tab`, `F5`, `Mod+K`. `Mod` is Cmd on macOS, Ctrl elsewhere. `app.Key` goes to the focus control. |
| `ScrollTo(sel)`, `ScrollBy(dx, dy)`, `ScrollIntoView()` | Scroll a ScrollView to a control, by pixels, or the ScrollView holding this control. `OnScroll` runs. |

`Key`: a menu item with that accelerator runs and the control sees nothing, as with macOS key equivalents. Otherwise `Window.OnKey` filters and the control's key listeners get KeyDown, then KeyUp; Enter that was not vetoed also activates. A key does not type text; use `Type`.

`app.Eventually(timeout, func() (observed any, ok bool))` polls for what settle cannot see (a server push, a file); the error shows the last observed value.

## Dialogs

In a QA build `MessageBox`, `Confirm`, `FileDialog`, `DirDialog`, `ColorDialog` and `FontDialog` are not shown. Queue the answer before the action:

```go
app.Answer(qa.Dialog{Kind: "confirm", Reply: true})
app.Find(qa.ID("delete")).Click()
d := app.Dialogs()[0] // Kind, Title, Message, Buttons, Dir, Filters ...
```

| Kind | Reply |
|---|---|
| `message` | button name: `"OK"`, `"Cancel"`, `"Yes"`, `"No"`, `"Retry"`, `"Abort"`, `"Ignore"` |
| `confirm` | `true` or `false` |
| `file` | a path, or a list of paths for a `Multi` open dialog |
| `dir` | a path |
| `color` | `"#rrggbb"` |
| `font` | `gowt.Font` fields, e.g. `map[string]any{"Name": "Menlo", "Size": 12}` |

`Cancel: true` answers as a dismissed dialog. Answers are used in order per kind. A dialog with no answer gets Cancel and fails the action that opened it, so a test never hangs on a native dialog. `App(t)` clears answers and the record.

## Waiting

An action returns after `settle`; call `app.Settle()` yourself after something the test did not start, for example a timer. Settle loops until all of these hold:

- the UI queue is empty: pending `app.Async` and `rx` effects, and window relayouts;
- no `app.Job` or `app.Track` is running, and its `Then` callbacks have run;
- no `app.After` timer shorter than one second is waiting;
- two dumps taken in a row are equal.

It does not wait for jobs started with `gowt.BackgroundJob()`, for `app.After` of one second or more, or for `app.Every`.

Settle gives up after 5 seconds. The error says what it was still waiting for:

```
settle: not quiet after 5s: waiting: job "Download report" 4.2s, track "Send message" 1.0s, 1 timer
settle: not quiet after 5s: waiting: nothing queued; the dump kept changing
```

The second form means the UI never stopped changing, for example a clock label. Stop such updates in the QA build or in `reset`.

Tests should not sleep. `app.Eventually` (see Actions) is for the rare condition outside the app; for the app's own work, make the app tell the driver about it:

| Work | Wrap it in | Waited for |
|---|---|---|
| Your goroutine: a request, a file scan | `app.Job(name, fn)`; update the UI in `.Then(...)` | yes |
| Work gowt does not start: a library callback, your own HTTP client | `done := app.Track(name)`; call `done()` inside `app.Async` after the result is shown | yes |
| A poll or sync loop that never ends | `app.Job(name, fn, gowt.BackgroundJob())` | no, but listed |

```go
app.Job("Load greeting", func(ctx context.Context, p *gowt.JobProgress) error {
	time.Sleep(time.Second) // a slow request
	return nil
}).Then(func(error) { greeting.Set("Loaded") })
```

The click that starts this job returns after the job and its `Then` have finished, so the test reads the result at once. A bare `go func()` that calls `app.Async` is invisible to settle: wrap it. Details of `Job`, `Track` and `Jobs` are in [events.md](events.md#background-work-job-track-jobs).

`app.Jobs()` returns the running jobs and tracks, background ones included, as `[]qa.Job` (`Name`, `Background`, `Track`, `ElapsedMS`, `Done`, `Total`, `Text`).

## State and isolation

- One app process serves a whole test package. The process starts at the first `qatest.App(t)` and stops after the last test. Tests of one package must not run in parallel.
- `qatest.App(t)` calls the `"reset"` handler before each test. Without one the state carries over from the previous test. Your `reset` should close extra windows, clear models and set inputs back; it does not restart anything.
- `qatest.RestartPerTest = true`, set in `TestMain` before `Main`, starts a fresh process for every `App(t)` instead. It is slow and gives a clean slate, including the data directory.
- The data directory is `GOWT_QA_DATA`, a new empty temp directory per process, removed when the app stops unless `GOWT_QA_KEEP` is set. An app that stores files in the user's directory will leave test data there; read `qa.EnvData` and use it when set.
- If a command times out or the connection breaks, the session is marked broken, the test fails, and the next `App(t)` starts a new process.
- `app.State(name, key, value, ...)` calls a handler from the test and returns its JSON result: `app.State("greeting")`, `app.State("chat", "room", "general")`. Handlers run on the UI thread; use them to read or set model state that has no control on screen, or to seed data.

## Screenshots and artifacts

- `app.Shot("name")` saves `name.png` of the first window and returns the path. For another window call `app.Call("shot", map[string]any{"window": 1})`.
- A failed test saves the following into `$GOWT_QA_ARTIFACTS/<test name>` (a new temp directory when the variable is unset; the path is logged):

| File | Content |
|---|---|
| `failure.png` | screenshot of the first window |
| `dump.json` | the full control tree |
| `commands.log` | every command sent, with arguments, result or error, and time taken |
| `app.log.tail` | the last 4 KiB of the app's stdout and stderr |

If the connection to the app is already lost, the screenshot and the dump are skipped and the log files are still written.

## Limits

- Not in the dump: the items of a native toolbar set with `look.SetToolbar` (NSToolbar), column headers of a table, context menus of controls (only the window's menu bar is walked), the tray, and the content of a `Browser` or web view. Native dialogs (message boxes, file choosers) are not in the dump either; see Dialogs.
- A modal dialog the facade does not route (one opened through `Unwrap()` or a custom modal `Shell` loop) blocks the UI thread. Commands that need it fail after 10 seconds with `the UI thread did not answer in 10s (a modal dialog?)`.
- There is no true headless mode on macOS and Windows: the app opens real windows on the desktop, without taking the focus. On Linux run the test under Xvfb (`xvfb-run go test ...`). The gowt stand for Linux is Docker with Xvfb.
- The QA driver was never run on Windows. The Windows port itself runs on a real Windows 11 arm64 virtual machine and under CrossOver/Wine; see [platforms.md](platforms.md).
- Input is semantic. A click is delivered to the control even when another control covers it on screen, and a text is typed into a field the user could not focus. A test can pass where a real user would be blocked. Check overlaps with `Bounds` and by eye with a screenshot.
- `Type` sends characters without key modifiers, input methods or shortcuts.
- Pixel output is not compared; `Shot` is for you to look at.
