package qatree

import (
	"encoding/json"
	"errors"

	"github.com/haiodo/gowt/swt"
)

func Dump(d *swt.Display) []Node { return nil }

func Find(d *swt.Display, s Selector) ([]Node, error) { return nil, errors.ErrUnsupported }

func Act(d *swt.Display, s Selector, action string, args json.RawMessage) error {
	return errors.ErrUnsupported
}
