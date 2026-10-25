//go:build !linux

package gowt

import "github.com/haiodo/gowt/swt"

func installStyle() {}

func groupCard(*swt.Group) {}

func scrollOverlay(interface{ SetScrollbarsMode(int32) }) {}
