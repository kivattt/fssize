package main

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FSSize struct {
	*tview.Box
	files []File
}

type File struct {
	path string
	sizeBytes int64
}

func NewFSSize() *FSSize {
	return &FSSize{
		Box: tview.NewBox().SetBackgroundColor(tcell.ColorDefault),
	}
}

func (fssize *FSSize) Draw(screen tcell.Screen) {
	x, _, w, h := fssize.GetInnerRect()

	for i := 0; i < len(fssize.files); i++ {
		if i >= h-1 { // The bottom row is occupied by the bottom bar
			break
		}

		tview.Print(screen, fssize.files[i].path, 0, i, w, tview.AlignLeft, tcell.ColorDefault)
		tview.Print(screen, "    " + strconv.FormatInt(fssize.files[i].sizeBytes, 10), 0, i, w, tview.AlignRight, tcell.ColorDefault)
	}

	for i := x; i < x+w; i++ {
		screen.SetContent(i, h-1, ' ', nil, tcell.StyleDefault.Background(tcell.ColorBlack))
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

func (fssize *FSSize) AccumulateFiles() error {
	return filepath.WalkDir("/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}

		if len(fssize.files) >= 100 {
			if info.Size() < fssize.files[len(fssize.files)-1].sizeBytes {
				return nil
			}

			fssize.files = fssize.files[:len(fssize.files) - 1]
			fssize.files = append(fssize.files, File{path: path, sizeBytes: info.Size()})
			fssize.SortFiles()
			return nil
		}

		fssize.files = append(fssize.files, File{path: path, sizeBytes: info.Size()})
		fssize.SortFiles()

		return nil
	})
}
