package lexer

import (
	"strings"
	"unicode"
)

// composeArabic applies the canonical (NFC) compositions that matter for Arabic
// source text: an alef/waw/yeh followed by a combining hamza or madda becomes the
// precomposed letter. Without this, a decomposed "أ" (ا + U+0654) would lose its
// hamza when diacritics are stripped from identifiers.
//
// TODO: switch to golang.org/x/text/unicode/norm for full NFC once dependencies
// are vendored; this table covers every composition Arabic script needs.
func composeArabic(src []rune) []rune {
	out := make([]rune, 0, len(src))
	for _, r := range src {
		if n := len(out); n > 0 {
			if c, ok := compositions[[2]rune{out[n-1], r}]; ok {
				out[n-1] = c
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

var compositions = map[[2]rune]rune{
	{'ا', 'ٓ'}: 'آ',
	{'ا', 'ٔ'}: 'أ',
	{'ا', 'ٕ'}: 'إ',
	{'و', 'ٔ'}: 'ؤ',
	{'ي', 'ٔ'}: 'ئ',
	{'ە', 'ٔ'}: 'ۀ',
	{'ہ', 'ٔ'}: 'ۂ',
	{'ے', 'ٔ'}: 'ۓ',
}

// isIgnorableInName reports characters that are dropped from identifiers and
// keywords: tashkeel (harakat, shadda, sukun, tanween…), superscript alef,
// Quranic annotation marks and tatweel.
func isIgnorableInName(r rune) bool {
	switch {
	case r >= 0x064B && r <= 0x065F, r == 0x0670, r == 0x0640,
		r >= 0x06D6 && r <= 0x06DC, r >= 0x06DF && r <= 0x06E4,
		r == 0x06E7, r == 0x06E8, r >= 0x06EA && r <= 0x06ED:
		return true
	}
	return false
}

// normalizeName returns the canonical spelling of an identifier or keyword.
func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !isIgnorableInName(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// foldLetters collapses commonly confused letter forms. It is used only to
// suggest a keyword when a name looks like a misspelling; names are never
// matched in folded form.
func foldLetters(s string) string {
	return letterFolder.Replace(s)
}

var letterFolder = strings.NewReplacer("أ", "ا", "إ", "ا", "آ", "ا", "ة", "ه", "ى", "ي")

// isForbidden reports bidirectional-control and zero-width characters, which are
// rejected outside string literals to prevent "Trojan Source" attacks.
func isForbidden(r rune) bool {
	switch {
	case r == 0x200E, r == 0x200F, // LRM, RLM
		r >= 0x202A && r <= 0x202E, // LRE, RLE, PDF, LRO, RLO
		r >= 0x2066 && r <= 0x2069, // LRI, RLI, FSI, PDI
		r >= 0x200B && r <= 0x200D, // ZWSP, ZWNJ, ZWJ
		r == 0xFEFF,                // BOM / ZWNBSP (allowed only as the very first character)
		r == 0x061C:                // Arabic letter mark
		return true
	}
	return false
}

func isNameStart(r rune) bool {
	return r == '_' || (unicode.IsLetter(r) && r != 0x0640)
}

func isNamePart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.Is(unicode.Mn, r) ||
		isASCIIDigit(r) || isArabicDigit(r)
}

func isASCIIDigit(r rune) bool   { return r >= '0' && r <= '9' }
func isArabicDigit(r rune) bool  { return r >= '٠' && r <= '٩' } // U+0660–U+0669
func isPersianDigit(r rune) bool { return r >= '۰' && r <= '۹' } // U+06F0–U+06F9

// ToArabicDigits renders the ASCII digits in s as Arabic-Indic digits.
func ToArabicDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if isASCIIDigit(r) {
			return '٠' + (r - '0')
		}
		return r
	}, s)
}
