// Package junit is the JUnit 5 surface the translated SWT tests (tests/swttests) use: the
// generated test registry and org.junit.jupiter.api.Assertions/Assumptions. A failed assertion
// panics *AssertionFailed, a failed assumption *Skipped; cmd/swttest recovers both per test.
package junit

import "time"

// Test is one test method (a @ParameterizedTest is expanded into one Test per argument).
type Test struct {
	Name    string
	Tags    []string
	Skip    string        // non-empty: not run, with this reason
	Timeout time.Duration // 0: the runner's default
	Run     func(instance any)
}

// Class is one test class: New creates a fresh instance per test, as JUnit's default lifecycle.
type Class struct {
	Name                  string
	New                   func() any
	BeforeAll, AfterAll   []func()
	BeforeEach, AfterEach []func(instance any)
	Tests                 []Test
}

// Classes holds every registered class, in registration (Go file init) order.
var Classes []*Class

func Register(c *Class) { Classes = append(Classes, c) }

// AssertionFailed is the panic value of a failed assertion or fail().
type AssertionFailed struct{ Message string }

func (e *AssertionFailed) Error() string { return e.Message }

// Skipped is the panic value of a failed assumption (assumeTrue/assumeFalse).
type Skipped struct{ Reason string }

func (e *Skipped) Error() string { return e.Reason }
