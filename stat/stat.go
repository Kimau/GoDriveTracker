package stat

import (
	"fmt"
	"time"
	//	"encoding/json"
)

// Sort by Modified Date and Type
type WordPairByVol []WordPair

func (a WordPairByVol) Len() int      { return len(a) }
func (a WordPairByVol) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a WordPairByVol) Less(i, j int) bool {
	return (a[i].Count > a[j].Count)
}

type WordPair struct {
	Word  string `json:"Word"`
	Count int    `json:"Count"`
}

type RevStat struct {
	RevId     string     `json:"RevId"`
	UserName  string     `json:"UserName"`
	WordCount int        `json:"WordCount"`
	ModDate   string     `json:"ModDate"`
	WordFreq  []WordPair `json:"WordFreq"`
}

type DocStat struct {
	FileId  string    `json:"FileId"`
	Title   string    `json:"Title"`
	LastMod string    `json:"LastMod"`
	RevList []RevStat `json:"RevList"`
}

type DailyStat struct {
	WordAdd  int                 `json:"WordAdd"`
	WordSub  int                 `json:"WordSub"`
	ModDate  string              `json:"ModDate"`
	FileRevs map[string][]string `json:"FileRevList"`
}

type UserStat struct {
	UpdateDate string `json:"UpdateDate"`
	Token      []byte `json:Token`
	Email      string `json:Email`
	UserID     string `json:Id`
}

func (rev RevStat) GetTime() string {
	x, _ := time.Parse("2006-01-02T15:04:05.000Z", rev.ModDate)
	return x.Format("15:04")
}

func (wp WordPair) String() string {
	return fmt.Sprintf("%s:%d", wp.Word, wp.Count)
}
func (rev RevStat) String() string {
	return fmt.Sprintf("[%s %s] %d words by %s. \n\t Words [%s]", rev.ModDate, rev.RevId, rev.WordCount, rev.UserName, rev.WordFreq)
}
func (doc DocStat) String() string {
	s := fmt.Sprintf("[%s] '%s' last mod on %s with revs\n", doc.FileId, doc.Title, doc.LastMod)
	for i, v := range doc.RevList {
		s += fmt.Sprintf("\t %d:%s\n", i, v)
	}
	return s
}
func (day DailyStat) String() string {
	return fmt.Sprintf("[%s] Words %d / %d with following edits { %s }", day.ModDate, day.WordAdd, day.WordSub, day.FileRevs)
}
func (usr *UserStat) String() string {
	return fmt.Sprintf("[%s] %s last updated on %s (TOKEN HIDDEN)", usr.UserID, usr.Email, usr.UpdateDate)
}
