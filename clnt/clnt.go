// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The clnt package provides definitions and functions used to implement
// a 9P2000 file client.
package clnt

import (
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/lionkov/ninep"
)

// Debug flags
const (
	DbgPrintFcalls  = (1 << iota) // print all 9P messages on stderr
	DbgPrintPackets               // print the raw packets on stderr
	DbgLogFcalls                  // keep the last N 9P messages (can be accessed over http)
	DbgLogPackets                 // keep the last N 9P messages (can be accessed over http)
)

type StatsOps interface {
	statsRegister()
	statsUnregister()
}

// The Clnt type represents a 9P2000 client. The client is connected to
// a 9P2000 file server and its methods can be used to access and manipulate
// the files exported by the server.
type Clnt struct {
	sync.Mutex
	Debuglevel int    // =0 don't print anything, >0 print Fcalls, >1 print raw packets
	Msize      uint32 // Maximum size of the 9P messages
	Dotu       bool   // If true, 9P2000.u protocol is spoken
	Root       *Fid   // Fid that points to the rood directory
	Id         string // Used when printing debug messages
	Log        *ninep.Logger

	conn     net.Conn
	tagpool  *pool
	fidpool  *pool
	reqout   chan *Req
	done     chan bool
	reqfirst *Req
	reqlast  *Req
	err      error

	reqchan chan *Req
	tchan   chan *ninep.Fcall

	next, prev *Clnt
}

// A Fid type represents a file on the server. Fids are used for the
// low level methods that correspond directly to the 9P2000 message requests
type Fid struct {
	sync.Mutex
	Clnt       *Clnt // Client the fid belongs to
	Iounit     uint32
	ninep.Qid         // The Qid description for the file
	Mode       uint8  // Open mode (one of ninep.O* values) (if file is open)
	Fid        uint32 // Fid number
	ninep.User        // The user the fid belongs to
	walked     bool   // true if the fid points to a walked file on the server
}

// The file is similar to the Fid, but is used in the high-level client
// interface.
type File struct {
	fid    *Fid
	offset uint64
}

type pool struct {
	sync.Mutex
	need  int
	nchan chan uint32
	maxid uint32
	imap  []byte
}

type Req struct {
	sync.Mutex
	Clnt       *Clnt
	Tc         *ninep.Fcall
	Rc         *ninep.Fcall
	Err        error
	Done       chan *Req
	Sent       chan bool
	tag        uint16
	prev, next *Req
	fid        *Fid
}

type ClntList struct {
	sync.Mutex
	clntList, clntLast *Clnt
}

var clnts *ClntList
var DefaultDebuglevel int
var DefaultLogger *ninep.Logger

func (clnt *Clnt) Rpcnb(r *Req) error {
	var tag uint16

	if r.Tc.Type == ninep.Tversion {
		tag = ninep.NOTAG
	} else {
		tag = r.tag
	}

	ninep.SetTag(r.Tc, tag)
	clnt.Lock()
	if clnt.err != nil {
		clnt.Unlock()
		return clnt.err
	}

	if clnt.reqlast != nil {
		clnt.reqlast.next = r
	} else {
		clnt.reqfirst = r
	}

	r.prev = clnt.reqlast
	clnt.reqlast = r
	clnt.Unlock()

	clnt.reqout <- r
	return nil
}

func (clnt *Clnt) Rpc(tc *ninep.Fcall) (rc *ninep.Fcall, err error) {
	r := clnt.ReqAlloc()
	r.Tc = tc
	r.Done = make(chan *Req)
	r.Sent = make(chan bool)
	err = clnt.Rpcnb(r)
	if err != nil {
		return
	}

	<-r.Done
	rc = r.Rc
	err = r.Err
	clnt.ReqFree(r)
	return
}

