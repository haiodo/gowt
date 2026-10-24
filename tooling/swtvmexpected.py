import re, sys
W = ""
D = ".work/winvm-results/"

def logs(p):
    r = {}
    order = []
    for l in open(p, encoding="utf-8", errors="replace"):
        m = re.match(r"(PASS|FAIL|SKIP) (\S+) \([\d.]+s\)(?:: (.*))?", l.rstrip("\n"))
        if m:
            msg = re.sub(r" at /Users\S*", "", m[3] or "")
            msg = msg.split(" ==> ")[0].strip()
            r[m[2]] = (m[1].lower(), msg)
            order.append(m[2])
    return r, order

runs = [logs(D + f"run{i}.log") for i in (1, 2, 3)]
order = runs[2][1]
CAUSE = ("window.gowt.call is answered as a subframe call on the VM: get_Source is about:blank for a NavigateToString page, "
         "the dialog Uri is the data: URL, sameDocument is false (webview/webview_windows.go:375), so the page gets an empty reply")
out = ["# Regression gate for make test-swt (cmd/swttest -expected). Recorded on real Windows 11 arm64 (UTM VM, build 26100, WebView2 Runtime 155), 3 full runs.",
       "# Runs 1-2: e6a6556, run 3: e91fc93 (WebView2 dialog fix 1057baa). A Browser test counts by run 3 where only the fix changed it; flaky = the runs disagree otherwise.",
       "# Compare with tests/expected_windows.txt (recorded under CrossOver/Wine) before using it as a gate."]
cnt = {}
for n in order:
    st = [r[0][n][0] if n in r[0] else "-" for r in runs]
    msgs = [r[0][n][1] if n in r[0] else "" for r in runs]
    browser = "browser_Browser" in n
    if browser and st[0] == st[1] == "fail" and st[2] == "pass":
        s = st[2]; why = msgs[2]; note = ""
    elif len(set(st)) == 1:
        s = st[0]; why = msgs[0]; note = ""
    else:
        s = "flaky"; why = next(m for m, x in zip(msgs, st) if x == "fail"); note = " (runs: " + "/".join(st) + ")"
    cnt[s] = cnt.get(s, 0) + 1
    if s == "pass":
        out.append(f"pass\t{n}")
        continue
    if s in ("fail", "flaky"):
        if any(k in n + why for k in ("BrowserFunction", "page did not answer", "is not defined", "Javascript", "javascript")):
            why = CAUSE + "; first message: " + why[:80]
        elif not why:
            why = "no message in the log (timeout or abort)"
        why = "VM: " + why
    out.append(f"{s}\t{n}\t{why}{note}")
open(W + "tests/expected_windows_vm.txt", "w").write("\n".join(out) + "\n")
print(cnt, len(order))
