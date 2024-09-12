package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/kivattt/getopt"
)

const programName = "fssize"
const version = "v0.0.2"

func printError(str string) {
	os.Stderr.WriteString("\x1b[0;31m" + programName + ": " + str + "\x1b[0m\n")
}

func main() {
	h := flag.Bool("help", false, "display this help and exit")
	v := flag.Bool("version", false, "output version information and exit")
	ignoreHiddenFiles := flag.Bool("ignore-hidden-files", false, "ignore files and folders starting with '.'")
	maxCount := flag.Int("max-file-count", 150, "max amount of files/folders to output")
	outputFiles := flag.Bool("output-files", false, "output to stdout, biggest filesize first, filenames with newlines omitted")
	outputDirs := flag.Bool("output-dirs", false, "output to stdout, biggest sum filesize first, paths with newlines omitted")
	outputPackages := flag.Bool("output-packages", false, "output to stdout, biggest estimated filesize first")

	getopt.CommandLine.SetOutput(os.Stdout)
	getopt.CommandLine.Init(programName, flag.ExitOnError)
	getopt.Aliases(
		"v", "version",
		"h", "help",
		"i", "ignore-hidden-files",
		"c", "max-file-count",
		"o", "output-files",
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

	btoi := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	if btoi(*outputFiles)+btoi(*outputDirs)+btoi(*outputPackages) > 1 {
		printError("More than one of: --output-files (-o), --output-dirs or --output-packages were set, pick one!")
		os.Exit(0)
	}

	if *outputFiles || *outputDirs || *outputPackages {
		if *outputPackages {
			fssize.AccumulatePackages()
		} else {
			fssize.AccumulateFilesAndFolders()
		}

		ptr := &fssize.files
		if *outputDirs {
			ptr = &fssize.folders
		} else if *outputPackages {
			ptr = &fssize.packages
		}

		for _, e := range *ptr {
			if strings.ContainsRune(e.path, '\n') {
				printError("Path omitted for containing a newline: \"" + e.path + "\"")
			} else {
				fmt.Println(e.path)
			}
		}

		if *outputPackages {
			printError(`You should not use --output-packages in scripts
Use dpkg-query directly, something like this:

dpkg-query -Wf '${Installed-Size}\t${Package}\n' | sort -rn

This outputs the estimated kibibyte (KiB) size of all packages`)
		}

		os.Exit(0)
	}

	app := tview.NewApplication()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		if event.Key() == tcell.KeyTab {
			fssize.TabForward()
		} else if event.Key() == tcell.KeyBacktab {
			fssize.TabBackward()
		}

		return event
	})

	fssize.app = app

	fssize.AccumulatePackages()
	go fssize.AccumulateFilesAndFolders()

	fssize.accumulating = true // Just to make sure the below goroutine doesn't quit early
	go func() {
		for fssize.accumulating {
			time.Sleep(250 * time.Millisecond)
			app.QueueUpdateDraw(func() {})
		}
	}()

	if err := app.SetRoot(fssize, true).Run(); err != nil {
		log.Fatal(err)
	}
}
