package updraft

import (
	"log"
	"os"
	"path/filepath"

	"github.com/rjeczalik/notify"
)

type FileSystemWatcher struct {
	start   string
	exclude []string
	C       <-chan *File
	c       chan<- *File

	notifications chan notify.EventInfo
}

func (w *FileSystemWatcher) Close() {
	notify.Stop(w.notifications)
	close(w.c)
}

func (w *FileSystemWatcher) sendFile(path string, info os.FileInfo) {
	f := DefaultFile()
	f.Name = path
	f.Length = int(info.Size())
	f.ModifiedTime = info.ModTime()
	w.c <- f
}

func (w *FileSystemWatcher) walk(path string, info os.FileInfo, inerr error) error {
	if inerr != nil {
		log.Print(inerr)
		return nil
	}

	// Skip files which match any pattern in the exclude list.
	for _, p := range w.exclude {
		if m, _ := filepath.Match(p, path); m {
			return nil
		}
	}

	// Skip irregular files (for now).
	if !info.Mode().IsRegular() {
		return nil
	}

	w.sendFile(path, info)
	return nil
}

func (w *FileSystemWatcher) doWalk() {
	if err := filepath.Walk(w.start, w.walk); err != nil {
		log.Print(err)
	}
}

func (w *FileSystemWatcher) do() {
	go w.doWalk() // Walk the FS to discover files.
	for {
		select {
		case e := <-w.notifications:
			log.Printf("event: %v", e.Event())
			info, err := os.Stat(e.Path())
			if err != nil {
				log.Printf("error stating: %v", err)
				continue
			}
			w.sendFile(e.Path(), info)
		}
	}
}

func NewFileSystemWatcher(start string, exclude []string) *FileSystemWatcher {
	c := make(chan *File)
	fsw := &FileSystemWatcher{
		start:         start,
		exclude:       exclude,
		C:             c,
		c:             c,
		notifications: make(chan notify.EventInfo, 1),
	}

	notify.Watch(filepath.Join(fsw.start, "/..."), fsw.notifications, notify.All)

	go fsw.do()

	return fsw
}
