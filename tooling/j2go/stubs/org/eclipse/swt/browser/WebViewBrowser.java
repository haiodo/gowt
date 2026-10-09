package org.eclipse.swt.browser;

import org.eclipse.swt.*;
import org.eclipse.swt.widgets.*;

// Type information only (a reference file of port.sh, never emitted): the real class is
// browser/webbrowser_manual.go. It declares what that class overrides so WebBrowser gets its dispatch.
class WebViewBrowser extends WebBrowser {
	@Override public boolean back () { return false; }
	@Override public void create (Composite parent, int style) {}
	@Override public boolean execute (String script) { return false; }
	@Override public boolean forward () { return false; }
	@Override public String getBrowserType () { return null; }
	@Override public String getText () { return null; }
	@Override public String getUrl () { return null; }
	@Override public boolean isBackEnabled () { return false; }
	@Override public boolean isForwardEnabled () { return false; }
	@Override public void refresh () {}
	@Override public boolean setText (String html, boolean trusted) { return false; }
	@Override public boolean setUrl (String url, String postData, String[] headers) { return false; }
	@Override public void stop () {}
	@Override public boolean close () { return true; }
	@Override public void createFunction (BrowserFunction function) {}
	@Override public void destroyFunction (BrowserFunction function) {}
	@Override public Object evaluate (String script) throws SWTException { return null; }
	@Override public Object evaluate (String script, boolean trusted) throws SWTException { return null; }
	@Override public boolean isFocusControl () { return false; }
	@Override void nonBlockingExecute (String script) {}
	@Override String getJavaCallDeclaration () { return null; }
}
