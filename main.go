package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"./login"
	"./web"
)

type CommandFunc func() error

// Flags
var (
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

	wf := web.MakeWebFace("127.0.0.1:1667", "./static", "./templates")
	SetupWebFace(wf)

	if *debug {
		log.Println("Debug Active")
	}

	loginClient, err := login.StartClient(wf, GetClientScope())
	if err != nil {
		log.Fatalln(err)
	}

	setupClients(loginClient)

	iStr, iErr := GetIdentity(login.Token.AccessToken)
	if iErr != nil {
		log.Fatalln(iErr)
	}
	log.Println("Token Str", iStr)

	OpenDB("_data.db")

	//DumpDocListKeys()
	//LoadFileDumpStats("1tD8oE8lgA06p39utoNP_NCE-kToLaws46SCiWKbpi68")
	//FullFileSweep()
	//FullFileStatPrintout()
	commandLoop()

	CloseDB()
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
