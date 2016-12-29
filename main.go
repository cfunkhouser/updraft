package main

import (
	"log"

	"github.com/cfunkhouser/updraft/updraft"
)

func main() {
	fsw := updraft.NewFileSystemWatcher("/Users/cfunk/", []string{"/Users/cfunk/Applications", "/Users/cfunk/Library"})
	defer fsw.Close()
	for {
		select {
		case f := <-fsw.C:
			log.Printf("Got file: %q\n", f)
		}
	}
}
