package main

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FSSize struct {
	*tview.Box
	app          *tview.Application
	files        []File
	maxCount int
	ignoreHiddenFiles bool
	accumulating bool
	rootFolderPath string
	//	filesToShow []File //
}

type File struct {
	path      string
	sizeBytes int64
}

func NewFSSize() *FSSize {
	return &FSSize{
		Box: tview.NewBox().SetBackgroundColor(tcell.ColorBlack),
	}
}

func (fssize *FSSize) Draw(screen tcell.Screen) {
	x, _, w, h := fssize.GetInnerRect()
	fssize.Box.DrawForSubclass(screen, fssize)

	for i := 0; i < len(fssize.files); i++ {
		if i >= h-1 { // The bottom row is occupied by the bottom bar
			break
		}

		styleText := ""
		if i % 2 == 0 {
			styleText = "[:#141414:]"
		}

		var relPath string
		if fssize.rootFolderPath == "/" {
			relPath = fssize.files[i].path
		} else {
			var err error
			relPath, err = filepath.Rel(fssize.rootFolderPath, fssize.files[i].path)
			if err != nil {
				relPath = fssize.files[i].path
			}
		}

		_, sizePrintedLength := tview.Print(screen, styleText+"[::b]"+BytesToHumanReadableUnitString(uint64(fssize.files[i].sizeBytes), 3), 0, i, w, tview.AlignRight, tcell.ColorWhite)
		// Flawed when FilenameInvisibleCharactersAsCodeHighlighted does anything
		if len(relPath) > w-sizePrintedLength-1 {
			relPath = relPath[:max(0, w-sizePrintedLength-1-3)] + "[yellow]..."
		}

		filenameText := FilenameInvisibleCharactersAsCodeHighlighted(relPath, styleText)

		_, pathPrintedLength := tview.Print(screen, styleText + filenameText, 0, i, w-sizePrintedLength, tview.AlignLeft, tcell.NewRGBColor(200, 200, 200))
		//bruh := int32(max(20, 255-(i*8)))
		//tview.Print(screen, styleText+fssize.files[i].path, 0, i, w, tview.AlignLeft, tcell.NewRGBColor(bruh, bruh, bruh))

		if i % 2 == 0 {
			for j := pathPrintedLength; j < w-sizePrintedLength; j++ {
				screen.SetContent(j, i, ' ', nil, tcell.StyleDefault.Background(tcell.NewRGBColor(0x14, 0x14, 0x14)))
			}
		}
	}

	for i := x; i < x+w; i++ {
		screen.SetContent(i, h-1, ' ', nil, tcell.StyleDefault.Background(tcell.ColorWhite))
	}
	if fssize.accumulating {
		tview.Print(screen, "[:yellow] Searching... ", 0, h-1, w, tview.AlignLeft, tcell.ColorBlack)
	} else {
		tview.Print(screen, "[:#00ff00:] Done ", 0, h-1, w, tview.AlignLeft, tcell.ColorBlack)
	}

//	tview.Print(screen, "Press 'q' to quit", 0, h-1, w, tview.AlignRight, tcell.ColorWhite)
	tview.Print(screen, "Press 'q' to quit", 0, h-1, w, tview.AlignRight, tcell.ColorBlack)
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

func (fssize *FSSize) AccumulateFiles() error {
	fssize.accumulating = true
	err := filepath.WalkDir(fssize.rootFolderPath, func(path string, e fs.DirEntry, err error) error {
		if fssize.ignoreHiddenFiles {
			if strings.HasPrefix(e.Name(), ".") {
				if e.IsDir() {
					return filepath.SkipDir
				} else {
					return nil
				}
			}
		}

		if path == "/dev" || path == "/proc" || path == "/sys" || path == "/home/.ecryptfs" {
			return filepath.SkipDir
		}

		if !e.Type().IsRegular() {
			return nil
		}

		info, infoErr := e.Info()
		if infoErr != nil {
			return nil
		}

		if len(fssize.files) >= fssize.maxCount {
			if info.Size() < fssize.files[len(fssize.files)-1].sizeBytes {
				return nil
			}

			fssize.files = fssize.files[:len(fssize.files)-1]
			fssize.files = append(fssize.files, File{path: path, sizeBytes: info.Size()})
			fssize.SortFiles()

			if fssize.app != nil {
				fssize.app.QueueUpdateDraw(func() {})
			}
			return nil
		}

		fssize.files = append(fssize.files, File{path: path, sizeBytes: info.Size()})
		fssize.SortFiles()

		return nil
	})

	fssize.accumulating = false
	if fssize.app != nil {
		fssize.app.QueueUpdateDraw(func() {})
	}
	return err
}
