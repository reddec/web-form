package utils

import (
	"strings"
)

func Quote(value string, q rune) string {
	var out strings.Builder
	QuoteBuilder(&out, value, q)
	return out.String()
}

func QuoteBuilder(out interface{ WriteRune(rune) (int, error) }, value string, q rune) {
	const escape = '\\'
	_, _ = out.WriteRune(q)
	for _, c := range value {
		if c == q || c == escape {
			_, _ = out.WriteRune(escape)
		}
		_, _ = out.WriteRune(c)
	}
	_, _ = out.WriteRune(q)
}
