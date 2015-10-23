// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clnt

import (
	"github.com/lionkov/ninep"
	"net"
)

// Creates an authentication fid for the specified user. Returns the fid, if
// successful, or an Error.
func (clnt *Clnt) Auth(user ninep.User, aname string) (*Fid, error) {
	fid := clnt.FidAlloc()
	tc := clnt.NewFcall()
	err := ninep.PackTauth(tc, fid.Fid, user.Name(), aname, uint32(user.Id()), clnt.Dotu)
	if err != nil {
		return nil, err
	}

	_, err = clnt.Rpc(tc)
	if err != nil {
		return nil, err
	}

	fid.Iounit = clnt.Msize - ninep.IOHDRSZ
	fid.User = user
	fid.walked = true
	return fid, nil
}

// Creates a fid for the specified user that points to the root
// of the file server's file tree. Returns a Fid pointing to the root,
// if successful, or an Error.
func (clnt *Clnt) Attach(afid *Fid, user ninep.User, aname string) (*Fid, error) {
	var afno uint32

	if afid != nil {
		afno = afid.Fid
	} else {
		afno = ninep.NOFID
	}

	fid := clnt.FidAlloc()
	tc := clnt.NewFcall()

	err := ninep.PackTattach(tc, fid.Fid, afno, user.Name(), aname, uint32(user.Id()), clnt.Dotu)
	if err != nil {
		return nil, err
	}

	rc, err := clnt.Rpc(tc)
	if err != nil {
		return nil, err
	}

	fid.Qid = rc.Qid
	fid.User = user
	fid.walked = true
	clnt.Root = fid
	return fid, nil
}

// Connects to a file server and attaches to it as the specified user.
func Mount(ntype, addr, aname string, msize uint32, user ninep.User) (*Clnt, error) {
	c, e := net.Dial(ntype, addr)
	if e != nil {
		return nil, &ninep.Error{e.Error(), ninep.EIO}
	}

	return MountConn(c, aname, msize, user)
}

func MountConn(c net.Conn, aname string, msize uint32, user ninep.User) (*Clnt, error) {
	clnt, err := Connect(c, msize+ninep.IOHDRSZ, true)
	if err != nil {
		return nil, err
	}

	fid, err := clnt.Attach(nil, user, aname)
	if err != nil {
		clnt.Unmount()
		return nil, err
	}

	clnt.Root = fid
	return clnt, nil
}

// Closes the connection to the file sever.
func (clnt *Clnt) Unmount() {
	clnt.Lock()
	clnt.err = &ninep.Error{"connection closed", ninep.EIO}
	clnt.conn.Close()
	clnt.Unlock()
}
