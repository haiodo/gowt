#!/usr/bin/env bash
# The README hero: a looping video of the live charts of examples/apps/monitor.
# Usage: tooling/demo-video.sh [record|assemble|all] <label>      (all is the default; macOS: tooling/demo-video.sh macos)
# record    builds the QA build of the monitor, shoots its window through the QA socket (tooling/demovideo, no desktop in
#           the frames) for WARMUP + LOOP seconds into $FRAMES. It needs a window server, so it runs where the app does.
# assemble  turns $FRAMES into $OUT/video/monitor-<label>.mp4 (H.264, yuv420p, faststart, native size) and
#           $OUT/img/monitor-<label>.webp (animated, WIDTH px wide). It needs ffmpeg and img2webp, no window.
# The loop is a hard cut of the take: the charts scroll, so the join is not a seam the eye looks for.
# The raw frames (about 3 GB) are deleted after assemble unless KEEP=1.
# Env: LOOP=20 FPS=15 WIDTH=1200 WARMUP=60 (the charts show the last minute) Q=70 (WebP quality)
#      OUT=docs FRAMES=bin/demo/<label> BIN=<prebuilt QA monitor> MINFPS=10.
# Run it from the repository root. See docs/screenshots.md.
set -euo pipefail
mode=all
case ${1:-} in record|assemble|all) mode=$1; shift ;; esac
label=${1:?usage: tooling/demo-video.sh [record|assemble|all] <label>}
loop=${LOOP:-20} fps=${FPS:-15} width=${WIDTH:-1200} warmup=${WARMUP:-60} q=${Q:-70}
out=${OUT:-docs}
frames=${FRAMES:-bin/demo/$label}
mkdir -p "$frames"
frames=$(cd "$frames" && pwd) # absolute: go test runs in the package directory
log=${LOG:-$frames/record.log}

record() {
	local bin=${BIN:-}
	mkdir -p "$frames"
	rm -f "$frames"/f-*.png "$frames"/f-*.rgba "$frames"/frames.txt "$frames"/raw.txt
	if [ -z "$bin" ]; then
		bin=$frames/Monitor # the process list of the app shows this name
		go build -tags gowtqa -o "$bin" ./examples/apps/monitor
	fi
	# Light appearance like the app shots of docs/screenshots.md.
	GOWT_DEMO=1 GOWT_APPEARANCE=light GOWT_DEMO_BIN=$(cd "$(dirname "$bin")" && pwd)/$(basename "$bin") GOWT_DEMO_OUT=$frames \
		GOWT_DEMO_WARMUP=$warmup GOWT_DEMO_SECONDS=$(awk "BEGIN{print $loop+0.5}") \
		go test -count=1 -v -run TestRecord ./tooling/demovideo 2>&1 | tee "$log"
	local got
	got=$(sed -n 's/.* frames in .*: \([0-9.]*\) fps.*/\1/p' "$log")
	echo "capture rate: $got fps"
	# Below MINFPS the motion is choppy: record the window with the OS instead (macOS: screencapture -v -R<window rect>).
	awk "BEGIN{exit !($got >= ${MINFPS:-10})}" || { echo "demo-video: QA shots too slow ($got fps)" >&2; exit 1; }
}

assemble() {
	local n=$(awk "BEGIN{printf \"%d\", $loop*$fps+0.5}")
	local work=$frames/loop
	rm -rf "$work"; mkdir -p "$work" "$out/video" "$out/img"
	# Constant rate; the first LOOP seconds of the take.
	# macOS frames are raw RGBA (raw.txt: size and rate), Linux and Windows ones are PNGs paced by frames.txt.
	local src=(-f concat -safe 0 -i "$frames/frames.txt") feed=(true)
	if [ -f "$frames/raw.txt" ]; then
		read -r size rate < "$frames/raw.txt"
		src=(-f rawvideo -pixel_format rgba -video_size "$size" -framerate "$rate" -i -)
		feed=(cat "$frames"/f-*.rgba)
	fi
	# ffmpeg stops reading after LOOP seconds, so cat dies of SIGPIPE: only the status of ffmpeg counts.
	set +o pipefail
	"${feed[@]}" | ffmpeg -v error -y "${src[@]}" -filter_complex "
		[0:v]fps=$fps,format=rgb24,trim=start_frame=0:end_frame=$n,setpts=PTS-STARTPTS[v]" -map '[v]' -r "$fps" "$work/%04d.png"
	[ "${PIPESTATUS[1]}" = 0 ] || { echo "demo-video: ffmpeg failed" >&2; exit 1; }
	set -o pipefail
	local mp4=$out/video/monitor-$label.mp4 webp=$out/img/monitor-$label.webp
	# Crop to even sides; the take may be odd (Linux frames are 1276x796).
	ffmpeg -v error -y -framerate "$fps" -i "$work/%04d.png" -vf "crop=trunc(iw/2)*2:trunc(ih/2)*2" \
		-c:v libx264 -crf 18 -preset slow -pix_fmt yuv420p -movflags +faststart "$mp4"
	mkdir -p "$work/small"
	ffmpeg -v error -y -framerate "$fps" -i "$work/%04d.png" -vf "scale=$width:-2:flags=lanczos" "$work/small/%04d.png"
	img2webp -loop 0 -d $((1000 / fps)) -lossy -q "$q" -m 6 "$work"/small/*.png -o "$webp" >/dev/null
	ls -l "$mp4" "$webp"
	[ "${KEEP:-}" = 1 ] || rm -f "$frames"/f-*.rgba
}

[ "$mode" = assemble ] || record
[ "$mode" = record ] || assemble
