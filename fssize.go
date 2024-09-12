package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Tab int

const (
	Files    Tab = 0
	Folders      = 1
	Packages     = 2 // dpkg-query
)

type FSSize struct {
	*tview.Box
	app               *tview.Application
	currentTab        Tab
	files             []File
	folders           []File
	packages          []File // path = package name, sizeBytes = estimated size in kibibytes
	dpkgQueryWorked   bool
	maxCount          int
	ignoreHiddenFiles bool
	accumulating      bool
	rootFolderPath    string
}

type File struct {
	path      string
	sizeBytes int64
}

func NewFSSize() *FSSize {
	return &FSSize{
		Box:             tview.NewBox().SetBackgroundColor(tcell.NewRGBColor(46, 52, 54)),
		currentTab:      Files,
		dpkgQueryWorked: true,
	}
}

func (fssize *FSSize) TabForward() {
	fssize.currentTab++
	fssize.currentTab %= 3
}

func (fssize *FSSize) TabBackward() {
	fssize.currentTab--
	if fssize.currentTab < 0 {
		fssize.currentTab = 2
	}
}

func (fssize *FSSize) Draw(screen tcell.Screen) {
	x, _, w, h := fssize.GetInnerRect()
	fssize.Box.DrawForSubclass(screen, fssize)

	// Top bar
	/*for i := x; i < x+w; i++ {
		screen.SetContent(i, 0, ' ', nil, tcell.StyleDefault.Background(tcell.NewRGBColor(46, 52, 54)).Underline(true))
	}*/
	filesStyle := ""
	folderStyle := ""
	packagesStyle := ""
	if fssize.currentTab == Files {
		filesStyle = "[::br]"
	} else if fssize.currentTab == Folders {
		folderStyle = "[::br]"
	} else if fssize.currentTab == Packages {
		packagesStyle = "[::br]"
	}

	tview.Print(screen, filesStyle+" Files [-:-:-:-]"+folderStyle+" Folders [-:-:-:-]"+packagesStyle+" Packages (dpkg-query) [-:-:-:-]", 0, 0, w, tview.AlignLeft, tcell.ColorDefault)
	tview.Print(screen, "<- Press Tab or Shift+Tab to switch ", 0, 0, w, tview.AlignRight, tcell.ColorDefault)

	ptr := &fssize.files
	if fssize.currentTab == Folders {
		ptr = &fssize.folders
	} else if fssize.currentTab == Packages {
		ptr = &fssize.packages
	}

	if fssize.currentTab == Packages && len(*ptr) == 0 {
		tview.Print(screen, "[::b]Failed to run dpkg-query", 0, h/2, w, tview.AlignCenter, tcell.ColorDefault)
	} else {
		for i := 0; i < len(*ptr); i++ {
			if i+1 >= h-1 { // The bottom row is occupied by the bottom bar
				break
			}

			styleText := ""
			if i%2 == 0 {
				styleText = "[:#141414:]"
			}

			var relPath string
			if fssize.rootFolderPath == "/" {
				relPath = (*ptr)[i].path
			} else {
				var err error
				relPath, err = filepath.Rel(fssize.rootFolderPath, (*ptr)[i].path)
				if err != nil {
					relPath = (*ptr)[i].path
				}
			}

			prefix := ""
			if fssize.currentTab == Packages {
				prefix = "~"
			}
			_, sizePrintedLength := tview.Print(screen, styleText+"[::b]"+prefix+BytesToHumanReadableUnitString(uint64((*ptr)[i].sizeBytes), 3), 0, i+1, w, tview.AlignRight, tcell.ColorWhite)
			// Flawed when FilenameInvisibleCharactersAsCodeHighlighted does anything
			if len(relPath) > w-sizePrintedLength-1 {
				relPath = relPath[:max(0, w-sizePrintedLength-1-3)] + "[#606060]..."
			}

			filenameText := FilenameInvisibleCharactersAsCodeHighlighted(relPath, styleText)
			_, pathPrintedLength := tview.Print(screen, styleText+filenameText, 0, i+1, w-sizePrintedLength, tview.AlignLeft, tcell.NewRGBColor(200, 200, 200))

			if i%2 == 0 {
				for j := pathPrintedLength; j < w-sizePrintedLength; j++ {
					screen.SetContent(j, i+1, ' ', nil, tcell.StyleDefault.Background(tcell.NewRGBColor(0x14, 0x14, 0x14)))
				}
			}
		}
	}

	// Bottom bar
	color := tcell.ColorYellow
	if !fssize.accumulating {
		color = tcell.NewRGBColor(0, 255, 0)
	}
	for i := x; i < x+w; i++ {
		//		screen.SetContent(i, h-1, ' ', nil, tcell.StyleDefault.Background(tcell.ColorWhite))
		screen.SetContent(i, h-1, ' ', nil, tcell.StyleDefault.Background(color))
	}
	if fssize.accumulating {
		tview.Print(screen, "[:yellow] Searching... ", 0, h-1, w, tview.AlignLeft, tcell.ColorBlack)
	} else {
		tview.Print(screen, "[:#00ff00:] Finished ", 0, h-1, w, tview.AlignLeft, tcell.ColorBlack)
	}

	tview.Print(screen, programName+" "+version, 0, h-1, w, tview.AlignCenter, tcell.ColorBlack)
	tview.Print(screen, "Press 'q' to quit ", 0, h-1, w, tview.AlignRight, tcell.ColorBlack)
}

