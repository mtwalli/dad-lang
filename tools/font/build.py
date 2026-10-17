#!/usr/bin/env python3
"""Builds Ḍād Mono (family name "Dad Mono"), the coding font for Ḍād.

Arabic comes from Noto Kufi Arabic; Latin letters, digits and symbols come from a
monospaced font: JetBrains Mono when its files are in fonts-src/JetBrains_Mono,
otherwise DejaVu Sans Mono. The Arabic is scaled to sit with the Latin letters,
hidden direction characters get a visible glyph, and the result is renamed so
it can be installed next to the original fonts.

Usage (from the repository root):  python3 tools/font/build.py
Needs: fontTools (pip install fonttools).

Writes fonts/DadMono-{Regular,Bold}.{ttf,woff} and fonts/LICENSE.txt, and copies
them to docs/assets/fonts/ and editors/vscode/fonts/.
"""
import io
import pathlib
import shutil

from fontTools import subset
from fontTools.merge import Merger
from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTFont
from fontTools.ttLib.scaleUpem import scale_upem

ROOT = pathlib.Path(__file__).resolve().parents[2]
SRC = ROOT / "fonts-src"
OUT = ROOT / "fonts"
# Where the built files are copied, and which kinds each place needs.
COPIES = {
    ROOT / "docs" / "assets" / "fonts": (".ttf", ".woff", ".txt"),     # web font + downloads
    ROOT / "editors" / "vscode" / "fonts": (".ttf", ".txt"),          # installed by the extension
}

VERSION = "1.000"
FAMILY = "Dad Mono"

# Arabic is drawn a little smaller than its native size so its letter bodies match
# the height of Latin lowercase letters.
ARABIC_SCALE = 0.95

ARABIC_RANGES = [
    (0x0600, 0x06FF), (0x0750, 0x077F), (0x0870, 0x08FF),
    (0xFB50, 0xFDFF), (0xFE70, 0xFEFF), (0x200C, 0x200D), (0x25CC, 0x25CC),
]
# Characters that change text direction without being seen. Ḍād rejects them
# outside strings; the font draws them so nobody can hide them in code.
BIDI_CONTROLS = [0x061C, 0x200E, 0x200F, *range(0x202A, 0x202F), *range(0x2066, 0x206A)]


def in_arabic(cp):
    return any(a <= cp <= b for a, b in ARABIC_RANGES)


def latin_sources():
    jb = SRC / "JetBrains_Mono" / "static"
    if (jb / "JetBrainsMono-Regular.ttf").exists():
        return "JetBrains Mono", {"Regular": jb / "JetBrainsMono-Regular.ttf", "Bold": jb / "JetBrainsMono-Bold.ttf"}, \
            SRC / "JetBrains_Mono" / "OFL.txt"
    dv = SRC / "DejaVu_Sans_Mono"
    return "DejaVu Sans Mono", {"Regular": dv / "DejaVuSansMono.ttf", "Bold": dv / "DejaVuSansMono-Bold.ttf"}, \
        dv / "LICENSE.txt"


ARABIC_SOURCES = {
    "Regular": SRC / "Noto_Kufi_Arabic" / "static" / "NotoKufiArabic-Medium.ttf",
    "Bold": SRC / "Noto_Kufi_Arabic" / "static" / "NotoKufiArabic-Bold.ttf",
}


def subset_font(path, keep):
    font = TTFont(path)
    cmap = font.getBestCmap()
    opts = subset.Options()
    opts.layout_features = ["*"]
    opts.hinting = False
    opts.notdef_outline = True
    opts.name_IDs = ["*"]
    opts.name_languages = ["*"]
    opts.glyph_names = True
    opts.drop_tables += ["FFTM", "STAT", "DSIG"]
    sub = subset.Subsetter(opts)
    sub.populate(unicodes=[cp for cp in cmap if keep(cp)])
    sub.subset(font)
    return font


def to_bytes(font):
    buf = io.BytesIO()
    font.save(buf)
    buf.seek(0)
    return buf


