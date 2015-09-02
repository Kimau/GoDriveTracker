package main

import (
//	"encoding/json"
)

// Sort by Modified Date and Type
type WordPairByVol []WordPair

func (a WordPairByVol) Len() int      { return len(a) }
func (a WordPairByVol) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a WordPairByVol) Less(i, j int) bool {
	return (a[i].Count < a[j].Count)
}

type WordPair struct {
	Word  string `json:"Word"`
	Count int    `json:"Count"`
}

type RevStat struct {
	RevId     string     `json:"RevId"`
	WordCount int        `json:"WordCount"`
	ModDate   string     `json:"ModDate"`
	WordFreq  []WordPair `json:"WordFreq"`
}

type DocStat struct {
	FileId  string    `json:"FileId"`
	LastMod string    `json:"LastMod"`
	RevList []RevStat `json:"RevList"`
}
