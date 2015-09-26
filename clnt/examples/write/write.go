package main

import (
	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/clnt"
	"flag"
	"io"
	"log"
	"os"
)

var debuglevel = flag.Int("d", 0, "debuglevel")
var addr = flag.String("addr", "127.0.0.1:5640", "network address")
var msize = flag.Uint("m", 8192, "Msize for 9p")

func main() {
	var n, m int
	var user ninep.User
	var err error
	var c *clnt.Clnt
	var file *clnt.File
	var buf []byte

	flag.Parse()
	user = ninep.OsUsers.Uid2User(os.Geteuid())
	clnt.DefaultDebuglevel = *debuglevel
	c, err = clnt.Mount("tcp", *addr, "", uint32(*msize), user)
	if err != nil {
		goto error
	}

	if flag.NArg() != 1 {
		log.Println("invalid arguments")
		return
	}

	file, err = c.FOpen(flag.Arg(0), ninep.OWRITE|ninep.OTRUNC)
	if err != nil {
		file, err = c.FCreate(flag.Arg(0), 0666, ninep.OWRITE)
		if err != nil {
			goto error
		}
	}

	buf = make([]byte, 8192)
	for {
		n, err = os.Stdin.Read(buf)
		if err != nil && err != io.EOF {
			goto error
		}

		if n == 0 {
			break
		}

		m, err = file.Write(buf[0:n])
		if err != nil {
			goto error
		}

		if m != n {
			err = &ninep.Error{"short write", 0}
			goto error
		}
	}

	file.Close()
	return

error:
	log.Println("Error", err)
}
