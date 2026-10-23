package qatree_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/haiodo/gowt/internal/qatree"
	"github.com/haiodo/gowt/qa"
)

// qa mirrors the types of qatree because the client cannot link the toolkit; this keeps them equal.
func TestClientTypesMatch(t *testing.T) {
	sameFields(t, reflect.TypeOf(qa.Selector{}), reflect.TypeOf(qatree.Selector{}))
	sameFields(t, reflect.TypeOf(qa.Node{}), reflect.TypeOf(qatree.Node{}))
}

func sameFields(t *testing.T, a, b reflect.Type) {
	t.Helper()
	if a.NumField() != b.NumField() {
		t.Fatalf("%s has %d fields, %s has %d", a, a.NumField(), b, b.NumField())
	}
	for i := range a.NumField() {
		fa, fb := a.Field(i), b.Field(i)
		if fa.Name != fb.Name || fa.Tag != fb.Tag {
			t.Errorf("%s.%s %q != %s.%s %q", a, fa.Name, fa.Tag, b, fb.Name, fb.Tag)
		}
		if ta, tb := elem(fa.Type), elem(fb.Type); ta.Kind() != tb.Kind() || (ta.Kind() != reflect.Struct && ta != tb) {
			t.Errorf("%s.%s is %s, %s.%s is %s", a, fa.Name, fa.Type, b, fb.Name, fb.Type)
		}
	}
}

func elem(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return t
}

func TestClientJSONRoundTrip(t *testing.T) {
	sel := qa.Text("~Send").In(qa.Class("Shell").In(qa.ID("main")))
	b, _ := json.Marshal(sel)
	var got qatree.Selector
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	want := qatree.Selector{Text: "~Send", Within: &qatree.Selector{Class: "Shell", Within: &qatree.Selector{ID: "main"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("selector: got %+v, want %+v", got, want)
	}

	checked := true
	n := qatree.Node{ID: "a", Class: "Button", Text: "x", Enabled: true, Visible: true, Focused: true, Bounds: [4]int{1, 2, 3, 4},
		Checked: &checked, Selected: []int{1}, Value: "v", Path: "0/1", Children: []qatree.Node{{Class: "Label", Path: "0/1/0"}}}
	b, _ = json.Marshal(n)
	var back qa.Node
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	b2, _ := json.Marshal(back)
	if string(b) != string(b2) {
		t.Errorf("node: %s != %s", b, b2)
	}
}
