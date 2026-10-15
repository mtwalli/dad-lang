"""MkDocs hook: add a content hash to our own CSS/JS links (e.g. dad.css?v=1a2b3c4d).

Browsers cache stylesheets and scripts, so without this, returning visitors can keep
an old dad.css after a deploy. The hash changes only when the file changes.
"""

import hashlib
import os


def _versioned(path: str, docs_dir: str) -> str:
    clean = path.split("?", 1)[0]
    full = os.path.join(docs_dir, clean)
    if not os.path.isfile(full):
        return path
    with open(full, "rb") as f:
        digest = hashlib.sha256(f.read()).hexdigest()[:8]
    return f"{clean}?v={digest}"


def on_config(config, **kwargs):
    docs_dir = config["docs_dir"]
    config["extra_css"] = [_versioned(p, docs_dir) for p in config["extra_css"]]
    for script in config["extra_javascript"]:
        if hasattr(script, "path"):
            script.path = _versioned(script.path, docs_dir)
    config["extra_javascript"] = [
        _versioned(s, docs_dir) if isinstance(s, str) else s
        for s in config["extra_javascript"]
    ]
    return config
