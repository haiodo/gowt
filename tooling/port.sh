#!/usr/bin/env bash
# Runs j2go over the stage-1 file list and writes generated Go into swt/.
set -euo pipefail
cd "$(dirname "$0")/.."

SWT_REPO="${SWT_REPO:-/Users/haiodo/Develop/repos/eclipse.platform.swt}"

mvn -q -f tooling/j2go/pom.xml package

java -jar tooling/j2go/target/j2go.jar --swt "$SWT_REPO" --out swt \
	org/eclipse/swt/graphics/Point.java \
	org/eclipse/swt/graphics/Rectangle.java \
	org/eclipse/swt/graphics/RGB.java \
	org/eclipse/swt/graphics/RGBA.java

gofmt -w swt/*.go
