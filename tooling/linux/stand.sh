#!/bin/sh
# Starts Xvfb :99, openbox, x11vnc and noVNC (6080), then execs the arguments.
export DISPLAY=:99
if ! pgrep -x Xvfb >/dev/null; then
  Xvfb :99 -screen 0 1280x1024x24 >/tmp/xvfb.log 2>&1 &
  until xdpyinfo >/dev/null 2>&1 || [ -e /tmp/.X11-unix/X99 ]; do sleep 0.1; done
  openbox >/tmp/openbox.log 2>&1 &
  x11vnc -display :99 -forever -shared -nopw -quiet -rfbport 5900 >/tmp/x11vnc.log 2>&1 &
  websockify --web /usr/share/novnc 6080 localhost:5900 >/tmp/novnc.log 2>&1 &
fi
exec "$@"
