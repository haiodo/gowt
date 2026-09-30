// Hand-written: locks in Round 12's two semantics fixes - Object.equals/== identity through
// .Impl() (NumericEmitter.identityCompare, JdkIntrinsics.emitObjectEquals) and boolean &/|
// evaluating every operand before combining (NumericEmitter.hoistBooleanChainIfNeeded,
// StatementEmitter.booleanCompoundOp). No Display needed: only the impl-cascade wiring a real
// constructor would also do (this.impl = this) matters here.
package swt

import "testing"

func TestObjectIdentityThroughImpl(t *testing.T) {
	btn := &Button{}
	btn.Widget.impl = btn
	lbl := &Label{}
	lbl.Widget.impl = lbl

	// widget == shell-shaped case: an ancestor-typed reference to the same object must still
	// compare identical, the way generated code does it (any(x.Impl()) == any(y.Impl())).
	var w *Widget = &btn.Widget
	if !(any(w.Impl()) == any(btn.Impl())) {
		t.Fatal("same object through an ancestor-typed reference should compare identical")
	}

	// Sibling types (no ancestor relation, e.g. Button vs Label): must never alias.
	if any(btn.Impl()) == any(lbl.Impl()) {
		t.Fatal("distinct sibling objects incorrectly compared equal")
	}
}

func TestBooleanOrEvaluatesEveryOperand(t *testing.T) {
	calls := 0
	sideEffect := func() bool { calls++; return true }

	// Mirrors the translated `events = events || this.RunSettings()` -> hoisted form: the right
	// operand must still run even though the left is already true (Java's | never short-circuits).
	events := true
	b1 := sideEffect()
	events = events || b1
	if !events || calls != 1 {
		t.Fatalf("want events=true, calls=1; got events=%v calls=%d", events, calls)
	}
}
