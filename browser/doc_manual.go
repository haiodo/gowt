// Package browser is SWT's Browser widget (org.eclipse.swt.browser) for porting SWT code: the
// Browser class, its location, progress, title, status, window and authentication listeners
// and BrowserFunction. The engine is package webview (WKWebView, WebView2, WebKitGTK).
//
// New programs that need a web page use package webview directly. Known gaps against SWT's
// Browser are listed in https://github.com/haiodo/gowt/blob/main/docs/webview-browser.md.
package browser
