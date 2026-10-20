"""MkDocs hook: the Arabic tutorial (docs/tutorial/) reads right to left.

The rest of the site is in English, so the direction is set for those pages only:
their content is wrapped in <div class="dad-rtl" dir="rtl" lang="ar">.
"""


def on_page_content(html, page, **kwargs):
    if page.file.src_uri.startswith("tutorial/"):
        return f'<div class="dad-rtl" dir="rtl" lang="ar">{html}</div>'
    return html
