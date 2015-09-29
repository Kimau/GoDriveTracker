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
	reDayPathMatch  = regexp.MustCompile("/day/([0-9]+)[/\\-]([0-9]+)[/\\-]([0-9]+)")
	reFilePathMatch = regexp.MustCompile("/file/([^/]+)")
)

const (
	dateFormat     = "2006-01-02"
	dateFormatLong = "2006-01-02T15:04:05.000Z"
)

func SetupWebFace(wf *web.WebFace, dbPtr *database.StatTrackerDB) {
	sh := SummaryHandle{db: dbPtr}
	sh.Setup()
	wf.Router.Handle("/", sh)
	wf.Router.Handle("/day/", DayHandle{db: dbPtr})
	wf.Router.Handle("/file/", FileHandle{db: dbPtr})
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
	Stat  *stat.DailyUserStat
	X, Y  int
	Boxes []svgBox
}

type SummaryHandle struct {
	db           *database.StatTrackerDB
	DayList      map[int]map[time.Month]map[int]*stat.DailyUserStat
	LatestGraph  []gPoint
	GridLines    []int
	GridDayLines []int
	GridWidth    int
	GridHeight   int
}

func (sh *SummaryHandle) Setup() {
	sh.DayList = make(map[int]map[time.Month]map[int]*stat.DailyUserStat)

	// Sumary Setup
	d := sh.db.LoadNextDailyUserStat("")
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
		d = sh.db.LoadNextDailyUserStat(d.ModDate)
	}

	newDate := time.Now()
	for prevDate.Before(newDate) {
		sh.SetDayListDay(prevDate, nil)
		prevDate = prevDate.AddDate(0, 0, 1)
	}

	// Graph Setup
	sh.LatestGraph = []gPoint{}
	firstDay := newDate.AddDate(0, 0, -100)

	sh.GridDayLines = []int{}
	sh.GridLines = []int{}
	sh.GridWidth = 800
	sh.GridHeight = 300
	XStep := sh.GridWidth / 100

	yH := sh.GridHeight
	yStep := 100
	for y := 0; yH > 0; y += yStep {
		yH = sh.GridHeight - int(math.Pow(float64(y), 0.7))
		sh.GridLines = append(sh.GridLines, yH)

		if y == 1000 {
			yStep = 1000
		}
	}

	newDate = firstDay
	dateList := make([]time.Time, 100)
	dayList := make([]*stat.DailyUserStat, 100)
	for i := 0; i < 100; i += 1 {
		dateList[i] = newDate
		dayList[i] = sh.GetDayListDay(newDate)
		newDate = newDate.AddDate(0, 0, 1)
	}

	dailyWordHistChart("./static/days.png", 700, 400, dayList, dateList)

	for i, day := range dateList {
		d := sh.GetDayListDay(day)
		if d != nil {
			h := int(math.Pow(float64(d.WordAdd), 0.7))

			p := gPoint{
				X:     i * XStep,
				Y:     sh.GridHeight - h,
				Stat:  d,
				Boxes: []svgBox{},
			}

			p.Boxes = append(p.Boxes, svgBox{X: i * XStep, Y: sh.GridHeight - h, W: XStep, H: h, Classname: "addBar"})
			p.Boxes = append(p.Boxes, svgBox{X: i * XStep, Y: sh.GridHeight - h, W: XStep, H: int(math.Pow(float64(0-d.WordSub), 0.7)), Classname: "subBar"})

			if p.Y < 0 {
				p.Y = 30
			}

			sh.LatestGraph = append(sh.LatestGraph, p)
		}

		if dateList[i].Weekday() == time.Monday {
			sh.GridDayLines = append(sh.GridDayLines, i*XStep)
		}
	}

	fmt.Println("Setup Summary Handle")
}

