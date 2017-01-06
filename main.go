package main

import (
	"flag"
	"log"
	"strings"

	"github.com/cfunkhouser/updraft/updraft"
)

var (
	root    = flag.String("backup_root", "", "Root of the backup tree")
	exclude = flag.String("exclude", "", "CSV list of paths under --backup_root to exclude")
)

func main() {
	flag.Parse()
	if *root == "" {
		log.Fatal("--backup_root must be specified")
	}

	var exc []string
	if *exclude != "" {
		exc = strings.Split(*exclude, ",")
	}

	fsw := updraft.NewFileSystem(*root, exc)
	defer fsw.Close()
	for {
		select {
		case f := <-fsw.C:
			log.Printf("Found file: %q\n", f)
		}
	}
}
