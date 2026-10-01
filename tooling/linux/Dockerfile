# Linux GUI stand for the gtk port. Native arm64 on Apple silicon; on amd64 hosts nothing
# changes (all packages are multi-arch), only pass --platform linux/amd64 when cross-building.
FROM golang:1.27-trixie

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config git ca-certificates curl procps \
    libgtk-3-dev libglib2.0-dev libcairo2-dev libpango1.0-dev libatk1.0-dev libgdk-pixbuf-2.0-dev \
    gobject-introspection libgirepository1.0-dev \
    gir1.2-gtk-3.0 gir1.2-gdkpixbuf-2.0 gir1.2-pango-1.0 gir1.2-atk-1.0 \
    gir1.2-gtk-4.0 gir1.2-graphene-1.0 \
    python3-gi python3-gi-cairo \
    xvfb openbox x11vnc novnc websockify x11-apps scrot \
    weston \
    && rm -rf /var/lib/apt/lists/*

# GLib/GObject/Gio/cairo GIRs live in libglib2.0-dev / libcairo2-dev; gir1.2-* covers the rest.
ENV DISPLAY=:99 GOFLAGS=-buildvcs=false GOCACHE=/gocache GOMODCACHE=/gomod
COPY stand.sh /usr/local/bin/stand.sh
WORKDIR /src
CMD ["stand.sh", "sleep", "infinity"]
