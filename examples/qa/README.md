# QA driver example

A greeter and a test that drives it like a user.

- `main.go`: widgets named with `gowt.ID`, a `qa.Handle("reset", ...)` handler that returns the app to its start state, and a `qa.HandleValue` that reads the model.
- `qa_test.go`: `qatest.Main` starts the app built with `-tags gowtqa`; `qatest.App(t)` resets it; `Find(qa.ID(...))` with `Type`, `Click`, `Node`; `Settle`; `Shot`.
- The Load button starts an `app.Job`; `TestLoad` clicks it and reads the result at once, because actions settle and settle waits for jobs (see events.md, "Background work").

Run (opens a window):

    GOWT_GUI_TEST=1 go test ./examples/qa

Without `-tags gowtqa` the app has no server and `qa.Handle` does nothing. A failed test leaves a screenshot, the dump and the command log in `$GOWT_QA_ARTIFACTS/<test>` (a temp directory when unset).
