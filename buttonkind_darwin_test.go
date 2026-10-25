package gowt

import (
	"testing"

	"github.com/haiodo/gowt/internal/cocoa"
)

func checkButtonKinds(t *testing.T, plain, primary, destructive *ButtonView) {
	if cocoa.ButtonDestructive(plain.Unwrap().View) || cocoa.ButtonDestructive(primary.Unwrap().View) {
		t.Error("only a Destructive button has hasDestructiveAction")
	}
	if !cocoa.ButtonDestructive(destructive.Unwrap().View) {
		t.Error("Destructive: hasDestructiveAction not set")
	}
}
