package webview

import "github.com/haiodo/gowt/internal/cocoa"

func cookies(url string, done func([]Cookie, bool)) {
	cocoa.WKGetCookies(url, func(cs []cocoa.WKCookie) {
		out := make([]Cookie, len(cs))
		for i, c := range cs {
			out[i] = Cookie(c)
		}
		done(out, true)
	})
}

func setCookie(url, header string, done func(bool)) { cocoa.WKSetCookie(url, header, done) }

func clearSessionCookies(done func()) { cocoa.WKClearSessionCookies(done) }
