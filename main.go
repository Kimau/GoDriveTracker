package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	google "./google"
	stat "./stat"
	web "./web"
)

type CommandFunc func() error

type DocRevStruct struct {
	Name string
	Prev *google.RevBody
	Next *google.RevBody
}

var (
	addr         = flag.String("addr", "127.0.0.1:1667", "Web Address")
	staticFldr   = flag.String("static", "./static", "Static Folder")
	templateFldr = flag.String("template", "./templates", "Templates Folder")
	debug        = flag.Bool("debug", false, "show HTTP traffic")
	commandFuncs = make(map[string]CommandFunc)
	userStat     *stat.UserStat
)

func init() {
	commandFuncs["help"] = listCommands
	commandFuncs["activity"] = listActivity

	switch runtime.GOOS {
	case "windows":
		commandFuncs["clear"] = func() error {
			cmd := exec.Command("cmd", "/c", "cls")
			cmd.Stdout = os.Stdout
			cmd.Run()
			return nil
		}

	case "linux":
		fallthrough
	default:
		commandFuncs["clear"] = func() error {
			print("\033[H\033[2J")
			return nil
		}
	}
}

func main() {
	commandFuncs["clear"]()
	flag.Parse()
	if *debug {
		log.Println("Debug Active")
	}

	// Start Web Server
	log.Println("Start Web Server")
	wf := web.MakeWebFace(*addr, *staticFldr, *templateFldr)
	wf.RedirectHandler = func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "Starting Server on %s", *addr)
	}

	// Get Identity
	{
		// Login
		log.Println("Login")
		Tok, cErr := google.Login(wf, google.GetClientScope())
		if cErr != nil {
			log.Fatalln("Login Error:", cErr)
		}

		iTok, iErr := google.GetIdentity(Tok)
		if iErr != nil {
			log.Fatalln("Identity Error:", iErr)
		}

		b := new(bytes.Buffer)
		google.EncodeToken(Tok, b)

		userStat = &stat.UserStat{
			UpdateDate: time.Now().String(),
			Token:      b.Bytes(),
			Email:      iTok.Email,
			UserID:     iTok.UserId,
		}

		// TODO :: Save User
	}

	log.Println("User", userStat.UserID, userStat.Email)
	wf.RedirectHandler = nil

	// Gather
	var err error
	var wc int
	var sc []*stat.CountStat
	sc, wc, err = gatherDocsChangedOnDate(1)
	if err != nil {
		panic(err)
	}
	fmt.Println("Words: ", wc)
	for _, v := range sc {
		fmt.Println("\t", v)
	}

	sc, wc, err = gatherDocsChangedOnDate(2)
	if err != nil {
		panic(err)
	}
	fmt.Println("Words: ", wc)
	for _, v := range sc {
		fmt.Println("\t", v)
	}

	sc, wc, err = gatherDocsChangedOnDate(5)
	if err != nil {
		panic(err)
	}
	fmt.Println("Words: ", wc)
	for _, v := range sc {
		fmt.Println("\t", v)
	}

	sc, wc, err = gatherDocsChangedOnDate(6)
	if err != nil {
		panic(err)
	}
	fmt.Println("Words: ", wc)
	for _, v := range sc {
		fmt.Println("\t", v)
	}

	// Running Loop
	log.Println("Running Loop")
	commandLoop()

	// Clean up
	log.Println("Clean up")
}

func commandLoop() {
	lines := scanForInput()

	for {
		fmt.Println("Enter Command: ")
		select {
		case line := <-lines:
			line = strings.ToLower(line)
			toks := strings.Split(line, " ")

			valFunc, ok := commandFuncs[toks[0]]
			if ok {
				err := valFunc()

				if err != nil {
					log.Printf("Error [%s]: %s", line, err.Error())
				}
			} else if line == "quit" {
				return
			} else {
				log.Printf("Unknown command: %s", line)
				listCommands()
			}

		}
	}

}

func scanForInput() chan string {
	lines := make(chan string)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	return lines
}

func listCommands() error {
	commandOut := "Commands: "
	for i := range commandFuncs {
		commandOut += i + ", "
	}

	fmt.Println(commandOut)
	return nil
}

func listActivity() error {
	aList, err := google.ListActivities(20)
	if err != nil {
		return err
	}

	fmt.Println("== Activity ==")
	for _, a := range aList.Activities {
		fmt.Println("----")
		for _, e := range a.SingleEvents {
			fmt.Println(e.User.Name, e.PrimaryEventType, e.Target.MimeType, e.Target.Name)
		}
	}

	fmt.Println("==============")
	return nil
}

