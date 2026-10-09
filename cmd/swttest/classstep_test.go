package main

import (
	"testing"

	"github.com/haiodo/gowt/internal/junit"
)

func TestClassStep(t *testing.T) {
	ran := false
	res := classStep([]func(){func() { panic("no browser") }, func() { ran = true }})
	if res.status != "FAIL" || ran {
		t.Fatalf("panic: got %+v, second ran=%v", res, ran)
	}
	res = classStep([]func(){func() { panic(&junit.Skipped{Reason: "no edge"}) }})
	if res.status != "SKIP" || res.message != "no edge" {
		t.Fatalf("assumption: got %+v", res)
	}
	if res = classStep([]func(){func() {}}); res.status != "" {
		t.Fatalf("ok: got %+v", res)
	}
}
