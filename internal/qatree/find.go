package qatree

import (
	"fmt"
	"strings"

	"github.com/haiodo/gowt/swt"
)

func (s Selector) String() string {
	var p []string
	if s.ID != "" {
		p = append(p, "id="+s.ID)
	}
	if s.Class != "" {
		p = append(p, "class="+s.Class)
	}
	if s.Text != "" {
		p = append(p, "text="+s.Text)
	}
	out := "{" + strings.Join(p, " ") + "}"
	if s.Within != nil {
		out += " within " + s.Within.String()
	}
	return out
}

// matches compares Text with the text and the value of the node. A Text that starts with "~"
// matches a part of either, so a literal text cannot start with it.
func (s Selector) matches(n *Node) bool {
	if s.ID != "" && s.ID != n.ID || s.Class != "" && s.Class != n.Class {
		return false
	}
	switch {
	case s.Text == "":
	case strings.HasPrefix(s.Text, "~"):
		part := s.Text[1:]
		return strings.Contains(n.Text, part) || strings.Contains(n.Value, part)
	default:
		return s.Text == n.Text || s.Text == n.Value
	}
	return true
}

func collect(e *entry, s Selector, out *[]*entry) {
	if s.matches(&e.n) {
		*out = append(*out, e)
	}
	for _, k := range e.kids {
		collect(k, s, out)
	}
}

// resolve finds the entries s picks. Within must pick one entry, and only its descendants match.
func resolve(roots []*entry, s Selector) ([]*entry, error) {
	var out []*entry
	if s.Within == nil {
		for _, r := range roots {
			collect(r, s, &out)
		}
		return out, nil
	}
	in, err := resolve(roots, *s.Within)
	if err != nil {
		return nil, err
	}
	if len(in) != 1 {
		return nil, fmt.Errorf("qatree: within %s matches %d controls%s", s.Within, len(in), paths(in))
	}
	for _, k := range in[0].kids {
		collect(k, s, &out)
	}
	return out, nil
}

func paths(es []*entry) string {
	if len(es) == 0 {
		return ""
	}
	var p []string
	for _, e := range es {
		p = append(p, fmt.Sprintf("%s (%s %q)", e.n.Path, e.n.Class, e.n.Text))
	}
	return ": " + strings.Join(p, ", ")
}

// Find lists the nodes s picks, subtrees included. No match is an empty list, not an error.
func Find(d *swt.Display, s Selector) ([]Node, error) {
	es, err := resolve(walk(d), s)
	var out []Node
	for _, e := range es {
		out = append(out, toNode(e))
	}
	return out, err
}
