package gowt

import (
	"net/url"

	"github.com/haiodo/gowt/internal/dbus"
)

func singleInstance(id string, args []string) (bool, error) {
	c, err := dbus.Connect()
	if err != nil {
		return true, err
	}
	owner, err := c.RequestName(id)
	if err != nil {
		c.Close()
		return true, err
	}
	if owner {
		go func() { c.Serve(inboxPush); c.Close() }()
		return true, nil
	}
	defer c.Close()
	uris := make([]string, len(args))
	for i, p := range args {
		if openURL(p) {
			uris[i] = p
			continue
		}
		uris[i] = (&url.URL{Scheme: "file", Path: p}).String()
	}
	if err := c.Open(id, uris); err != nil {
		return true, err
	}
	return false, nil
}
