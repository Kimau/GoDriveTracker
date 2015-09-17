package main

import (
	"fmt"
	"html/template"
	"log"
	"math"
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
)

const (
	dateFormat = "2006-01-02"
)

func SetupWebFace(wf *web.WebFace, dbPtr *database.StatTrackerDB) {
	sh := SummaryHandle{db: dbPtr}
	sh.Setup()
	wf.Router.Handle("/", sh)

	wf.Router.Handle("/day/", DayHandle{db: dbPtr})
}

////////////////////////////////////////////////////////////////////////////////
// Summary Handle
type svgBox struct {
	X         int
	Y         int
	W         int
	H         int
	Classname string
}

type gPoint struct {
	Stat  *stat.DailyStat
	Boxes []svgBox
}

type SummaryHandle struct {
	db           *database.StatTrackerDB
	DayList      map[int]map[time.Month]map[int]*stat.DailyStat
	LatestGraph  []gPoint
	GridLines    []int
	GridDayLines []int
	GridWidth    int
	GridHeight   int
	GridHalf     int
}

func (sh *SummaryHandle) Setup() {
	sh.DayList = make(map[int]map[time.Month]map[int]*stat.DailyStat)

	// Sumary Setup
	d := sh.db.LoadNextDailyStat("")
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
			sh.SetDayListDay(prevDate, nil)
			prevDate = prevDate.AddDate(0, 0, 1)
		}

		// Setup Day
		sh.SetDayListDay(prevDate, d)
		prevDate = prevDate.AddDate(0, 0, 1)

		// Onto Next Date
		d = sh.db.LoadNextDailyStat(d.ModDate)
	}

	newDate := time.Now()
	for prevDate.Before(newDate) {
		sh.SetDayListDay(prevDate, nil)
		prevDate = prevDate.AddDate(0, 0, 1)
	}

	// Graph Setup
	sh.LatestGraph = []gPoint{}
	newDate = newDate.AddDate(0, 0, -100)

	sh.GridDayLines = []int{}
	sh.GridLines = []int{}
	sh.GridWidth = 800
	sh.GridHeight = 300
	sh.GridHalf = sh.GridHeight
	XStep := sh.GridWidth / 100

	yH := sh.GridHalf
	yStep := 100
	for y := 0; yH > 0; y += yStep {
		yH = sh.GridHalf - int(math.Pow(float64(y), 0.7))
		sh.GridLines = append(sh.GridLines, yH)

		if y == 1000 {
			yStep = 1000
		}
	}

	for i := 0; i < 100; i += 1 {
		d := sh.GetDayListDay(newDate)
		if d != nil {
			p := gPoint{
				Stat:  d,
				Boxes: []svgBox{},
			}

			{
				// Add Bar
				h := int(math.Pow(float64(d.WordAdd), 0.7))
				p.Boxes = append(p.Boxes, svgBox{X: i * XStep, Y: sh.GridHalf - h, W: XStep, H: h, Classname: "addBar"})

				// Sub Bar
				p.Boxes = append(p.Boxes, svgBox{X: i * XStep, Y: sh.GridHalf - h, W: XStep, H: int(math.Pow(float64(0-d.WordSub), 0.7)), Classname: "subBar"})
			}

			sh.LatestGraph = append(sh.LatestGraph, p)
		}

		if newDate.Weekday() == time.Monday {
			sh.GridDayLines = append(sh.GridDayLines, i*XStep)
		}

		newDate = newDate.AddDate(0, 0, 1)
	}

	fmt.Println("Setup Summary Handle")
}

func (sh *SummaryHandle) GetDayListDay(dateKey time.Time) *stat.DailyStat {
	var ok bool
	var year map[time.Month]map[int]*stat.DailyStat
	var month map[int]*stat.DailyStat

	yKey := -dateKey.Year()
	year, ok = sh.DayList[yKey]
	if !ok {
		return nil
	}

	mKey := dateKey.Month()
	month, ok = year[mKey]
	if !ok {
		return nil
	}

	return month[dateKey.Day()]
}

func (sh *SummaryHandle) SetDayListDay(dateKey time.Time, data *stat.DailyStat) {
	var ok bool
	var year map[time.Month]map[int]*stat.DailyStat
	var month map[int]*stat.DailyStat

	yKey := -dateKey.Year()
	year, ok = sh.DayList[yKey]
	if !ok {
		sh.DayList[yKey] = make(map[time.Month]map[int]*stat.DailyStat)
		year = sh.DayList[yKey]
	}

	mKey := dateKey.Month()
	month, ok = year[mKey]
	if !ok {
		year[mKey] = make(map[int]*stat.DailyStat)
		month = year[mKey]

		firstDay := time.Date(dateKey.Year(), dateKey.Month(), 1, 0, 0, 0, 0, time.UTC).Weekday()
		dayOff := 1
		for firstDay != time.Monday {
			dayOff -= 1
			month[dayOff] = nil
			if firstDay == time.Sunday {
				firstDay = time.Saturday
			} else {
				firstDay -= 1
			}
		}
	}

	month[dateKey.Day()] = data
}

func (sh SummaryHandle) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	sumTemp, err := template.ParseFiles("summary.html")
	if err != nil {
		log.Fatalln("Error parsing:", err)
	}
	e := sumTemp.Execute(rw, sh)
	if e != nil {
		log.Println("Error in Temp", e)
	}
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

type DayData struct {
	FullDate  string
	Stat      *stat.DailyStat
	WordTotal int
	DocList   []*stat.DocStat
	RevList   []*stat.RevStat
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

	date := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)

	shortDate := date.Format("2006-01-02")

	dayStat := dh.db.LoadDailyStats(shortDate)
	if dayStat == nil {
		fmt.Fprintf(rw, "No stats for %s", shortDate)
		return
	}

	sumTemp, err := template.ParseFiles("dailyStat.html")
	if err != nil {
		log.Fatalln("Error parsing:", err)
	}

	dList := []*stat.DocStat{}
	rList := []*stat.RevStat{}
	for k, v := range dayStat.FileRevs {
		file := dh.db.LoadFileStats(k)
		if file == nil {
			http.Error(rw, "Error finding file", 500)
		} else {
			for _, vRevID := range v {
				dList = append(dList, file)
				for _, vRev := range file.RevList {
					if vRev.RevId == vRevID {
						rList = append(rList, &vRev)
					}
				}
			}
		}
	}

	e := sumTemp.Execute(rw, DayData{
		FullDate:  date.Format("Monday, 2 Jan 2006"),
		Stat:      dayStat,
		WordTotal: dayStat.WordAdd + dayStat.WordSub,
		DocList:   dList,
		RevList:   rList,
	})
	if e != nil {
		log.Println("Error in Temp", e)
	}

}
