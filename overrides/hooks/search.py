"""Split the search index by language.

mkdocs-static-i18n writes one search index for the whole site, so searching
one language also finds the other's pages (both have Arabic code). This hook
keeps the default language (Arabic, at the site root) in
search/search_index.json and moves each other language's pages, which live
under /<lang>/, to search/search_index.<lang>.json. overrides/main.html makes
those pages load their own file.
"""

import json
import os

from mkdocs.plugins import event_priority

OTHER_LANGUAGES = ("en",)


@event_priority(-100)  # after the search and i18n plugins
def on_post_build(config, **kwargs):
    folder = os.path.join(config["site_dir"], "search")
    path = os.path.join(folder, "search_index.json")
    if not os.path.exists(path):
        return
    with open(path, encoding="utf-8") as f:
        index = json.load(f)
    docs = index.get("docs", [])
    parts = {lang: [d for d in docs if d["location"].startswith(lang + "/")] for lang in OTHER_LANGUAGES}
    if not any(parts.values()):
        return  # already split, or the other builds haven't run yet
    taken = {id(d) for part in parts.values() for d in part}
    parts[""] = [d for d in docs if id(d) not in taken]
    for lang, part in parts.items():
        name = f"search_index.{lang}.json" if lang else "search_index.json"
        langs = [lang] if lang else [l for l in index["config"]["lang"] if l not in OTHER_LANGUAGES] or ["ar"]
        out = dict(index, docs=part, config=dict(index["config"], lang=langs))
        with open(os.path.join(folder, name), "w", encoding="utf-8") as f:
            json.dump(out, f, ensure_ascii=False, separators=(",", ":"))
