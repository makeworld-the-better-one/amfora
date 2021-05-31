package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/makeworld-the-better-one/amfora/bookmarks"
	"github.com/makeworld-the-better-one/amfora/client"
	"github.com/makeworld-the-better-one/amfora/config"
	"github.com/makeworld-the-better-one/amfora/display"
	"github.com/makeworld-the-better-one/amfora/subscriptions"
)

var (
	version = "v1.8.0"
	commit  = "unknown"
	builtBy = "unknown"
)

func main() {
	// err := logger.Init()
	// if err != nil {
	// 	panic(err)
	// }

	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Println("Amfora", version)
			fmt.Println("Commit:", commit)
			fmt.Println("Built by:", builtBy)
			return
		}
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			fmt.Println("Amfora is a fancy terminal browser for the Gemini protocol.")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("amfora [URL]")
			fmt.Println("amfora --version, -v")
			return
		}
	}

	err := config.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}
	client.Init()

	err = subscriptions.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "subscriptions.json error: %v\n", err)
		os.Exit(1)
	}
	err = bookmarks.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bookmarks.xml error: %v\n", err)
		os.Exit(1)
	}

	// Initialize lower-level cview app
	if err = display.App.Init(); err != nil {
		panic(err)
	}

	// Initialize Amfora's settings
	display.Init(version, commit, builtBy)
	display.NewTab()
	if len(os.Args[1:]) > 0 {
		url := os.Args[1]
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "gemini://") && !strings.HasPrefix(url, "file://") {
			fileName := url
			if !strings.HasPrefix(fileName, "/") {
				cwd, err := os.Getwd()
				if err != nil {
					fmt.Fprintf(os.Stderr, "cannot get working directory path, error: %v\n", err)
					os.Exit(1)
				}
				fileName = cwd + "/" + fileName
			}
			url = "file://" + fileName
		}
		display.URL(url)
	}

	// Start
	if err = display.App.Run(); err != nil {
		panic(err)
	}
}
