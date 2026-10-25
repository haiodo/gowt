"""Side-by-side sheets of the look-check shots: sheet.py <out-dir> <app>...

Reads <out-dir>/shots/<platform>/<app>.png and writes <out-dir>/<app>.png with the platforms that
have a shot, left to right in PLATFORMS order, scaled to one logical width so the text sizes compare.
"""
import sys
from pathlib import Path

from PIL import Image, ImageDraw

PLATFORMS = [("macos", "macOS"), ("windows", "Windows 11"), ("linux-vm", "Linux GNOME (VM)"), ("linux-docker", "Linux (Docker, Xvfb)")]
WIDTH = 1280  # logical width of the apps; a 2x or 1.5x capture is scaled down to it
GAP, HEAD = 16, 26


def main(out, apps):
    out = Path(out)
    for app in apps:
        cells = []
        for key, title in PLATFORMS:
            p = out / "shots" / key / f"{app}.png"
            if p.exists() and p.stat().st_size:
                im = Image.open(p).convert("RGB")
                px = im.width
                im = im.resize((WIDTH, round(im.height * WIDTH / im.width)), Image.LANCZOS)
                cells.append((f"{title}  ({px}px captured)", im))
        if not cells:
            print("no shots for", app)
            continue
        h = max(c[1].height for c in cells)
        sheet = Image.new("RGB", (len(cells) * WIDTH + (len(cells) + 1) * GAP, h + HEAD + GAP), (90, 90, 96))
        d = ImageDraw.Draw(sheet)
        for i, (title, im) in enumerate(cells):
            x = GAP + i * (WIDTH + GAP)
            d.text((x, 6), f"{app}: {title}", fill=(255, 255, 255))
            sheet.paste(im, (x, HEAD))
        sheet.save(out / f"{app}.png")
        print("sheet", out / f"{app}.png", len(cells), "platforms")


main(sys.argv[1], sys.argv[2:])
