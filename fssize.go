package main

import (
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FSSize struct {
	*tview.Box
	app          *tview.Application
	files        []File
	accumulating bool
	//	filesToShow []File //
}

type File struct {
	path      string
	sizeBytes int64
}

func NewFSSize(app *tview.Application) *FSSize {
	return &FSSize{
		Box: tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
		app: app,
	}
}

func (fssize *FSSize) Draw(screen tcell.Screen) {
	x, _, w, h := fssize.GetInnerRect()

	for i := 0; i < len(fssize.files); i++ {
		if i >= h-1 { // The bottom row is occupied by the bottom bar
			break
		}

		styleText := "[:black]"
		/*		if i % 2 == 1 {
				styleText = "[:#232323:]"
			}*/

		//		tview.Print(screen, styleText + fssize.files[i].path, 0, i, w, tview.AlignLeft, tcell.ColorDefault)
		bruh := int32(max(20, 255-(i*8)))
		tview.Print(screen, styleText+fssize.files[i].path, 0, i, w, tview.AlignLeft, tcell.NewRGBColor(bruh, bruh, bruh))
		/*if i % 2 == 1 {
			for j := len(fssize.files[i].path); j < w; j++ {
				screen.SetContent(j, i, ' ', nil, tcell.StyleDefault.Background(tcell.NewRGBColor(0x23, 0x23, 0x23)))
			}
		}*/
		tview.Print(screen, styleText+"    "+BytesToHumanReadableUnitString(uint64(fssize.files[i].sizeBytes), 3), 0, i, w, tview.AlignRight, tcell.ColorDefault)
	}

	for i := x; i < x+w; i++ {
		screen.SetContent(i, h-1, ' ', nil, tcell.StyleDefault.Background(tcell.ColorBlack))
	}
	if fssize.accumulating {
		tview.Print(screen, "Searching...", 0, h-1, w, tview.AlignLeft, tcell.ColorDefault)
	} else {
		tview.Print(screen, "Done", 0, h-1, w, tview.AlignLeft, tcell.NewRGBColor(0, 255, 0))
	}
}

func (fssize *FSSize) SortFiles() {
	slices.SortFunc(fssize.files, func(a, b File) int {
		if a.sizeBytes < b.sizeBytes {
			return 1
		} else if a.sizeBytes > b.sizeBytes {
			return -1
		}

		return 0
	})
}

func (fssize *FSSize) AccumulateFiles(rootFolderPath string) error {
	fssize.accumulating = true
	err := filepath.WalkDir(rootFolderPath, func(path string, e fs.DirEntry, err error) error {
		/*		if e == nil {
				return nil
			}*/

		if path == "/proc" {
			return filepath.SkipDir
		}

		if !e.Type().IsRegular() {
			return nil
		}

		info, infoErr := e.Info()
		if infoErr != nil {
			return nil
		}

		//if info.Mode()

		if len(fssize.files) >= 100 {
			if info.Size() < fssize.files[len(fssize.files)-1].sizeBytes {
				return nil
			}

			fssize.files = fssize.files[:len(fssize.files)-1]
			fssize.files = append(fssize.files, File{path: path, sizeBytes: info.Size()})
			fssize.SortFiles()

			fssize.app.QueueUpdateDraw(func() {})
			return nil
		}

		fssize.files = append(fssize.files, File{path: path, sizeBytes: info.Size()})
		fssize.SortFiles()

		return nil
	})

	fssize.accumulating = false
	fssize.app.QueueUpdateDraw(func() {})
	return err
}
