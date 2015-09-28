package stat

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Sort by Modified Date and Type
type WordPairByVol []WordPair

func (a WordPairByVol) Len() int      { return len(a) }
func (a WordPairByVol) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a WordPairByVol) Less(i, j int) bool {
	return (a[i].Count > a[j].Count)
}

type WordPair struct {
	Word  string  `json:"Word"`
	Count int     `json:"Count"`
	Ratio float32 `json:"Ratio"`
}

func GetTopWords(s string) ([]WordPair, int) {
	m, wc := wordCount(s)
	return topWordPairFromMap(m, wc, 10, 3), wc
}

// WordCount returns a map of the counts of each “word” in the string s.
func wordCount(s string) (map[string]int, int) {

	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && (c != '`') && (c != '\'')
	}

	trimF := func(c rune) bool {
		return !unicode.IsLetter(c)
	}

	words := strings.FieldsFunc(s, f)

	counts := make(map[string]int, len(words))
	for _, word := range words {
		w := strings.ToLower(strings.TrimFunc(word, trimF))
		counts[w]++
	}

	return counts, len(words)
}

func topWordPairFromMap(wordMap map[string]int, wordCount int, numResults int, minFreq int) (wList []WordPair) {

	wcf := float32(wordCount)

	for k, v := range wordMap {
		if v >= minFreq {
			wList = append(wList, WordPair{Word: k, Count: v, Ratio: float32(v) / wcf})
		}
	}

	sort.Sort(WordPairByVol(wList))

	if numResults > 0 && len(wList) > numResults {
		wList = wList[:numResults]
	}

	return wList

}

func (wp WordPair) String() string {
	return fmt.Sprintf("%s:%d (%.2f)", wp.Word, wp.Count, wp.Ratio)
}