func (sh *SummaryHandle) GetDayListDay(dateKey time.Time) *stat.DailyUserStat {
	var ok bool
	var year map[time.Month]map[int]*stat.DailyUserStat
	var month map[int]*stat.DailyUserStat

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

func (sh *SummaryHandle) SetDayListDay(dateKey time.Time, data *stat.DailyUserStat) {
	var ok bool
	var year map[time.Month]map[int]*stat.DailyUserStat
	var month map[int]*stat.DailyUserStat

	yKey := -dateKey.Year()
	year, ok = sh.DayList[yKey]
	if !ok {
		sh.DayList[yKey] = make(map[time.Month]map[int]*stat.DailyUserStat)
		year = sh.DayList[yKey]
	}

	mKey := dateKey.Month()
	month, ok = year[mKey]
	if !ok {
		year[mKey] = make(map[int]*stat.DailyUserStat)
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
	sumTemp, err := template.ParseFiles("./templates/summary.html")
	if err != nil {
		log.Fatalln("Error parsing:", err)
	}
	e := sumTemp.Execute(rw, sh)
	if e != nil {
		log.Println("Error in Temp", e)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Day Handle
type DayHandle struct {
	db *database.StatTrackerDB
}

type DayData struct {
	FullDate  string
	Stat      *stat.DailyUserStat
	WordTotal int
	DocList   []*stat.DocStat
	RevList   []*stat.RevStat
}

func (dh DayHandle) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	matches := reDayPathMatch.FindAllStringSubmatch(req.URL.String(), -1)
	if len(matches) < 1 {
		http.Error(rw, "Invalid Path", 400)
		return
	}

	year, errYear := strconv.Atoi(matches[0][1])
	month, errMonth := strconv.Atoi(matches[0][2])
	day, errDate := strconv.Atoi(matches[0][3])

	if errYear != nil || errMonth != nil || errDate != nil {
		http.Error(rw, fmt.Sprintf("Error parsing Path: \n %s \n %s \n %s", errYear, errMonth, errDate), 400)
		return
	}

	date := time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.UTC)

	shortDate := date.Format("2006-01-02")

	dayStat := dh.db.LoadDailyUserStats(shortDate)
	if dayStat == nil {
		fmt.Fprintf(rw, "No stats for %s", shortDate)
		return
	}

	sumTemp, err := template.ParseFiles("./templates/DailyUserStat.html")
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error parsing: %s", err), 500)
		return
	}

	dList := []*stat.DocStat{}
	rList := []*stat.RevStat{}
	for k, v := range dayStat.FileRevs {
		file := dh.db.LoadFileStats(k)
		if file == nil {
			http.Error(rw, fmt.Sprintf("Error finding file: %s", k), 500)
			return
		}

		for _, vRev := range file.RevList {
			for _, vRevID := range v {
				if vRev.RevId == vRevID {
					dList = append(dList, file)
					rList = append(rList, &vRev)
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

////////////////////////////////////////////////////////////////////////////////
// File Handle
type FileHandle struct {
	db *database.StatTrackerDB
}

func (dh FileHandle) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	fileTemp, err := template.ParseFiles("./templates/fileStat.html")
	if err != nil {
		http.Error(rw, fmt.Sprintf("Error parsing: %s", err), 500)
		return
	}

	matches := reFilePathMatch.FindAllStringSubmatch(req.URL.String(), -1)
	if len(matches) < 1 {
		fmt.Println(matches, req.URL)
		http.Error(rw, "Invalid Path", 400)
		return
	}

	fileStat := dh.db.LoadFileStats(matches[0][1])
	if fileStat == nil {
		fmt.Fprintf(rw, "No stats for %s", matches[0][1])
		return
	}

	date, errDate := time.Parse(dateFormatLong, fileStat.LastMod)
	if errDate != nil {
		http.Error(rw, fmt.Sprintf("Error parsing: %s", fileStat.LastMod), 500)
		return
	}

	e := fileTemp.Execute(rw, struct {
		FullDate string
		ModDate  string
		Stat     *stat.DocStat
	}{
		date.Format("Monday, 2 Jan 2006"),
		date.Format(dateFormat),
		fileStat,
	})

	if e != nil {
		log.Println("Error in Temp", e)
	}

}
