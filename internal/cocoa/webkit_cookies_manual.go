//go:build darwin

package cocoa

import (
	"net/url"
	"strings"
)

// WKCookie is the part of a cookie the webview package reports.
type WKCookie struct{ Name, Value string }

// httpCookieStore is the store of the default data store, the one every view without a data store of its own uses.
func httpCookieStore() uintptr {
	wkSetup()
	return msg(msg(class("WKWebsiteDataStore"), "defaultDataStore"), "httpCookieStore")
}

func nsArray(arr uintptr) []uintptr {
	n := int(msg(arr, "count"))
	out := make([]uintptr, n)
	for i := range out {
		out[i] = msg(arr, "objectAtIndex:", uintptr(i))
	}
	return out
}

// WKGetCookies reports the cookies the store would send to rawURL; the store cannot filter, so this does.
func WKGetCookies(rawURL string, done func([]WKCookie)) {
	u, err := url.Parse(rawURL)
	if err != nil {
		done(nil)
		return
	}
	host, path, https := u.Hostname(), u.EscapedPath(), u.Scheme == "https"
	if path == "" {
		path = "/"
	}
	msg(httpCookieStore(), "getAllCookies:", NewBlock(1, func(a []uintptr) {
		var out []WKCookie
		for _, c := range nsArray(a[0]) {
			d := strings.TrimPrefix(goString(msg(c, "domain")), ".")
			cp := goString(msg(c, "path"))
			if (host == d || strings.HasSuffix(host, "."+d)) && strings.HasPrefix(path, cp) &&
				(https || msg(c, "isSecure")&0xff == 0) {
				out = append(out, WKCookie{goString(msg(c, "name")), goString(msg(c, "value"))})
			}
		}
		done(out)
	}))
}

// WKSetCookie stores header, the value of a Set-Cookie field, as if rawURL had sent it.
func WKSetCookie(rawURL, header string, done func(ok bool)) {
	nsurl := msg(class("NSURL"), "URLWithString:", nsString(rawURL))
	hdr := msg(class("NSDictionary"), "dictionaryWithObject:forKey:", nsString(header), nsString("Set-Cookie"))
	var cookies []uintptr
	if nsurl != 0 {
		cookies = nsArray(msg(class("NSHTTPCookie"), "cookiesWithResponseHeaderFields:forURL:", hdr, nsurl))
	}
	if len(cookies) == 0 {
		done(false)
		return
	}
	left := len(cookies)
	store := httpCookieStore()
	for _, c := range cookies {
		msg(store, "setCookie:completionHandler:", c, NewBlock(0, func([]uintptr) {
			if left--; left == 0 {
				done(true)
			}
		}))
	}
}

// WKClearSessionCookies deletes the cookies without an expiry date.
func WKClearSessionCookies(done func()) {
	store := httpCookieStore()
	msg(store, "getAllCookies:", NewBlock(1, func(a []uintptr) {
		var session []uintptr
		for _, c := range nsArray(a[0]) {
			if msg(c, "isSessionOnly")&0xff != 0 {
				session = append(session, c)
			}
		}
		if len(session) == 0 {
			done()
			return
		}
		left := len(session)
		for _, c := range session {
			msg(store, "deleteCookie:completionHandler:", c, NewBlock(0, func([]uintptr) {
				if left--; left == 0 {
					done()
				}
			}))
		}
	}))
}
