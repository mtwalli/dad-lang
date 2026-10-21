"""Split the search index by language.

mkdocs-static-i18n writes one search index for the whole site, so searching
the Arabic site also finds English pages (their code is Arabic too). This
hook keeps the English pages in search/search_index.json and moves the
Arabic ones to search/search_index.ar.json; overrides/main.html makes the
Arabic pages load that file.
"""

import json
import os

from mkdocs.plugins import event_priority


@event_priority(-100)  # after the search and i18n plugins
def on_post_build(config, **kwargs):
    path = os.path.join(config["site_dir"], "search", "search_index.json")
    if not os.path.exists(path):
        return
    with open(path, encoding="utf-8") as f:
        index = json.load(f)
    docs = index.get("docs", [])
    ar = [d for d in docs if d["location"].startswith("ar/")]
    if not ar:
        return  # already split, or the Arabic build hasn't run yet
    en = [d for d in docs if not d["location"].startswith("ar/")]
    for name, part, lang in (("search_index.json", en, "en"), ("search_index.ar.json", ar, "ar")):
        out = dict(index, docs=part, config=dict(index["config"], lang=[lang]))
        with open(os.path.join(os.path.dirname(path), name), "w", encoding="utf-8") as f:
            json.dump(out, f, ensure_ascii=False, separators=(",", ":"))
