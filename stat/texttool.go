package stat

import (
	"strings"
	"unicode"
)

// WordCount returns a map of the counts of each “word” in the string s.
func WordCount(s string) (map[string]int, int) {

	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && (c != '`') && (c != '\'') && (c != '-')
	}

	trimF := func(c rune) bool {
		return !unicode.IsLetter(c)
	}

	words := strings.FieldsFunc(s, f)

	counts := make(map[string]int, len(words))
	for _, word := range words {
		w := strings.ToLower(strings.TrimFunc(word, trimF))
		if len(w) > 3 {
			counts[w]++
		}
	}

	return counts, len(words)
}
