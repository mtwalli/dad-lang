"""Pygments lexer for the Ḍād (ضاد) programming language.

Used by the documentation site for syntax highlighting of ```dad code blocks.
Keep KEYWORDS in sync with internal/token/token.go.
"""

from pygments.lexer import RegexLexer, bygroups, include, words
from pygments.token import (Error, Comment, Keyword, Name, Number, Operator,
                            Punctuation, String, Text, Whitespace)

__all__ = ["DadLexer"]

# Letters, digits, underscore, tatweel and Arabic diacritics all continue a name.
_NAME_CHAR = r"\wـً-ٰٟ"
_B = rf"(?<![{_NAME_CHAR}])"   # start-of-word boundary that understands diacritics
_E = rf"(?![{_NAME_CHAR}])"    # end-of-word boundary

DECLARATIONS = ("ثابت", "متغير", "دالة", "نوع", "تعداد", "عقد", "امتداد",
                "استورد", "خاص", "خادم", "مكون", "حالة", "جدول", "مسار")
CONTROL = ("إذا", "وإلا", "طابق", "لكل", "في", "من", "إلى", "حتى", "بخطوة",
           "طالما", "أرجع", "توقف", "تابع", "حاول", "التقط", "ارفع",
           "و", "أو", "ليس", "هو", "ك")
CONSTANTS = ("صح", "خطأ", "عدم", "هذا")
TYPES = ("صحيح", "عشري", "نص", "منطقي", "فراغ")


class DadLexer(RegexLexer):
    name = "Dad"
    aliases = ["dad", "ضاد"]
    filenames = ["*.dad"]
    mimetypes = ["text/x-dad"]

    tokens = {
        "root": [
            (r"\s+", Whitespace),
            (r"//.*?$", Comment.Single),
            (r"/\*", Comment.Multiline, "comment"),
            (r'"""', String, "triple"),
            (r'"', String, "dq"),
            (r"«", String, "guillemet"),
            (words(DECLARATIONS, prefix=_B, suffix=_E), Keyword.Declaration),
            # «صفحة» is a keyword only before an address: صفحة "/المهام" = …
            (rf'{_B}صفحة(?=[ \t]*")', Keyword.Declaration),
            (words(CONTROL, prefix=_B, suffix=_E), Keyword),
            (words(CONSTANTS, prefix=_B, suffix=_E), Keyword.Constant),
            (words(TYPES, prefix=_B, suffix=_E), Keyword.Type),
            # Ḍād numbers use the digits 0-9 only; other digit forms are mistakes.
            (r"[0-9][0-9_]*\.[0-9][0-9_]*", Number.Float),
            (r"[0-9][0-9_]*", Number.Integer),
            (r"[\u0660-\u0669\u06F0-\u06F9\u066B\u066C]+", Error),
            (rf"([^\W\d][{_NAME_CHAR}]*)(\s*)(?=\()", bygroups(Name.Function, Text)),
            (rf"[^\W\d][{_NAME_CHAR}]*", Name),
            (r"[?؟][?؟]|[?؟]\.|=>|\.\.|[=!<>]=|[+\-*/]=|&&|\|\||[-+*/%=<>!?؟.:]", Operator),
            (r"[(){}\[\],،;؛]", Punctuation),
            (r".", Text),
        ],
        "comment": [
            (r"[^*]+", Comment.Multiline),
            (r"\*/", Comment.Multiline, "#pop"),
            (r"\*", Comment.Multiline),
        ],
        "escape": [
            (r"\\.", String.Escape),
            (r"\{", String.Interpol, "interp"),
        ],
        "dq": [
            include("escape"),
            (r'"', String, "#pop"),
            (r'[^"\\{]+', String),
        ],
        "triple": [
            include("escape"),
            (r'"""', String, "#pop"),
            (r'[^"\\{]+|"', String),
        ],
        "guillemet": [
            include("escape"),
            (r"»", String, "#pop"),
            (r"[^»\\{]+", String),
        ],
        "interp": [
            (r"\}", String.Interpol, "#pop"),
            include("root"),
        ],
    }
