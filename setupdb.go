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
	dates := stat.CreateDailyUserStat(docStatList)

	// Slower but good test (and get sorting from DB)
	for _, v := range dates {
		db.WriteDailyUserStats(&v)
	}

	wf.RedirectHandler = nil
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
