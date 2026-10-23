//go:build gowtqa

package qa

import (
	"encoding/json"
	"fmt"
	"sync"
)

// dialogs is the script of the dialogs the app opens and the record of what it asked. A dialog
// of the app runs on the UI thread and must not block, so it takes the answer the test queued.
var dialogs struct {
	sync.Mutex
	active  bool
	answers []Dialog // Reply is json.RawMessage
	asked   []Dialog
	failed  []string
}

// Ask answers a dialog of the app from the script. It returns false when no QA server runs, and
// the app shows the native dialog. The reply is JSON; a dialog without a scripted answer is
// cancelled, counted as a failure of the current action, and answers true with a nil reply and Cancel.
func Ask(req Dialog) (reply json.RawMessage, cancel, handled bool) {
	dialogs.Lock()
	defer dialogs.Unlock()
	if !dialogs.active {
		return nil, false, false
	}
	for i, a := range dialogs.answers {
		if a.Kind != req.Kind {
			continue
		}
		dialogs.answers = append(dialogs.answers[:i], dialogs.answers[i+1:]...)
		req.Cancel = a.Cancel
		if !a.Cancel {
			reply, _ = a.Reply.(json.RawMessage)
			req.Reply = reply
		}
		dialogs.asked = append(dialogs.asked, req)
		return reply, a.Cancel, true
	}
	req.Unanswered, req.Cancel = true, true
	dialogs.asked = append(dialogs.asked, req)
	dialogs.failed = append(dialogs.failed, fmt.Sprintf("the app opened the %s dialog %q with no scripted answer (it got Cancel); queue one with Answer before the action", req.Kind, req.Title))
	return nil, true, true
}

// Failed fails the current command with msg: the scripted reply did not fit the dialog.
func Failed(msg string) {
	dialogs.Lock()
	defer dialogs.Unlock()
	dialogs.failed = append(dialogs.failed, msg)
}

func activateDialogs() {
	dialogs.Lock()
	dialogs.active = true
	dialogs.Unlock()
}

// takeDialogFailures returns and clears the failures, as one error for the command that ran.
func takeDialogFailures() error {
	dialogs.Lock()
	defer dialogs.Unlock()
	if len(dialogs.failed) == 0 {
		return nil
	}
	msg := dialogs.failed[0]
	for _, f := range dialogs.failed[1:] {
		msg += "; " + f
	}
	dialogs.failed = nil
	return fmt.Errorf("dialog: %s", msg)
}

// answerCmd queues the answer args describe.
func answerCmd(raw json.RawMessage) error {
	var a struct {
		Kind   string          `json:"kind"`
		Reply  json.RawMessage `json:"reply"`
		Cancel bool            `json:"cancel"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return fmt.Errorf("answer: %w", err)
	}
	switch a.Kind {
	case "message", "confirm", "file", "dir", "color", "font":
	default:
		return fmt.Errorf("answer: unknown dialog kind %q", a.Kind)
	}
	if !a.Cancel && len(a.Reply) == 0 {
		return fmt.Errorf("answer: the %s answer has no Reply and is not Cancel", a.Kind)
	}
	dialogs.Lock()
	defer dialogs.Unlock()
	dialogs.answers = append(dialogs.answers, Dialog{Kind: a.Kind, Reply: a.Reply, Cancel: a.Cancel})
	return nil
}

// dialogsCmd returns the dialogs asked since the last clear; with "clear" it also drops the
// queued answers and the failures.
func dialogsCmd(raw json.RawMessage) ([]Dialog, error) {
	var a struct {
		Clear bool `json:"clear"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, fmt.Errorf("dialogs: %w", err)
		}
	}
	dialogs.Lock()
	defer dialogs.Unlock()
	out := dialogs.asked
	if a.Clear {
		dialogs.asked, dialogs.answers, dialogs.failed = nil, nil, nil
	}
	return out, nil
}
