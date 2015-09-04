package main

import (
	//"html/template"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

type WebFace struct {
	Addr       string
	StaticRoot string
	Templates  string
	Router     *http.ServeMux
	Redirect   string
	LoginURL   string

	OutMsg chan string
	InMsg  chan string
}

var (
	rePathMatch = regexp.MustCompile("/day/([0-9]+)[/\\-]([0-9]+)[/\\-]([0-9]+)")
	activeWF    *WebFace
)

func MakeWebFace(addr string, static_root string, templatesFolder string) *WebFace {
	activeWF = &WebFace{
		Addr:       addr,
		Router:     http.NewServeMux(),
		StaticRoot: static_root,
		Templates:  templatesFolder,

		OutMsg: make(chan string),
		InMsg:  make(chan string),
	}

	activeWF.Router.Handle("/static/", http.FileServer(http.Dir(static_root)))
	activeWF.Router.HandleFunc("/", SummaryHandle)
	activeWF.Router.HandleFunc("/day/", DayHandle)

	go activeWF.HostLoop()

	return activeWF
}

func (wf *WebFace) HostLoop() {
	defer log.Println("Stopped Listening")

	log.Println("Listening on", wf.Addr)
	err := http.ListenAndServe(wf.Addr, wf)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func (wf *WebFace) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if activeWF.Redirect != "" {
		http.Redirect(rw, req, activeWF.Redirect, 307)
		return
	}

	if len(activeWF.LoginURL) > 1 {
		fmt.Fprintf(rw, `<h1><a href="%s" target="_blank">Click to Login</a></h1>`, activeWF.LoginURL)
		return
	}

	wf.Router.ServeHTTP(rw, req)
}

func setLoginURL(url string) {
	if activeWF == nil {
		return
	}

	activeWF.LoginURL = url
}

func setRedirectURL(url string) {
	if activeWF == nil {
		return
	}

	activeWF.Redirect = url
}

func SummaryHandle(rw http.ResponseWriter, req *http.Request) {
	for dayStat := LoadNextDailyStat(""); dayStat != nil; dayStat = LoadNextDailyStat(dayStat.ModDate) {
		fmt.Fprintf(rw, `<div><a href="/day/%s">%s</a> you wrote %d words and deleted %d words.</div>`, dayStat.ModDate, dayStat.ModDate, dayStat.WordAdd, dayStat.WordSub)
	}

}

func DayHandle(rw http.ResponseWriter, req *http.Request) {

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

	dayStat := LoadDailyStats(shortDate)
	if dayStat == nil {
		fmt.Fprintf(rw, "No stats for %s", shortDate)
		return
	}

	fmt.Fprintf(rw, "On %s you wrote %d words and deleted %d words.\n", dayStat.ModDate, dayStat.WordAdd, dayStat.WordSub)
	fmt.Fprintf(rw, "Edited %d files. \n", len(dayStat.FileRevs))
	for k, v := range dayStat.FileRevs {
		file := LoadFile(k)
		if file == nil {
			fmt.Fprintf(rw, "> File [%s] not found [%d revisions] \n", k, len(v))
		} else {
			fmt.Fprintf(rw, "> %s  [%d revisions]\n", file.Title, len(v))
		}
	}
}

//===
