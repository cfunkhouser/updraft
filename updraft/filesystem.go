package updraft

import (
	"log"
	"os"
	"path/filepath"

	"github.com/rjeczalik/notify"
)

type FileSystem struct {
	start   string
	exclude []string
	C       <-chan *RevisionID
	c       chan<- *RevisionID

	notifications chan notify.EventInfo
}

func (w *FileSystem) Close() {
	notify.Stop(w.notifications)
	close(w.c)
}

func (w *FileSystem) sendFile(path string, info os.FileInfo) {
	w.c <- &RevisionID{
		Name:    path,
		ModTime: info.ModTime(),
	}
}

func (w *FileSystem) walk(path string, info os.FileInfo, inerr error) error {
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

func (w *FileSystem) doWalk() {
	if err := filepath.Walk(w.start, w.walk); err != nil {
		log.Print(err)
	}
}

func (w *FileSystem) do() {
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

func NewFileSystem(start string, exclude []string) *FileSystem {
	c := make(chan *RevisionID)
	fsw := &FileSystem{
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
