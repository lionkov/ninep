package main

import (
	"flag"
	"log"

	"github.com/lionkov/ninep/srv/nullfs"
)

var (
	addr  = flag.String("addr", ":5640", "network address")
	debug = flag.Int("d", 0, "debuglevel")
)

func main() {
	var err error

	flag.Parse()
	f, err := nullfs.NewNullFS(*debug)
	if err != nil {
		log.Fatalf("New NullFS failure: %v", err)
	}
	if err = f.StartNetListener("tcp", *addr); err != nil {
		log.Fatalf("Start failure: %v", err)
	}
}
