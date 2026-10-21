"""Keep old /ar/… links working.

Until 0.2 the Arabic pages lived under /ar/ and English at the root. Arabic is
now the main site, so each old /ar/<page>/ address gets a small page that
forwards to /<page>/, keeping any #anchor.
"""

import os

PAGE = """<!doctype html>
<html lang="ar" dir="rtl">
<head>
<meta charset="utf-8">
<title>ضاد</title>
<link rel="canonical" href="{url}">
<meta name="robots" content="noindex">
<meta http-equiv="refresh" content="0; url={target}">
<script>location.replace({target!r} + location.hash)</script>
</head>
<body><a href="{target}">{url}</a></body>
</html>
"""


def on_post_build(config, **kwargs):
    site = config["site_dir"]
    root = (config.get("site_url") or "/").rstrip("/") + "/"
    english = os.path.join(site, "en")
    if not os.path.isdir(english):
        return
    for folder, _, files in os.walk(english):
        if "index.html" not in files:
            continue
        rel = os.path.relpath(folder, english)
        rel = "" if rel == "." else rel.replace(os.sep, "/") + "/"
        if not os.path.exists(os.path.join(site, rel, "index.html")):
            continue
        out = os.path.join(site, "ar", rel)
        os.makedirs(out, exist_ok=True)
        depth = rel.count("/") + 1
        target = "../" * depth + rel
        with open(os.path.join(out, "index.html"), "w", encoding="utf-8") as f:
            f.write(PAGE.format(url=root + rel, target=target))
