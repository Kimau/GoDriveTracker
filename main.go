package main

import (
	"bufio"
	"bytes"
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
	gatherDocsChangedOnDate(1)
	gatherDocsChangedOnDate(2)
	gatherDocsChangedOnDate(3)
	gatherDocsChangedOnDate(4)

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

func gatherDocsChangedOnDate(daysAgo uint) error {
	aResp, err := google.ListActivities(20)
	if err != nil {
		return err
	}

	aList := aResp.Activities

	n := time.Now()
	startTime := time.Date(n.Year(), n.Month(), n.Day()-(int)(daysAgo), 0, 0, 0, 0, time.UTC)
	endTime := time.Date(n.Year(), n.Month(), n.Day()-(int)(daysAgo)+1, 0, 0, 0, 0, time.UTC)

	lastTimeStamp := time.Unix((int64)(aList[len(aList)-1].CombinedEvent.EventTimeMillis/1000), 0)

	fmt.Println("Fetching activity from ", startTime.String(), " to ", endTime.String())

	// Time
	pageCounter := 0
	if lastTimeStamp.After(startTime) {
		pageCounter += 1
		aResp, err = google.NextPage(20, aResp)

		aList = append(aList, aResp.Activities...)
	}

	// Gather Docs changed
	docsChanged := make(map[string]string)
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

			docsChanged[e.Target.Id] = e.Target.Name
			fmt.Println(e.User.Name, e.PrimaryEventType, e.Target.MimeType, e.Target.Name)
		}
	}

	// Gather Revisions
	for k, v := range docsChanged {
		rList, err := google.AllRevisions(k)
		if err != nil {
			return err
		}

		firstRev := rList[0]
		lastRev := rList[len(rList)-1]
		ft, _ := time.Parse(google.TimeFMT, firstRev.ModifiedTime)
		lt, _ := time.Parse(google.TimeFMT, lastRev.ModifiedTime)

		for _, r := range rList {
			dt, _ := time.Parse(google.TimeFMT, r.ModifiedTime)
			if dt.Before(startTime) && dt.After(ft) {
				firstRev = r
				ft = dt
			}

			if dt.After(endTime) && dt.Before(lt) {
				firstRev = r
				ft = dt
			}
		}

		fmt.Println(v, ft, lt)
	}

	return nil
}