func (clnt *Clnt) recv() {
	var err error
	var buf []byte

	err = nil
	pos := 0
	for {
		// Connect can change the client Msize.
		clntmsize := int(atomic.LoadUint32(&clnt.Msize))
		if len(buf) < clntmsize {
			b := make([]byte, clntmsize*8)
			copy(b, buf[0:pos])
			buf = b
			b = nil
		}

		n, oerr := clnt.conn.Read(buf[pos:])

		if oerr != nil || n == 0 {
			err = &ninep.Error{oerr.Error(), ninep.EIO}
			clnt.Lock()
			clnt.err = err
			clnt.Unlock()
			goto closed
		}

		pos += n
		for pos > 4 {
			sz, _ := ninep.Gint32(buf)
			if pos < int(sz) {
				if len(buf) < int(sz) {
					b := make([]byte, atomic.LoadUint32(&clnt.Msize)*8)
					copy(b, buf[0:pos])
					buf = b
					b = nil
				}

				break
			}

			fc, err, fcsize := ninep.Unpack(buf, clnt.Dotu)
			clnt.Lock()
			if err != nil {
				clnt.err = err
				clnt.conn.Close()
				clnt.Unlock()
				goto closed
			}

			if clnt.Debuglevel > 0 {
				clnt.logFcall(fc)
				if clnt.Debuglevel&DbgPrintPackets != 0 {
					log.Println("}-}", clnt.Id, fmt.Sprint(fc.Pkt))
				}

				if clnt.Debuglevel&DbgPrintFcalls != 0 {
					log.Println("}}}", clnt.Id, fc.String())
				}
			}

			var r *Req = nil
			for r = clnt.reqfirst; r != nil; r = r.next {
				if r.Tc.Tag == fc.Tag {
					break
				}
			}

			if r == nil {
				clnt.err = &ninep.Error{"unexpected response", ninep.EINVAL}
				clnt.conn.Close()
				clnt.Unlock()
				goto closed
			}

			// Good clean fun. There's a race where you can get the response BEFORE the loop
			// in send() thinks it is done with the request. So we have to block on
			// it being sent, because we really can't dequeue any more requests until
			// this one is wrapped up. TODO: consider a goroutine per fid for this,
			// if we need it. The reason I feel this is safe is that if we got a tag back
			// for a request we sent, then r.Sent should be written. If we got a tag
			// back for a request we did not sent, we won't find it here anyway.
			<-r.Sent

			r.Rc = fc
			switch {
			case r.next == nil && r.prev == nil:
				clnt.reqlast = nil
				clnt.reqfirst = nil
			case r.next == nil:
				clnt.reqlast = r.prev
				r.prev.next = nil
				r.prev = nil
			case r.prev == nil:
				clnt.reqfirst = r.next
				r.next.prev = nil
				r.next = nil
			default:
				r.next.prev = r.prev
				r.prev.next = r.next
				r.next = nil
				r.prev = nil
			}
			clnt.Unlock()

			if r.Tc.Type != r.Rc.Type-1 {
				if r.Rc.Type != ninep.Rerror {
					r.Err = &ninep.Error{"invalid response", ninep.EINVAL}
				} else {
					if r.Err == nil {
						r.Err = &ninep.Error{r.Rc.Error, r.Rc.Errornum}
					}
				}
			}

			if r.Done != nil {
				r.Done <- r
			}

			pos -= fcsize
			buf = buf[fcsize:]
		}
	}

closed:
	clnt.done <- true

	/* send error to all pending requests */
	clnt.Lock()
	r := clnt.reqfirst
	clnt.reqfirst = nil
	clnt.reqlast = nil
	if err == nil {
		err = clnt.err
	}
	clnt.Unlock()
	for r != nil {
		next := r.next
		r.Err = err
		r.next = nil
		r.prev = nil
		if r.Done != nil {
			r.Done <- r
		}
		r = next
	}

	clnts.Lock()
	if clnt.prev != nil {
		clnt.prev.next = clnt.next
		clnt.prev = nil
	} else {
		clnts.clntList = clnt.next
	}

	if clnt.next != nil {
		clnt.next.prev = clnt.prev
		clnt.next = nil
	} else {
		clnts.clntLast = clnt.prev
	}
	clnts.Unlock()

	if sop, ok := (interface{}(clnt)).(StatsOps); ok {
		sop.statsUnregister()
	}
}

func (clnt *Clnt) send() {
	for {
		select {
		case <-clnt.done:
			return

		case req := <-clnt.reqout:
			if clnt.Debuglevel > 0 {
				clnt.logFcall(req.Tc)
				if clnt.Debuglevel&DbgPrintPackets != 0 {
					log.Print("{-{", clnt.Id, fmt.Sprint(req.Tc.Pkt))
					if req.Tc.Type == ninep.Twrite {
						log.Print(fmt.Sprint(req.Tc.Data))
					}
					fmt.Println("")
				}

				if clnt.Debuglevel&DbgPrintFcalls != 0 {
					log.Println("{{{", clnt.Id, req.Tc.String())
				}
			}

			for buf := req.Tc.Pkt; len(buf) > 0; {
				n, err := clnt.conn.Write(buf)
				if err != nil {
					/* just close the socket, will get signal on clnt.done */
					clnt.conn.Close()
					break
				}

				buf = buf[n:]
			}
			if req.Tc.Type == ninep.Twrite {
				for buf := req.Tc.Data; len(buf) > 0; {
					n, err := clnt.conn.Write(buf)
					if err != nil {
						/* just close the socket, will get signal on clnt.done */
						clnt.conn.Close()
						break
					}

					buf = buf[n:]
				}
			}
			req.Sent <- true
		}
	}
}

