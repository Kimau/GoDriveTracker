package stat

import "testing"

var ()

func TestWordCount(t *testing.T) {

	td := []struct {
		text        string
		wordCount   int
		uniqueWords int
	}{{`I think I may do a bit too much soemtimes...got a half written Vulkan renderer in
		  flight atm too.`, 20, 17},
		{`Some glory in their birth, some in their skill,
Some in their wealth, some in their body's force,
Some in their garments though new-fangled ill;
Some in their hawks and hounds, some in their horse;
And every humour hath his adjunct pleasure,
Wherein it finds a joy above the rest:
But these particulars are not my measure,
All these I better in one general best.
Thy love is better than high birth to me,
Richer than wealth, prouder than garments' cost,
Of more delight than hawks and horses be;
And having thee, of all men's pride I boast:
   Wretched in this alone, that thou mayst take
   All this away, and me most wretched make.`, 115, 75}}

	for i, v := range td {

		m, wc := wordCount(v.text)

		if wc != v.wordCount {
			t.Errorf("[%d] Word Count failed %d != %d", i, wc, v.wordCount)
		}
		if len(m) != v.uniqueWords {
			t.Errorf("[%d] Unique Count failed %d != %d", i, len(m), v.uniqueWords)
		}

		x := topWordPairFromMap(m, wc, -1, -1)
		t.Log(x)
	}

}
