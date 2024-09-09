package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/kivattt/getopt"
)

const programName = "fssize"
const version = "v0.0.1"

func main() {
	h := flag.Bool("help", false, "display this help and exit")
	v := flag.Bool("version", false, "output version information and exit")
	ignoreHiddenFiles := flag.Bool("ignore-hidden-files", false, "ignore files and folders starting with '.'")
	maxCount := flag.Int("max-count", 150, "max amount of files to output")
	output := flag.Bool("output", false, "output to stdout, biggest filesize first, filenames with newlines omitted")

	getopt.CommandLine.SetOutput(os.Stdout)
	getopt.CommandLine.Init(programName, flag.ExitOnError)
	getopt.Aliases(
		"v", "version",
		"h", "help",
		"i", "ignore-hidden-files",
		"c", "max-count",
		"o", "output",
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

	fssize := NewFSSize()
	fssize.ignoreHiddenFiles = *ignoreHiddenFiles
	if *maxCount <= 0 {
		os.Exit(0)
	}
	fssize.maxCount = *maxCount

	path := "/"
	if len(getopt.CommandLine.Args()) > 0 {
		path = getopt.CommandLine.Arg(0)
	}

	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
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

	fssize.rootFolderPath = path

	if *output {
		fssize.AccumulateFiles()

		for _, e := range fssize.files {
			if strings.ContainsRune(e.path, '\n') {
				os.Stderr.WriteString("\x1b[0;31m" + programName + ": (stderr) Path omitted for containing a newline: \"" + e.path + "\"\n\x1b[0m")
			} else {
				fmt.Println(e.path)
			}
		}
		os.Exit(0)
	}

	app := tview.NewApplication()
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
	fssize.app = app

	go fssize.AccumulateFiles()

	if err := app.SetRoot(fssize, true).Run(); err != nil {
		log.Fatal(err)
	}
}
