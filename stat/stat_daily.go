package stat

import (
	"fmt"
)

type DailyUserStat struct {
	WordAdd  int                 `json:"WordAdd"`
	WordSub  int                 `json:"WordSub"`
	ModDate  string              `json:"ModDate"`
	FileRevs map[string][]string `json:"FileRevList"`
}

func (day DailyUserStat) String() string {
	return fmt.Sprintf("[%s] Words %d / %d with following edits { %s }", day.ModDate, day.WordAdd, day.WordSub, day.FileRevs)
}

func CreateDailyUserStat(docStatList []*DocStat) (dates map[string]DailyUserStat) {

	dates = make(map[string]DailyUserStat)

	for _, fileStat := range docStatList {
		prev := 0

		// Faster to do all dates then merge
		for _, v := range fileStat.RevList {
			shortDate := v.ModDate[:10]

			dv, ok := dates[shortDate]
			if !ok {
				dv = DailyUserStat{
					WordAdd:  0,
					WordSub:  0,
					ModDate:  shortDate,
					FileRevs: map[string][]string{fileStat.FileId: {v.RevId}},
				}
			} else {
				revSubList, okFile := dv.FileRevs[fileStat.FileId]
				if !okFile {
					dv.FileRevs[fileStat.FileId] = []string{v.RevId}
				} else {
					dv.FileRevs[fileStat.FileId] = append(revSubList, v.RevId)
				}
			}

			diff := v.WordCount - prev
			if diff >= 0 {
				dv.WordAdd = dv.WordAdd + diff
			} else {
				dv.WordSub = dv.WordSub + diff
			}
			dates[shortDate] = dv

			prev = v.WordCount
		}
	}

	return dates
}