// Creates and initializes a new Clnt object. Doesn't send any data
// on the wire.
func NewClnt(c net.Conn, msize uint32, dotu bool) *Clnt {
	clnt := new(Clnt)
	clnt.conn = c
	clnt.Msize = msize
	clnt.Dotu = dotu
	clnt.Debuglevel = DefaultDebuglevel
	clnt.Log = DefaultLogger
	clnt.Id = c.RemoteAddr().String() + ":"
	clnt.tagpool = newPool(uint32(ninep.NOTAG))
	clnt.fidpool = newPool(ninep.NOFID)
	clnt.reqout = make(chan *Req)
	clnt.done = make(chan bool)
	clnt.reqchan = make(chan *Req, 16)
	clnt.tchan = make(chan *ninep.Fcall, 16)

	go clnt.recv()
	go clnt.send()

	clnts.Lock()
	if clnts.clntLast != nil {
		clnts.clntLast.next = clnt
	} else {
		clnts.clntList = clnt
	}

	clnt.prev = clnts.clntLast
	clnts.clntLast = clnt
	clnts.Unlock()

	if sop, ok := (interface{}(clnt)).(StatsOps); ok {
		sop.statsRegister()
	}

	return clnt
}

// Establishes a new socket connection to the 9P server and creates
// a client object for it. Negotiates the dialect and msize for the
// connection. Returns a Clnt object, or Error.
func Connect(c net.Conn, msize uint32, dotu bool) (*Clnt, error) {
	clnt := NewClnt(c, msize, dotu)
	ver := "9P2000"
	if clnt.Dotu {
		ver = "9P2000.u"
	}

	clntmsize := atomic.LoadUint32(&clnt.Msize)
	tc := ninep.NewFcall(clntmsize)
	err := ninep.PackTversion(tc, clntmsize, ver)
	if err != nil {
		return nil, err
	}

	rc, err := clnt.Rpc(tc)
	if err != nil {
		return nil, err
	}

	if rc.Msize < atomic.LoadUint32(&clnt.Msize) {
		atomic.StoreUint32(&clnt.Msize, rc.Msize)
	}

	clnt.Dotu = rc.Version == "9P2000.u" && clnt.Dotu
	return clnt, nil
}

// Creates a new Fid object for the client
func (clnt *Clnt) FidAlloc() *Fid {
	fid := new(Fid)
	fid.Fid = clnt.fidpool.getId()
	fid.Clnt = clnt

	return fid
}

func (clnt *Clnt) NewFcall() *ninep.Fcall {
	select {
	case tc := <-clnt.tchan:
		return tc
	default:
	}
	return ninep.NewFcall(atomic.LoadUint32(&clnt.Msize))
}

func (clnt *Clnt) FreeFcall(fc *ninep.Fcall) {
	if fc != nil && len(fc.Buf) >= int(atomic.LoadUint32(&clnt.Msize)) {
		select {
		case clnt.tchan <- fc:
			break
		default:
		}
	}
}

func (clnt *Clnt) ReqAlloc() *Req {
	var req *Req
	select {
	case req = <-clnt.reqchan:
		break
	default:
		req = new(Req)
		req.Clnt = clnt
		req.tag = uint16(clnt.tagpool.getId())
	}
	return req
}

func (clnt *Clnt) ReqFree(req *Req) {
	clnt.FreeFcall(req.Tc)
	req.Tc = nil
	req.Rc = nil
	req.Err = nil
	req.Done = nil

	select {
	case clnt.reqchan <- req:
		break
	default:
		clnt.tagpool.putId(uint32(req.tag))
	}
}

func NewFile(f *Fid, offset uint64) *File {
	return &File{f, offset}
}

func (f *File) Fid() *Fid {
	return f.fid
}

func (clnt *Clnt) logFcall(fc *ninep.Fcall) {
	if clnt.Debuglevel&DbgLogPackets != 0 {
		pkt := make([]byte, len(fc.Pkt))
		copy(pkt, fc.Pkt)
		clnt.Log.Log(pkt, clnt, DbgLogPackets)
	}

	if clnt.Debuglevel&DbgLogFcalls != 0 {
		f := new(ninep.Fcall)
		*f = *fc
		f.Pkt = nil
		clnt.Log.Log(f, clnt, DbgLogFcalls)
	}
}

func init() {
	clnts = new(ClntList)
	if sop, ok := (interface{}(clnts)).(StatsOps); ok {
		sop.statsRegister()
	}
}
