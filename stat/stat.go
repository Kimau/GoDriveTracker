package stat

import (
	"fmt"
	"time"
	//	"encoding/json"
)

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

func (rev RevStat) GetTime() string {
	x, _ := time.Parse("2006-01-02T15:04:05.000Z", rev.ModDate)
	return x.Format("15:04")
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
