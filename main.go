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
)

type CommandFunc func() error

// Flags
var (
	cacheToken   = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
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

	startClient()
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