def bidi_glyph(font, width, height):
    """A hollow box with a dot: visible, but plainly not a letter."""
    pen = TTGlyphPen(None)
    m = width // 6
    x0, x1, y0, y1 = m, width - m, -height // 6, height
    t = max(width // 14, 40)

    def rect(a, b, c, d, clockwise=True):
        pts = [(a, b), (a, d), (c, d), (c, b)] if clockwise else [(a, b), (c, b), (c, d), (a, d)]
        pen.moveTo(pts[0])
        for p in pts[1:]:
            pen.lineTo(p)
        pen.closePath()

    rect(x0, y0, x1, y1)                          # outer
    rect(x0 + t, y0 + t, x1 - t, y1 - t, False)   # hole
    cx, cy, r = width // 2, (y0 + y1) // 2, t * 1.4
    rect(int(cx - r), int(cy - r), int(cx + r), int(cy + r))  # dot
    glyph = pen.glyph()
    name = "dad.bidimark"
    font["glyf"][name] = glyph
    font["hmtx"][name] = (width, x0)
    font.setGlyphOrder(font.getGlyphOrder() + [name])
    for table in font["cmap"].tables:
        if table.isUnicode():
            for cp in BIDI_CONTROLS:
                if table.format != 4 or cp <= 0xFFFF:
                    table.cmap[cp] = name
    return name


def set_names(font, style):
    name = font["name"]
    name.names = []
    full = FAMILY if style == "Regular" else f"{FAMILY} {style}"
    ps = f"DadMono-{style}"
    records = {
        0: "Copyright 2026 The Ḍād authors. Arabic from Noto Kufi Arabic, copyright 2019-2022 Google LLC. "
           "Latin from DejaVu Sans Mono / Bitstream Vera, copyright 2003 Bitstream, Inc., or JetBrains Mono, "
           "copyright 2020 The JetBrains Mono Project Authors.",
        1: FAMILY,
        2: style,
        3: f"{VERSION};DAD;{ps}",
        4: full,
        5: f"Version {VERSION}",
        6: ps,
        11: "https://mtwalli.github.io/dad-lang/",
        13: "This Font Software is licensed under the SIL Open Font License, Version 1.1. "
            "The Latin glyphs remain under their original licenses; see LICENSE.txt.",
        14: "https://openfontlicense.org",
    }
    for nid, text in records.items():
        name.setName(text, nid, 3, 1, 0x409)   # Windows
        name.setName(text.replace("Ḍād", "Dad"), nid, 1, 0, 0)  # Mac (Mac Roman has no Ḍ)
    bold = style == "Bold"
    os2 = font["OS/2"]
    os2.usWeightClass = 700 if bold else 400
    os2.fsSelection = (os2.fsSelection & ~0b1100001) | (0b100000 if bold else 0b1000000)
    font["head"].macStyle = 1 if bold else 0
    font["post"].isFixedPitch = 1


def fix_metrics(font):
    """A coding-font line height (about 1.37 em) that still fits Arabic letters with marks.

    Kufi reserves room for tall mark stacks, which would make every line 1.8 em high.
    The line is sized for letters with their usual marks (أ, لإ, ـجـ); rare taller stacks
    may touch the next line, as in other coding fonts. The Windows clipping box
    (usWin*) still covers every glyph, so nothing is cut off.
    """
    upem = font["head"].unitsPerEm
    glyf = font["glyf"]
    y_max = y_min = 0
    for g in font.getGlyphOrder():
        glyph = glyf[g]
        if glyph.numberOfContours:
            glyph.recalcBounds(glyf)
            y_max, y_min = max(y_max, glyph.yMax), min(y_min, glyph.yMin)
    asc, desc = round(upem * 0.977), -round(upem * 0.39)
    hhea, os2 = font["hhea"], font["OS/2"]
    hhea.ascent, hhea.descent, hhea.lineGap = asc, desc, 0
    os2.sTypoAscender, os2.sTypoDescender, os2.sTypoLineGap = asc, desc, 0
    os2.usWinAscent, os2.usWinDescent = max(asc, y_max), max(-desc, -y_min)
    os2.fsSelection |= 1 << 7  # USE_TYPO_METRICS


def build(style, latin_path):
    latin = subset_font(latin_path, lambda cp: not in_arabic(cp) and cp not in BIDI_CONTROLS)
    upem = latin["head"].unitsPerEm
    arabic = subset_font(ARABIC_SOURCES[style], in_arabic)
    # Scale the Arabic outlines by (upem / its upem) * ARABIC_SCALE, then declare the
    # Latin font's units per em so both share one coordinate system.
    k = upem / arabic["head"].unitsPerEm * ARABIC_SCALE
    scale_upem(arabic, round(arabic["head"].unitsPerEm * k))
    arabic["head"].unitsPerEm = upem

    merged = Merger().merge([to_bytes(latin), to_bytes(arabic)])
    cell = latin["hmtx"]["space"][0] if "space" in latin["hmtx"].metrics else upem * 6 // 10
    bidi_glyph(merged, cell, int(upem * 0.55))
    set_names(merged, style)
    fix_metrics(merged)
    merged["head"].fontRevision = float(VERSION)
    # Fixed timestamps keep rebuilds byte-for-byte identical (2026-10-17).
    merged.recalcTimestamp = False
    merged["head"].created = merged["head"].modified = 3875040000

    OUT.mkdir(exist_ok=True)
    ttf = OUT / f"DadMono-{style}.ttf"
    merged.save(ttf)
    woff = TTFont(ttf, recalcTimestamp=False)
    woff.flavor = "woff"
    woff.save(OUT / f"DadMono-{style}.woff")
    return ttf


def main():
    latin_name, latin_paths, latin_license = latin_sources()
    built = [build(style, path) for style, path in latin_paths.items()]
    licence = (
        "Ḍād Mono (\"Dad Mono\")\n"
        "=====================\n\n"
        f"Arabic glyphs: Noto Kufi Arabic. Latin glyphs, digits and symbols: {latin_name}.\n"
        "Ḍād Mono is a modified version of these fonts, renamed as their licenses require.\n\n"
        "------------------------------------------------------------------------\n"
        "Noto Kufi Arabic — SIL Open Font License 1.1\n"
        "------------------------------------------------------------------------\n\n"
        + (SRC / "Noto_Kufi_Arabic" / "OFL.txt").read_text(encoding="utf-8")
        + "\n\n------------------------------------------------------------------------\n"
        f"{latin_name}\n"
        "------------------------------------------------------------------------\n\n"
        + latin_license.read_text(encoding="utf-8")
    )
    (OUT / "LICENSE.txt").write_text(licence, encoding="utf-8")
    for dest, kinds in COPIES.items():
        dest.mkdir(parents=True, exist_ok=True)
        for f in OUT.iterdir():
            if f.suffix in kinds:
                shutil.copy2(f, dest / f.name)
    for f in built:
        print(f"built {f.relative_to(ROOT)} (Latin: {latin_name})")


if __name__ == "__main__":
    main()
