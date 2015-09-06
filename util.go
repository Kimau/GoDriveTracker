package main

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"sync"

	database "./database"
	google "./google"
	stat "./stat"
	drive "google.golang.org/api/drive/v2" // DO NOT LIKE THIS! Want to encapse this in google package
)

func FullFileSweep(db *database.StatTrackerDB) error {

	for file := db.LoadNextFile(""); file != nil; file = db.LoadNextFile(file.Id) {
		dStat := stat.DocStat{FileId: file.Id, LastMod: file.ModifiedDate}

		for rev := db.LoadNextRevision(file.Id, ""); rev != nil; rev = db.LoadNextRevision(file.Id, rev.Id) {
			r, e := google.GetAuth(rev.ExportLinks["text/plain"])
			if e != nil {
				log.Println("Failed to get text file", e.Error())
				continue
			}

			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			bodyStr := buf.String()
			wCount, wTotal := WordCount(bodyStr)

			revStat := stat.RevStat{RevId: rev.Id, WordCount: wTotal, ModDate: rev.ModifiedDate}
			for k, v := range wCount {
				revStat.WordFreq = append(revStat.WordFreq, stat.WordPair{Word: k, Count: v})
			}
			sort.Sort(stat.WordPairByVol(revStat.WordFreq))
			dStat.RevList = append(dStat.RevList, revStat)

		}

		db.WriteFileStats(&dStat)
		log.Println("Stats File Generated:", file.Title, file.Id)
	}

	return nil
}

func GetFileRevsWriteDB(file *drive.File, db *database.StatTrackerDB) error {
	revLists, err := google.AllRevisions(file.Id)

	for _, rev := range revLists {
		go db.WriteRevision(file.Id, rev)
	}

	if err != nil {
		log.Fatalln("Failed to get File Revisions", err)
		return err
	}

	return nil
}

// Sort by Modified Date and Type
type ByTypeThenModMeDesc []*drive.File

func (a ByTypeThenModMeDesc) Len() int      { return len(a) }
func (a ByTypeThenModMeDesc) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTypeThenModMeDesc) Less(i, j int) bool {
	return (a[i].MimeType < a[j].MimeType) ||
		((a[i].MimeType == a[j].MimeType) && ((a[i].ModifiedByMeDate < a[j].ModifiedByMeDate) ||
			((a[i].ModifiedByMeDate == a[j].ModifiedByMeDate) && (a[i].ModifiedDate < a[j].ModifiedDate))))
}

func GetSortDriveList(db *database.StatTrackerDB) error {
	var files []*drive.File
	{
		var err error
		files, err = google.AllFiles("mimeType = 'application/vnd.google-apps.document'")
		if err != nil {
			log.Fatalln("Failed to get File List", err)
			return err
		}
	}

	sort.Sort(ByTypeThenModMeDesc(files))

	var wg sync.WaitGroup

	for _, v := range files {
		wg.Add(1)
		go func(f *drive.File) {
			db.WriteFile(f)
			GetFileRevsWriteDB(f, db)
			wg.Done()
		}(v)
	}
	// Waiting on Writes
	fmt.Println("Waiting on Web Requests...")
	wg.Wait()

	return nil
}

func LoadFileDumpStats(fileId string, db *database.StatTrackerDB) {
	f := db.LoadFile(fileId)
	if f == nil {
		fmt.Println("File not found:", fileId)
		return
	}

	fmt.Printf("%3d \tTitle: %s \n\t Last Mod: %s \n", f.Version, f.Title, f.ModifiedDate)

	r, e := google.GetAuth(f.ExportLinks["text/plain"])
	if e != nil {
		log.Println("Failed to get text file", e.Error())
	} else {

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		bodyStr := buf.String()
		wCount, total := WordCount(bodyStr)
		fmt.Printf("Word Count: %d \n Different Words: %d \n", total, len(wCount))

		/*
			fmt.Println("\n=========================================\n")

			fmt.Print(bodyStr)
			fmt.Println("\n=========================================\n")

			for k, v := range wCount {
				fmt.Println(k, ":", v)
			}*/
	}

	// Attempt to get revisions
	for rev := db.LoadNextRevision(fileId, ""); rev != nil; rev = db.LoadNextRevision(fileId, rev.Id) {
		r, e := google.GetAuth(rev.ExportLinks["text/plain"])
		if e != nil {
			log.Println("Failed to get text file", e.Error())
			continue
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		bodyStr := buf.String()
		wCount, total := WordCount(bodyStr)
		fmt.Printf("REV: %s %s \n Word Count: %d \n Different Words: %d \n", rev.Id, rev.ModifiedDate, total, len(wCount))
	}

}

func FullFileStatPrintout(db *database.StatTrackerDB) error {

	var dates = map[string]stat.DailyStat{}

	// Loading File mostly for debug info not smart
	// Might want to move over to LoadNextFileStat and putting more info in stat file??
	// Title is something that changes, but thats true for files as well
	for file := db.LoadNextFile(""); file != nil; file = db.LoadNextFile(file.Id) {
		fileStat := db.LoadFileStats(file.Id)

		fmt.Printf("%3d \tTitle: %s \n\t Last Mod: %s \n", file.Version, file.Title, file.ModifiedDate)
		prev := 0

		// Faster to do all dates then merge
		for _, v := range fileStat.RevList {
			shortDate := v.ModDate[:10]

			dv, ok := dates[shortDate]
			if !ok {
				dv = stat.DailyStat{
					WordAdd:  0,
					WordSub:  0,
					ModDate:  shortDate,
					FileRevs: map[string][]string{file.Id: {v.RevId}},
				}
			} else {
				revSubList, okFile := dv.FileRevs[file.Id]
				if !okFile {
					dv.FileRevs[file.Id] = []string{v.RevId}
				} else {
					dv.FileRevs[file.Id] = append(revSubList, v.RevId)
				}
			}

			diff := v.WordCount - prev
			if diff >= 0 {
				dv.WordAdd = dv.WordAdd + diff
			} else {
				dv.WordSub = dv.WordSub + diff
			}
			dates[shortDate] = dv

			fmt.Printf("%s \t Word Count: %d (%+d) \t Different Words: %d \n", v.ModDate, v.WordCount, diff, len(v.WordFreq))
			prev = v.WordCount
		}
	}

	/* OVERWRITE FOR NOW
	for k, v := range dates {
		oldDay := LoadDailyStats(k)
		if(oldDay == nil) {
			WriteDailyStats(v)
			} else {
				oldDay.AddDay(k)
			}
		fmt.Println(k, v.add, v.sub)
	}
	*/

	// Slower but good test (and get sorting from DB)
	for _, v := range dates {
		db.WriteDailyStats(&v)
	}
	for day := db.LoadNextDailyStat(""); day != nil; day = db.LoadNextDailyStat(day.ModDate) {
		fmt.Printf("%s %d:%d  %d \n", day.ModDate, day.WordAdd, day.WordSub, len(day.FileRevs))
	}

	return nil
}
