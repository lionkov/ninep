// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clnt

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/srv/nullfs"
	"github.com/lionkov/ninep/srv/ufs"
)

var debug = flag.Int("debug", 0, "print debug messages")
var numDir = flag.Int("numdir", 16384, "Number of directory entries for readdir testing")
var numMount = flag.Int("nummount", 65536, "Number of mounts to make in TestAttach")

func TestTversionBadConn(t *testing.T) {
	conn0, conn1 := net.Pipe()
	conn1.Close()

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	_, err := MountConn(conn0, "/", 8192, user)
	ninepErr, ok := err.(*ninep.Error)
	if !ok {
		t.Fatalf("Attach expected ninep.Error, got %T: %v", err, err)
	}
	if ninepErr.Errornum != ninep.EIO {
		t.Fatalf("Attach expected %v, got %v", ninep.EIO, ninepErr.Errornum)
	}
}

func TestAttachWithoutTversion(t *testing.T) {
	var err error
	flag.Parse()
	ufs := new(ufs.Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Start(ufs)

	t.Log("ufs starting\n")
	// determined by build tags
	//extraFuncs()
	l, err := net.Listen("unix", "")
	if err != nil {
		t.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	t.Logf("Server is at %v", srvAddr)
	go func() {
		if err = ufs.StartListener(l); err != nil {
			t.Fatalf("Can not start listener: %v", err)
		}
	}()
	var conn net.Conn
	if conn, err = net.Dial("unix", srvAddr); err != nil {
		t.Fatalf("%v", err)
	}

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt := NewClnt(conn, 8192, false)
	_, err = clnt.Attach(nil, user, "/tmp")
	if err == nil {
		t.Fatalf("Attach without Tversion got nil, wanted err")
	}
}

func TestAttach(t *testing.T) {
	var err error
	flag.Parse()
	ufs := new(ufs.Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Start(ufs)

	t.Log("ufs starting\n")
	// determined by build tags
	//extraFuncs()
	l, err := net.Listen("unix", "")
	if err != nil {
		t.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	t.Logf("Server is at %v", srvAddr)
	go func() {
		if err = ufs.StartListener(l); err != nil {
			t.Fatalf("Can not start listener: %v", err)
		}
	}()
	// run enough mounts to maybe let the race detector trip.
	// The default, 1024, is lower than I'd like, but some environments don't
	// let you do a huge number, as they throttle the accept rate.
	for i := 0; i < *numMount; i++ {
		user := ninep.OsUsers.Uid2User(os.Geteuid())
		clnt, err := Mount("unix", srvAddr, "/", 8192, user)
		if err != nil {
			t.Fatalf("Connect failed: %v\n", err)
		}

		clnt.Unmount()
	}
}

func TestAttachOpenReaddir(t *testing.T) {
	var err error
	flag.Parse()
	ufs := new(ufs.Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Start(ufs)
	tmpDir, err := ioutil.TempDir("", "go9")
	if err != nil {
		t.Fatal("Can't create temp directory")
	}
	//defer os.RemoveAll(tmpDir)
	ufs.Root = tmpDir

	t.Logf("ufs starting in %v\n", tmpDir)
	// determined by build tags
	//extraFuncs()
	l, err := net.Listen("unix", "")
	if err != nil {
		t.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	t.Logf("Server is at %v", srvAddr)
	go func() {
		if err = ufs.StartListener(l); err != nil {
			t.Fatalf("Can not start listener: %v", err)
		}
	}()

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt, err := Mount("unix", srvAddr, "/", 8192, user)
	if err != nil {
		t.Fatalf("Connect failed: %v\n", err)
	}
	rootfid := clnt.Root
	clnt.Debuglevel = 0 // *debug
	t.Logf("mounted, rootfid %v\n", rootfid)

	dirfid := clnt.FidAlloc()
	if _, err = clnt.Walk(rootfid, dirfid, []string{"."}); err != nil {
		t.Fatalf("%v", err)
	}

	// Now create a whole bunch of files to test readdir
	for i := 0; i < *numDir; i++ {
		f := fmt.Sprintf(path.Join(tmpDir, fmt.Sprintf("%d", i)))
		if err := ioutil.WriteFile(f, []byte(f), 0600); err != nil {
			t.Fatalf("Create %v: got %v, want nil", f, err)
		}
	}

	if err = clnt.Open(dirfid, 0); err != nil {
		t.Fatalf("%v", err)
	}
	var b []byte
	if b, err = clnt.Read(dirfid, 0, 64*1024); err != nil {
		t.Fatalf("%v", err)
	}
	var i, amt int
	var offset uint64
	err = nil
	found := make([]int, *numDir)
	fail := false
	for err == nil {
		if b, err = clnt.Read(dirfid, offset, 64*1024); err != nil {
			t.Fatalf("%v", err)
		}
		t.Logf("clnt.Read returns [%v,%v]", len(b), err)
		if len(b) == 0 {
			break
		}
		for b != nil && len(b) > 0 {
			var d *ninep.Dir
			if d, b, amt, err = ninep.UnpackDir(b, ufs.Dotu); err != nil {
				t.Errorf("UnpackDir returns %v", err)
				break
			} else {
				if *debug > 128 {
					t.Logf("Entry %d: %v", i, d)
				}
				i++
				offset += uint64(amt)
				ix, err := strconv.Atoi(d.Name)
				if err != nil {
					t.Errorf("File name %v is wrong; %v (dirent %v)", d.Name, err, d)
					continue
				}
				if found[ix] > 0 {
					t.Errorf("Element %d already returned %d times", ix, found[ix])
				}
				found[ix]++
			}
		}
	}
	if i != *numDir {
		t.Fatalf("Reading %v: got %d entries, wanted %d, err %v", tmpDir, i, *numDir, err)
	}
	if fail {
		t.Fatalf("I give up")
	}

	t.Logf("-----------------------------> Alternate form, using readdir and File")
	// Alternate form, using readdir and File
	dirfile, err := clnt.FOpen(".", ninep.OREAD)
	if err != nil {
		t.Fatalf("%v", err)
	}
	i, amt, offset = 0, 0, 0
	err = nil
	passes := 0

	found = make([]int, *numDir)
	fail = false
	for err == nil {
		d, err := dirfile.Readdir(*numDir)
		if err != nil && err != io.EOF {
			t.Errorf("%v", err)
		}

		t.Logf("d is %v", d)
		if len(d) == 0 {
			break
		}
		for _, v := range d {
			ix, err := strconv.Atoi(v.Name)
			if err != nil {
				t.Errorf("File name %v is wrong; %v (dirent %v)", v.Name, err, v)
				continue
			}
			if found[ix] > 0 {
				t.Errorf("Element %d already returned %d times", ix, found[ix])
			}
			found[ix]++
		}
		i += len(d)
		if i >= *numDir {
			break
		}
		if passes > *numDir {
			t.Fatalf("%d iterations, %d read: no progress", passes, i)
		}
		passes++
	}
	if i != *numDir {
		t.Fatalf("Readdir %v: got %d entries, wanted %d", tmpDir, i, *numDir)
	}
}

func TestRename(t *testing.T) {
	var err error
	flag.Parse()
	ufs := new(ufs.Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Msize = 8192
	ufs.Start(ufs)

	tmpDir, err := ioutil.TempDir("", "go9")
	if err != nil {
		t.Fatal("Can't create temp directory")
	}
	defer os.RemoveAll(tmpDir)
	ufs.Root = tmpDir
	t.Logf("ufs starting in %v", tmpDir)
	// determined by build tags
	//extraFuncs()
	l, err := net.Listen("unix", "")
	if err != nil {
		t.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	t.Logf("Server is at %v", srvAddr)
	go func() {
		if err = ufs.StartListener(l); err != nil {
			t.Fatalf("Can not start listener: %v", err)
		}
	}()

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt, err := Mount("unix", srvAddr, "/", 8192, user)
	if err != nil {
		t.Fatalf("Connect failed: %v\n", err)
	}
	rootfid := clnt.Root
	clnt.Debuglevel = 0 // *debug
	t.Logf("attached to %v, rootfid %v\n", tmpDir, rootfid)
	// OK, create a file behind nineps back and then rename it.
	b := make([]byte, 0)
	from := path.Join(tmpDir, "a")
	to := path.Join(tmpDir, "b")
	if err = ioutil.WriteFile(from, b, 0666); err != nil {
		t.Fatalf("%v", err)
	}

	f := clnt.FidAlloc()
	if _, err = clnt.Walk(rootfid, f, []string{"a"}); err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("Walked to a")
	if _, err := clnt.Stat(f); err != nil {
		t.Fatalf("%v", err)
	}
	if err := clnt.FSync(f); err != nil {
		t.Fatalf("%v", err)
	}
	if err = clnt.Rename(f, "b"); err != nil {
		t.Errorf("%v", err)
	}
	// the old one should be gone, and the new one should be there.
	if _, err = ioutil.ReadFile(from); err == nil {
		t.Errorf("ReadFile(%v): got nil, want err", from)
	}

	if _, err = ioutil.ReadFile(to); err != nil {
		t.Errorf("ReadFile(%v): got %v, want nil", from, err)
	}

	// now try with an absolute path, which is supported on ufs servers.
	// It's not guaranteed to work on all servers, but it is hugely useful
	// on those that can do it -- which is almost all of them, save Plan 9
	// of course.
	from = to
	if err = clnt.Rename(f, "c"); err != nil {
		t.Errorf("%v", err)
	}

	// the old one should be gone, and the new one should be there.
	if _, err = ioutil.ReadFile(from); err == nil {
		t.Errorf("ReadFile(%v): got nil, want err", from)
	}

	to = path.Join(tmpDir, "c")
	if _, err = ioutil.ReadFile(to); err != nil {
		t.Errorf("ReadFile(%v): got %v, want nil", to, err)
	}

	// Make sure they can't walk out of the root.

	from = to
	if err = clnt.Rename(f, "../../../../d"); err != nil {
		t.Errorf("%v", err)
	}

	// the old one should be gone, and the new one should be there.
	if _, err = ioutil.ReadFile(from); err == nil {
		t.Errorf("ReadFile(%v): got nil, want err", from)
	}

	to = path.Join(tmpDir, "d")
	if _, err = ioutil.ReadFile(to); err != nil {
		t.Errorf("ReadFile(%v): got %v, want nil", to, err)
	}

}

func BenchmarkVersion(b *testing.B) {
	ufs := new(ufs.Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Msize = 8192
	ufs.Start(ufs)

	l, err := net.Listen("unix", "")
	if err != nil {
		b.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	b.Logf("Server is at %v", srvAddr)
	go func() {
		if err = ufs.StartListener(l); err != nil {
			b.Fatalf("Can not start listener: %v", err)
		}
		b.Fatalf("Listener returned")
	}()
	var conn net.Conn
	for i := 0; i < b.N; i++ {
		if conn, err = net.Dial("unix", srvAddr); err != nil {
			// Sometimes, things just happen.
			//b.Logf("%v", err)
		} else {
			conn.Close()
		}
	}

}

func BenchmarkAttach(b *testing.B) {
	var err error
	flag.Parse()
	n, err := nullfs.NewNullFS(*debug)
	if err != nil {
		b.Fatalf("could not start NullFS: %v", err)
	}
	n.Dotu = false
	n.Id = "nullfs"
	n.Start(n)

	b.Log("nullfs starting\n")
	l, err := net.Listen("unix", "")
	if err != nil {
		b.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	b.Logf("Server is at %v", srvAddr)
	go func() {
		if err = n.StartListener(l); err != nil {
			b.Fatalf("Can not start listener: %v", err)
		}
	}()
	user := ninep.OsUsers.Uid2User(os.Geteuid())
	for i := 0; i < b.N; i++ {
		var conn net.Conn
		if conn, err = net.Dial("unix", srvAddr); err != nil {
			b.Fatalf("%v", err)
		} else {
			b.Logf("Got a conn, %v\n", conn)
		}

		clnt, err := Mount("unix", srvAddr, "/", 8192, user)
		if err != nil {
			b.Fatalf("Connect failed: %v\n", err)
		}

		clnt.Unmount()
	}
}

func BenchmarkRootWalk(b *testing.B) {
	ufs := new(ufs.Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Msize = 8192
	ufs.Start(ufs)

	l, err := net.Listen("unix", "")
	if err != nil {
		b.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	go func() {
		if err = ufs.StartListener(l); err != nil {
			b.Fatalf("Can not start listener: %v", err)
		}
		b.Fatalf("Listener returned")
	}()

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt, err := Mount("unix", srvAddr, "/", 8192, user)
	if err != nil {
		b.Fatalf("Connect failed: %v\n", err)
	}
	rootfid := clnt.Root
	clnt.Debuglevel = 0 // *debug

	for i := 0; i < b.N; i++ {
		f := clnt.FidAlloc()
		if _, err = clnt.Walk(rootfid, f, []string{"bin"}); err != nil {
			b.Fatalf("%v", err)
		}
	}
}
func BenchmarkRootWalkBadFid(b *testing.B) {
	ufs := new(ufs.Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Msize = 8192
	ufs.Start(ufs)

	l, err := net.Listen("unix", "")
	if err != nil {
		b.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	go func() {
		if err = ufs.StartListener(l); err != nil {
			b.Fatalf("Can not start listener: %v", err)
		}
		b.Fatalf("Listener returned")
	}()

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt, err := Mount("unix", srvAddr, "/", 8192, user)
	if err != nil {
		b.Fatalf("Connect failed: %v\n", err)
	}
	rootfid := clnt.Root
	clnt.Debuglevel = 0 // *debug

	rootfid.Fid++
	for i := 0; i < b.N; i++ {
		if _, err = clnt.Walk(rootfid, rootfid, []string{"bin"}); err == nil {
			b.Fatalf("Did not get an expected error on walking a bad fid!")
		}
	}
}
