package main

import (
	"log"
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

type DailyStat struct {
	WordAdd  int                 `json:"WordAdd"`
	WordSub  int                 `json:"WordSub"`
	ModDate  string              `json:"ModDate"`
	FileRevs map[string][]string `json:"FileRevList"`
}

func (day *DailyStat) AddFile(newDoc *DocStat) {

	revSubList, okFile := day.FileRevs[newDoc.FileId]

	// Word Changes
	prev := 0
	for _, v := range newDoc.RevList {
		shortDate := v.ModDate[:10]

		if shortDate == day.ModDate {
			diff := v.WordCount - prev
			if diff >= 0 {
				day.WordAdd = day.WordAdd + diff
			} else {
				day.WordSub = day.WordSub + diff
			}

			if !okFile {
				day.FileRevs[newDoc.FileId] = []string{v.RevId}
				okFile = true
			} else {
				day.FileRevs[newDoc.FileId] = append(revSubList, v.RevId)
			}

			break
		}

		prev = v.WordCount
	}
}

func (day *DailyStat) AddDay(newDay *DailyStat) {
	if day.ModDate != newDay.ModDate {
		log.Fatalln("Dates must match", day.ModDate, newDay.ModDate)
		return
	}

	day.WordAdd += newDay.WordAdd
	day.WordSub += newDay.WordSub

	// File List
	for fk, fv := range newDay.FileRevs {
		revSubList, okFile := day.FileRevs[fk]
		if !okFile {
			day.FileRevs[fk] = fv
		} else {
			day.FileRevs[fk] = append(revSubList, fv...)
		}
	}

}

func FullFileSweep() error {

	for file := LoadNextFile(""); file != nil; file = LoadNextFile(file.Id) {
		GenerateStatsFile(file)
		log.Println(file.Id)
	}

	return nil
}
