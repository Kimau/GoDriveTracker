package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"./database"
	"./stat"
	"./web"
)

var (
	rePathMatch = regexp.MustCompile("/day/([0-9]+)[/\\-]([0-9]+)[/\\-]([0-9]+)")
	dayList     = map[int]map[int]map[int]*stat.DailyStat{}
)

const (
	dateFormat = "2006-01-02"
)

func SetDayListDay(dateKey time.Time, data *stat.DailyStat) {
	var ok bool
	var year map[int]map[int]*stat.DailyStat
	var month map[int]*stat.DailyStat

	yKey := -dateKey.Year()
	year, ok = dayList[yKey]
	if !ok {
		dayList[yKey] = make(map[int]map[int]*stat.DailyStat)
		year = dayList[yKey]
	}

	mKey := -int(dateKey.Month())
	month, ok = year[mKey]
	if !ok {
		year[mKey] = make(map[int]*stat.DailyStat)
		month = year[mKey]
	}

	month[dateKey.Day()] = data
}

func SetupWebFace(wf *web.WebFace, dbPtr *database.StatTrackerDB) {
	wf.Router.Handle("/", SummaryHandle{db: dbPtr})
	wf.Router.Handle("/day/", DayHandle{db: dbPtr})

	d := dbPtr.LoadNextDailyStat("")
	prevDate, dErr := time.Parse(dateFormat, d.ModDate)
	if dErr != nil {
		log.Fatalln("Cannot parse:", d)
	}

	for d != nil {
		// Skip Ahread Days
		newDate, nErr := time.Parse(dateFormat, d.ModDate)
		if nErr != nil {
			log.Fatalln("Cannot parse:", nErr, d)
		}

		for prevDate.Before(newDate) {
			SetDayListDay(prevDate, nil)
			prevDate = prevDate.AddDate(0, 0, 1)
		}

		// Setup Day
		SetDayListDay(prevDate, d)
		prevDate = prevDate.AddDate(0, 0, 1)

		// Onto Next Date
		d = dbPtr.LoadNextDailyStat(d.ModDate)
	}

	newDate := time.Now()
	for prevDate.Before(newDate) {
		SetDayListDay(prevDate, nil)
		prevDate = prevDate.AddDate(0, 0, 1)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Summary Handle
type SummaryHandle struct {
	db *database.StatTrackerDB
}

func (sh SummaryHandle) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	sumTemp, err := template.ParseFiles("summary.html")
	if err != nil {
		log.Fatalln("Error parsing:", err)
	}
	sumTemp.Execute(rw, dayList)
	/*
		for dayStat := sh.db.LoadNextDailyStat(""); dayStat != nil; dayStat = sh.db.LoadNextDailyStat(dayStat.ModDate) {
			fmt.Fprintf(rw, `<div><a href="/day/%s">%s</a> you wrote %d words and deleted %d words.</div>`, dayStat.ModDate, dayStat.ModDate, dayStat.WordAdd, dayStat.WordSub)
		}*/
}

////////////////////////////////////////////////////////////////////////////////
// Day Handle
type DayHandle struct {
	db *database.StatTrackerDB
}

func (dh DayHandle) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	matches := rePathMatch.FindAllStringSubmatch(req.URL.String(), -1)
	if len(matches) < 1 {
		http.Error(rw, "Invalid Path", 400)
		return
	}

	year, errYear := strconv.Atoi(matches[0][1])
	month, errMonth := strconv.Atoi(matches[0][2])
	day, errDate := strconv.Atoi(matches[0][3])

	if errYear != nil || errMonth != nil || errDate != nil {
		http.Error(rw, "Error parsing Path", 400)
		return
	}

	shortDate := fmt.Sprintf("%04d-%02d-%02d", year, month, day)

	dayStat := dh.db.LoadDailyStats(shortDate)
	if dayStat == nil {
		fmt.Fprintf(rw, "No stats for %s", shortDate)
		return
	}

	fmt.Fprintf(rw, "On %s you wrote %d words and deleted %d words.\n", dayStat.ModDate, dayStat.WordAdd, dayStat.WordSub)
	fmt.Fprintf(rw, "Edited %d files. \n", len(dayStat.FileRevs))
	for k, v := range dayStat.FileRevs {
		file := dh.db.LoadFile(k)
		if file == nil {
			fmt.Fprintf(rw, "> File [%s] not found [%d revisions] \n", k, len(v))
		} else {
			fmt.Fprintf(rw, "> %s  [%d revisions]\n", file.Title, len(v))
		}
	}
}
