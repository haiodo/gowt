package webview

import "github.com/haiodo/gowt/internal/webkit"

func cookies(url string, done func([]Cookie, bool)) {
	if webkit.Load() != nil {
		done(nil, false)
		return
	}
	webkit.GetCookies(url, func(cs []webkit.Cookie) {
		out := make([]Cookie, len(cs))
		for i, c := range cs {
			out[i] = Cookie(c)
		}
		done(out, true)
	})
}

func setCookie(url, header string, done func(bool)) {
	if webkit.Load() != nil {
		done(false)
		return
	}
	webkit.SetCookie(url, header, done)
}

func clearSessionCookies(done func()) {
	if webkit.Load() != nil {
		done()
		return
	}
	webkit.ClearSessionCookies(done)
}
