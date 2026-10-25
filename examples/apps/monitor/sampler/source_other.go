//go:build !darwin && !linux && !windows

package sampler

func newSource() (source, error) { return nil, ErrUnsupported }