func (fssize *FSSize) SortFiles(files *[]File) {
	slices.SortFunc(*files, func(a, b File) int {
		if a.sizeBytes < b.sizeBytes {
			return 1
		} else if a.sizeBytes > b.sizeBytes {
			return -1
		}

		return 0
	})
}

// https://cs.opensource.google/go/go/+/refs/tags/go1.23.1:src/path/filepath/path.go;l=309
func (fssize *FSSize) walkDir(path string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
	if err := walkDirFn(path, d, nil); err != nil || !d.IsDir() {
		if err == fs.SkipDir && d.IsDir() {
			err = nil
		}

		return err
	}

	files, err := os.ReadDir(path)
	if err != nil {
		err = walkDirFn(path, d, err)
		if err != nil {
			if err == fs.SkipDir && d.IsDir() {
				err = nil
			}

			return err
		}
	}

	directories := []fs.DirEntry{}
	var dirSize int64
	for _, file := range files {
		if file.IsDir() {
			directories = append(directories, file)
			continue
		}

		if !file.Type().IsRegular() {
			continue
		}

		info, infoErr := file.Info()
		if infoErr == nil {
			dirSize += info.Size()
		}

		path1 := filepath.Join(path, file.Name())
		err := walkDirFn(path1, file, nil)
		if err != nil {
			return err
		}
	}

	if len(fssize.folders) >= fssize.maxCount {
		if dirSize >= fssize.folders[len(fssize.folders)-1].sizeBytes {
			fssize.folders = fssize.folders[:len(fssize.folders)-1]
			fssize.folders = append(fssize.folders, File{path: path, sizeBytes: dirSize})
			fssize.SortFiles(&fssize.folders)

			/*if fssize.app != nil && fssize.currentTab == Folders {
				fssize.app.QueueUpdateDraw(func() {})
			}*/
		}
	} else {
		fssize.folders = append(fssize.folders, File{path: path, sizeBytes: dirSize})
		fssize.SortFiles(&fssize.folders)
	}

	/*		if len(fssize.files) >= fssize.maxCount {
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
			fssize.SortFiles()*/

	for _, d1 := range directories {
		path1 := filepath.Join(path, d1.Name())
		if err := fssize.walkDir(path1, d1, walkDirFn); err != nil {
			if err == fs.SkipDir {
				break
			}

			return err
		}
	}

	return nil
}

// https://cs.opensource.google/go/go/+/refs/tags/go1.23.1:src/path/filepath/path.go;l=395
func (fssize *FSSize) WalkDir(root string, fn fs.WalkDirFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = fssize.walkDir(root, fs.FileInfoToDirEntry(info), fn)
	}
	if err == fs.SkipDir || err == fs.SkipAll {
		return nil
	}
	return err
}

func (fssize *FSSize) AccumulatePackages() error {
	// According to the man page, --showformat has a short option '-f' since dpkg 1.13.1, so let's use the long option
	output, err := exec.Command("dpkg-query", "--show", "--showformat=${Installed-Size},${Package}\n").Output()
	if err != nil {
		fssize.dpkgQueryWorked = false
		return err
	}

	var builder strings.Builder
	for _, c := range output {
		if c == '\n' {
			split := strings.Split(builder.String(), ",")
			if len(split) != 2 {
				panic("unexpected output from dpkg-query, more than 1 comma in output")
			}
			packageName := split[1]
			estimatedKibibytes, err := strconv.Atoi(split[0])
			if err != nil {
				panic("unexpected output from dpkg-query, non-number estimated size")
			}

			// The ${Installed-Size} format is in estimated KiB
			// https://git.dpkg.org/git/dpkg/dpkg.git/tree/man/deb-substvars.pod#n176
			fssize.packages = append(fssize.packages, File{path: packageName, sizeBytes: int64(estimatedKibibytes) * 1024})

			builder.Reset()
			continue
		}

		builder.WriteByte(c)
	}

	fssize.SortFiles(&fssize.packages)
	return nil
}

func (fssize *FSSize) AccumulateFilesAndFolders() error {
	fssize.accumulating = true
	//	err := filepath.WalkDir(fssize.rootFolderPath, func(path string, e fs.DirEntry, err error) error {
	err := fssize.WalkDir(fssize.rootFolderPath, func(path string, e fs.DirEntry, err error) error {
		if fssize.ignoreHiddenFiles {
			if strings.HasPrefix(e.Name(), ".") {
				if e.IsDir() {
					return filepath.SkipDir
				} else {
					return nil
				}
			}
		}

		if (path == "/dev" || path == "/proc" || path == "/sys" || path == "/home/.ecryptfs") && e.IsDir() {
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
			fssize.SortFiles(&fssize.files)

			/*if fssize.app != nil && fssize.currentTab == Files {
				fssize.app.QueueUpdateDraw(func() {})
			}*/
			return nil
		}

		fssize.files = append(fssize.files, File{path: path, sizeBytes: info.Size()})
		fssize.SortFiles(&fssize.files)

		return nil
	})

	fssize.accumulating = false
	if fssize.app != nil {
		fssize.app.QueueUpdateDraw(func() {})
	}
	return err
}
