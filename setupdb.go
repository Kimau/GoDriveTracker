package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	database "./database"
	google "./google"
	stat "./stat"
	web "./web"
	drive "google.golang.org/api/drive/v2" // DO NOT LIKE THIS! Want to encapse this in google package
)

func SetupDatabase(wf *web.WebFace, db *database.StatTrackerDB) {
	webBuf := bytes.NewBufferString("Starting Server")
	fileCounter := 0
	numFiles := 0

	outBuf := io.MultiWriter(webBuf, os.Stdout)

	fmt.Fprintf(outBuf, "Hosting on %s \n", *addr)

	wf.RedirectHandler = func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "Files Processed: %4d/%d \n", fileCounter, numFiles)
		fmt.Fprint(rw, webBuf)
	}

	fmt.Fprintln(outBuf, "Fetching File List")

	cPage := make(chan int)

	var driveFilelist []*drive.File
	go func() {
		var errDrv error
		driveFilelist, errDrv = google.AllFiles("mimeType = 'application/vnd.google-apps.document'", cPage)
		if errDrv != nil {
			log.Fatalln("File List Error:", errDrv)
		}
	}()

	for i := <-cPage; i != -1; i = <-cPage {
		fmt.Fprintf(outBuf, "Getting Page: %d \n", i)
	}

	// Handle per file
	docStatList := []*stat.DocStat{}
	numFiles = len(driveFilelist)
	for ifile, file := range driveFilelist {
		fileCounter = ifile

		db.WriteFile(file)
		dStat := FilePullCalc(file, db)

		db.WriteFileStats(dStat)

		fmt.Fprintf(outBuf, "Stats File Generated: %s... %s %s\n", file.Id[:6], dStat.LastMod[:10], file.Title)
		docStatList = append(docStatList, dStat)
	}

	// Generate Daily Stat
	CreateDailyStat(docStatList, outBuf, db)

	wf.RedirectHandler = nil
}

func CreateDailyStat(docStatList []*stat.DocStat, rw io.Writer, db *database.StatTrackerDB) {
	var dates = map[string]stat.DailyStat{}

	for i, fileStat := range docStatList {
		fmt.Fprintf(rw, "Collecting Daily Numbers %4d/%d \n", i+1, len(docStatList))

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

	// Slower but good test (and get sorting from DB)
	for _, v := range dates {
		db.WriteDailyStats(&v)
	}

	//
}

func FilePullCalc(file *drive.File, db *database.StatTrackerDB) *stat.DocStat {
	dStat := stat.DocStat{
		FileId:  file.Id,
		Title:   file.Title,
		LastMod: file.ModifiedDate,
	}

	// Get Revisions List
	revLists, errRev := google.AllRevisions(file.Id)

	if errRev != nil {
		log.Fatalln("Revision List Error:", errRev)
	}

	for _, r := range revLists {
		db.WriteRevision(file.Id, r)

		rStat := RevisionPullCalc(r)
		dStat.RevList = append(dStat.RevList, rStat)
	}

	return &dStat
}

func RevisionPullCalc(rev *drive.Revision) stat.RevStat {
	rBody, e := google.GetAuth(rev.ExportLinks["text/plain"])
	if e != nil {
		log.Fatalln("Failed to get text file", e.Error())
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(rBody.Body)
	bodyStr := buf.String()

	revStat := stat.RevStat{
		RevId:    rev.Id,
		UserName: rev.LastModifyingUserName,
		ModDate:  rev.ModifiedDate,
	}

	revStat.WordFreq, revStat.WordCount = stat.GetTopWords(bodyStr)

	return revStat
}
