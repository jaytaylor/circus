package textmanip

// Converts UTF-8 to close approximate ASCII form.

import (
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var txtNormalizer = transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)

func ToASCII(s string) string {
	s = strings.Map(normalizeChars, s)
	ascii, _, _ := transform.String(txtNormalizer, s)
	return ascii
}

func normalizeChars(in rune) rune {
	switch in {
	case '“', '‹', '”', '›':
		return '"'
	case '‘', '’':
		return '\''
	}
	return in
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}
