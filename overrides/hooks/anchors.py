"""MkDocs hook: readable anchors for Arabic headings.

Python-Markdown makes anchors from ASCII only, so an Arabic heading such as
«## أول برنامج» gets an anchor like «_1». On the Arabic pages this hook gives such
headings an anchor made of their own words («أول-برنامج»), in the page and in its table
of contents. Headings with an explicit anchor ({ #formatting }) keep it, so links shared
between the English and Arabic pages keep working.
"""

import re

GENERATED = re.compile(r"^_\d+$")
TASHKEEL = re.compile(r"[ً-ٰٟـ]")
NOT_WORD = re.compile(r"[^\w\s-]", re.UNICODE)


def slug(text):
    text = TASHKEEL.sub("", text)
    text = NOT_WORD.sub("", text).strip().lower()
    return re.sub(r"[\s_]+", "-", text)


def on_page_content(html, page, config, **kwargs):
    if config["theme"].get("language") != "ar":
        return html
    used = set()

    def walk(items):
        nonlocal html
        for item in items:
            if GENERATED.match(item.id or ""):
                new = slug(item.title) or item.id
                base, n = new, 2
                while new in used:
                    new, n = f"{base}-{n}", n + 1
                html = html.replace(f'id="{item.id}"', f'id="{new}"').replace(
                    f'href="#{item.id}"', f'href="#{new}"'
                )
                item.id = new
            used.add(item.id)
            walk(item.children)

    walk(page.toc)
    return html
