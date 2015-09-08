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

	database "./database"
	google "./google"
	stat "./stat"
	web "./web"
)

type CommandFunc func() error

const ()

var (
	addr         = flag.String("addr", "127.0.0.1:1667", "Web Address")
	staticFldr   = flag.String("static", "./static", "Static Folder")
	templateFldr = flag.String("template", "./templates", "Templates Folder")
	debug        = flag.Bool("debug", false, "show HTTP traffic")
	commandFuncs = make(map[string]CommandFunc)
)

func init() {
	commandFuncs["help"] = listCommands

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

	// Login
	log.Println("Login")
	Tok, cErr := google.Login(wf, google.GetClientScope())
	if cErr != nil {
		log.Fatalln("Login Error:", cErr)
	}

	// Get Identity
	log.Println("Get Identity")
	iTok, iErr := google.GetIdentity(Tok)
	if iErr != nil {
		log.Fatalln("Identity Error:", iErr)
	}

	b := new(bytes.Buffer)
	google.EncodeToken(Tok, b)

	user := stat.UserStat{
		UpdateDate: time.Now().String(),
		Token:      b.Bytes(),
		Email:      iTok.Email,
		UserID:     iTok.UserId,
	}
	log.Println("User", user.UserID, user.Email)

	// Setup Database
	log.Println("Setup Database")
	db := database.OpenDB("_data.db")

	// Get & Write DB
	db.WriteUserStats(&user)

	// Init DB
	SetupDatabase(wf, db)

	// Setup Webface with Database
	log.Println("Setup Webface with Database")
	SetupWebFace(wf, db)

	// Running Loop
	log.Println("Running Loop")
	commandLoop()

	// Clean up
	log.Println("Clean up")
	db.CloseDB()
}

func commandLoop() {
	lines := scanForInput()

	for {
		fmt.Println("Enter Command: ")
		select {
		case line := <-lines:
			line = strings.ToLower(line)

			valFunc, ok := commandFuncs[line]
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
	for i, _ := range commandFuncs {
		commandOut += i + ", "
	}

	fmt.Println(commandOut)
	return nil
}