func gatherDocsChangedOnDate(daysAgo uint) ([]*stat.CountStat, int, error) {
	// Get Recent Activity
	aResp, err := google.ListActivities(20)
	if err != nil {
		return nil, 0, err
	}
	aList := aResp.Activities

	// Setup Time Window
	n := time.Now()
	startTime := time.Date(n.Year(), n.Month(), n.Day()-(int)(daysAgo), 0, 0, 0, 0, time.UTC)
	endTime := time.Date(n.Year(), n.Month(), n.Day()-(int)(daysAgo)+1, 0, 0, 0, 0, time.UTC)
	lastTimeStamp := time.Unix((int64)(aList[len(aList)-1].CombinedEvent.EventTimeMillis/1000), 0)

	fmt.Println("Fetching activity from ", startTime.String(), " to ", endTime.String())

	// Get pages until we cover the time frame
	pageCounter := 0
	if lastTimeStamp.After(startTime) {
		pageCounter += 1
		aResp, err = google.NextPage(20, aResp)

		aList = append(aList, aResp.Activities...)
	}

	// Gather Docs changed
	docsChanged := make(map[string]DocRevStruct)

	// Check we have at least some activity in time frame
	for _, a := range aList {
		for _, e := range a.SingleEvents {
			if e.Target.MimeType != google.MimeDoc {
				continue
			}

			ts := time.Unix((int64)(e.EventTimeMillis/1000), 0)
			if ts.After(endTime) || ts.Before(startTime) {
				continue
			}

			_, ok := docsChanged[e.Target.Id]
			if ok {
				continue
			}

			docsChanged[e.Target.Id] = DocRevStruct{Name: e.Target.Name}
			// fmt.Println(e.User.Name, e.PrimaryEventType, e.Target.MimeType, e.Target.Name)
		}
	}

	totalWC := 0
	var wordCounts []*stat.CountStat

	// Gather Revisions
	for k, v := range docsChanged {
		rList, err := google.AllRevisions(k)
		if err != nil {
			return nil, 0, err
		}

		// fmt.Println(len(rList), rList[0].ModifiedDate, rList[len(rList)-1].ModifiedDate)

		v.Prev = nil
		v.Next = nil

		// Find Last Rev before Time frame and Last Rev in Timeframe
		for _, r := range rList {
			dt, err := time.Parse(google.TimeFMT, r.ModifiedDate)
			if err != nil {
				return nil, 0, err
			}
			// fmt.Println(r.ModifiedDate, "\t", dt.Before(startTime), dt.After(startTime) && dt.Before(endTime), dt.After(endTime))

			if dt.Before(startTime) && (v.Prev == nil || dt.After(v.Prev.ModTime)) {
				v.Prev = &google.RevBody{Rev: r, ModTime: dt}
			}

			if dt.Before(endTime) && (v.Next == nil || dt.After(v.Next.ModTime)) {
				v.Next = &google.RevBody{Rev: r, ModTime: dt}
			}
		}

		if v.Next == nil {
			return nil, 0, errors.New(fmt.Sprintf("%s : Unable to find revision [%s]", v.Name, k))
		}

		// Get Body of Files
		var wc1 int
		if v.Prev == nil {
			v.Prev = &google.RevBody{Rev: nil, ModTime: startTime, Body: ""}
			wc1 = 0
		} else {
			v.Prev.Body, err = google.ExportRev(v.Prev.Rev)
			if err != nil {
				return nil, 0, err
			}

			_, wc1 = stat.GetTopWords(v.Prev.Body)
		}

		v.Next.Body, err = google.ExportRev(v.Next.Rev)
		if err != nil {
			return nil, 0, err
		}
		wordCloud, wc2 := stat.GetTopWords(v.Next.Body)

		wordCounts = append(wordCounts, &stat.CountStat{
			Name:      v.Name,
			WordCount: wc2 - wc1,
			WordFreq:  wordCloud,
		})

		if wc2 > wc1 {
			totalWC += wc2 - wc1
		}

		// fmt.Println(v.Name, ":\t", wc1, " - ", wc2, "\n\t", v.Prev.ModTime, "\t", v.Next.ModTime, "\n", wordCloud)
	}

	return wordCounts, totalWC, nil
}
