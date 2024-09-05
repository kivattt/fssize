package main

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	fssize := NewFSSize()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' {
			app.Stop()
			return nil
		}

		return event
	})

	go fssize.AccumulateFiles()

	if err := app.SetRoot(fssize, true).Run(); err != nil {
		log.Fatal(err)
	}
}
