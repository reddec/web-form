package utils_test

import (
	"testing"

	"github.com/reddec/web-form/internal/utils"
)

func TestQuote(t *testing.T) {
	cases := [][2]string{
		{"foo bar", "`foo bar`"},
		{"foo` bar", "`foo\\` bar`"},
		{"foo`` bar", "`foo\\`\\` bar`"},
		{"foo\\` bar", "`foo\\\\\\` bar`"},
	}

	for _, c := range cases {
		enc := utils.Quote(c[0], '`')
		if enc != c[1] {
			t.Errorf("%s != %s", enc, c[1])
		}
	}
}
