package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/kivattt/getopt"
)

const programName = "fssize"
const version = "v0.0.1"

func main() {
	h := flag.Bool("help", false, "display this help and exit")
	v := flag.Bool("version", false, "output version information and exit")

	getopt.CommandLine.SetOutput(os.Stdout)
	getopt.CommandLine.Init(programName, flag.ExitOnError)
	getopt.Aliases(
		"v", "version",
		"h", "help",
	)

	err := getopt.CommandLine.Parse(os.Args[1:])
	if err != nil {
		os.Exit(0)
	}

	if *v {
		fmt.Println(programName, version)
		os.Exit(0)
	}

	if *h {
		fmt.Println("Usage: " + filepath.Base(os.Args[0]) + " [OPTIONS] [FILES]")
		fmt.Println("Find biggest regular files")
		fmt.Println()
		getopt.PrintDefaults()
		os.Exit(0)
	}

	app := tview.NewApplication()
	fssize := NewFSSize(app)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		return event
	})

	/*	go func() {
		for {
			app.QueueUpdateDraw(func() {})
			time.Sleep(30 * time.Millisecond)
		}
	}()*/
	path := "/"
	if len(getopt.CommandLine.Args()) > 0 {
		path = getopt.CommandLine.Arg(0)
	}

	stat, err := os.Stat(path)
	if err != nil {
		fmt.Println("No such directory: " + path)
		os.Exit(1)
	}
	if !stat.IsDir() {
		fmt.Println("Not a directory: " + path)
		os.Exit(1)
	}

	go fssize.AccumulateFiles(path)

	if err := app.SetRoot(fssize, true).EnableMouse(true).Run(); err != nil {
		log.Fatal(err)
	}
}
