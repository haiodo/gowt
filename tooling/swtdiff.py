#!/usr/bin/env python3
"""swtdiff.py expected.txt run1.log [run2.log]: status diff of swttest logs against an expected file."""
import re, sys

def logs(p):
    r = {}
    for l in open(p, encoding="utf-8", errors="replace"):
        m = re.match(r"(PASS|FAIL|SKIP) (\S+) \([\d.]+s\)(?:: (.*))?", l.rstrip("\n"))
        if m:
            r[m[2]] = (m[1].lower().replace("pass", "pass"), (m[3] or "")[:160])
    return r

exp = {}
for l in open(sys.argv[1], encoding="utf-8"):
    if l.startswith("#") or not l.strip():
        continue
    f = l.rstrip("\n").split("\t")
    exp[f[1]] = (f[0], f[2] if len(f) > 2 else "")
runs = [logs(p) for p in sys.argv[2:]]
names = sorted(set(exp) | set().union(*runs))
bad = lambda s: s in ("fail", "flaky")
for n in names:
    e = exp.get(n, ("-", ""))
    st = [r.get(n, ("-", ""))[0] for r in runs]
    if len(set(st)) > 1:
        print("RUNFLAKY", n, st, "exp", e[0])
    s = st[0]
    if s == e[0] or (e[0] == "flaky" and s in ("pass", "fail")):
        continue
    if bad(e[0]) and s == "pass":
        print("VMPASS_EXPFAIL", n, "exp", e[0])
    elif e[0] == "pass" and s == "fail":
        print("VMFAIL_EXPPASS", n, runs[0][n][1])
    else:
        print("OTHER", n, "exp", e[0], "vm", s)
for n in names:
    if n in exp and exp[n][0] == "fail" and runs[0].get(n, ("",))[0] == "fail":
        print("BOTHFAIL", n, "| exp:", exp[n][1][:70], "| vm:", runs[0][n][1][:90])
