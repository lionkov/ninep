package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/clnt"
	"github.com/lionkov/ninep/srv/nullfs"
)

var (
	debug = flag.Int("d", 0, "debuglevel")
	bench = flag.String("bench", "all", "benchmark: read, write, all")
)

func benchZero(b *testing.B, msize, iosize uint32, clnt *clnt.Clnt, rootfid *clnt.Fid) {

	d := clnt.FidAlloc()
	if _, err := clnt.Walk(rootfid, d, []string{"zero"}); err != nil {
		b.Fatalf("%v", err)
	}

	if err := clnt.Open(d, 0); err != nil {
		b.Fatalf("%v", err)
	}

	for i := 0; i < b.N; i++ {
		for tot := uint32(0); tot < iosize; {
			if n, err := clnt.Read(d, 0, iosize); err != nil {
				b.Fatalf("%v: only got %d of %d bytes", err, len(n), iosize)
			} else {
				tot += uint32(len(n))
			}
		}
	}
}

func benchNull(b *testing.B, msize, iosize uint32, clnt *clnt.Clnt, rootfid *clnt.Fid) {

	d := clnt.FidAlloc()
	if _, err := clnt.Walk(rootfid, d, []string{"null"}); err != nil {
		b.Fatalf("%v", err)
	}

	if err := clnt.Open(d, 1); err != nil {
		b.Fatalf("%v", err)
	}

	data := make([]byte, iosize)
	for i := 0; i < b.N; i++ {
		for tot := uint32(0); tot < iosize; {
			if n, err := clnt.Write(d, data[:iosize-tot], 0); err != nil {
				b.Fatalf("%v: only wrote %d of %d bytes", err, tot, iosize)
			} else {
				tot += uint32(n)
			}
		}
	}
	clnt.Clunk(d)
}

func main() {
	var err error

	flag.Parse()
	f := new(nullfs.NullFS)
	f.Dotu = false
	f.Id = "ufs"
	f.Debuglevel = *debug
	if !f.Start(f) {
		log.Fatalf("Can't happen: Starting the server failed")
	}

	l, err := net.Listen("unix", "")
	if err != nil {
		log.Fatalf("net.Listen: want nil, got %v", err)
	}

	go func() {
		if err = f.StartListener(l); err != nil {
			log.Fatalf("Can not start listener: %v", err)
		}
	}()

	srvAddr := l.Addr().String()
	log.Printf("Server is at %v", srvAddr)

	user := ninep.OsUsers.Uid2User(os.Geteuid())

	if *bench == "all" || *bench == "read" {
		for msize := uint32(8192); msize <= 1048576; msize *= 2 {
			clnt, err := clnt.Mount("unix", srvAddr, "/", msize, user)
			if err != nil {
				log.Fatalf("Connect failed: %v\n", err)
			}
			rootfid := clnt.Root

			fmt.Printf("# read zero msize %v \n", msize)
			for iosize := uint32(1); iosize <= 1048576; iosize *= 2 {
				f := func(b *testing.B) {
					benchZero(b, msize, iosize, clnt, rootfid)
				}
				fmt.Printf("%d  ", iosize)
				r := testing.Benchmark(f)
				fmt.Printf("%v %v\n", r.NsPerOp(), (int64(iosize)*1000000)/(r.NsPerOp()/1000))
			}
			clnt.Unmount()
		}
	}
	if *bench == "write" || *bench == "all" {
		for msize := uint32(8192); msize <= 1048576; msize *= 2 {
			clnt, err := clnt.Mount("unix", srvAddr, "/", msize, user)
			if err != nil {
				log.Fatalf("Connect failed: %v\n", err)
			}
			rootfid := clnt.Root
			fmt.Printf("# write null msize %v \n", msize)
			for iosize := uint32(1); iosize <= 1048576; iosize *= 2 {
				f := func(b *testing.B) {
					benchNull(b, msize, iosize, clnt, rootfid)
				}
				fmt.Printf("%d  ", iosize)
				r := testing.Benchmark(f)
				fmt.Printf("%v %v\n", r.NsPerOp(), (int64(iosize)*1000000)/(r.NsPerOp()/1000))
			}
			clnt.Unmount()
		}
	}
}
