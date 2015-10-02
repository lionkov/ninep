// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nullfs

import (
	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/srv"
	"log"
)

type NullFS struct {
	srv.Srv
	user   ninep.User
	group  ninep.Group
	qidMap map[uint32]int
}

type NullFile struct {
	Name string
}

type Fid struct {
	NullFile
}

var (
	rsrv    NullFS
	dirQids = map[string]*ninep.Qid{
		".":        &ninep.Qid{ninep.QTDIR, 0777, 0},
		"null":     &ninep.Qid{0, 0666, 1},
		"zero":     &ninep.Qid{0, 0444, 2},
		"noaccess": &ninep.Qid{0, 0, 3},
	}
	_ = srv.ReqOps(&NullFS{})
)

func (f *NullFS) Read(r *srv.Req) {
	var count int

	ninep.InitRread(r.Rc, r.Tc.Count)
	fid := r.Fid.Aux.(*Fid)
	log.Printf("read: fid is %v", fid)
	if fid.Name == "." {
		var dirents []byte
		for path, v := range dirQids {
			d := &ninep.Dir{
				Qid:  *v,
				Type: uint16(v.Type),
				Mode: uint32(v.Type) | v.Version,
				Name: path,
				Uid:  "root",
				Gid:  "root",
			}
			b := ninep.PackDir(d, true)
			dirents = append(dirents, b...)
			count += len(b)
		}

		switch {
		case r.Tc.Offset > uint64(len(dirents)):
			count = 0
		case len(dirents[r.Tc.Offset:]) > int(r.Tc.Count):
			count = int(r.Tc.Count)
		default:
			count = len(dirents[r.Tc.Offset:])
		}

		if count == 0 && int(r.Tc.Offset) < len(dirents) && len(dirents) > 0 {
			r.RespondError(&ninep.Error{"too small read size for dir entry", ninep.EINVAL})
			return
		}
		copy(r.Rc.Data, dirents[r.Tc.Offset:int(r.Tc.Offset)+count])
	} else {
		if fid.Name == "zero" {
			count = int(r.Tc.Count)
		}
	}
	ninep.SetRreadCount(r.Rc, uint32(count))
	r.Respond()
}

func (*NullFS) Clunk(req *srv.Req) { req.RespondRclunk() }

func (f *NullFS) Write(r *srv.Req) {
	var count uint32

	if r.Tc.Qid.Path == 2 {
		count = r.Tc.Count
	}
	ninep.SetRreadCount(r.Rc, uint32(count))
	r.Respond()
}

func (f *NullFS) Walk(req *srv.Req) {
	tc := req.Tc

	if len(tc.Wname) > 1 && tc.Qid.Type != ninep.QTDIR {
		req.RespondError(ninep.ENOENT)
		return
	}

	if req.Newfid.Aux == nil {
		req.Newfid.Aux = &Fid{NullFile: NullFile{Name: "."}}
	}

	if len(tc.Wname) == 0 {
		req.RespondRwalk([]ninep.Qid{})
		return
	}

	if q, ok := dirQids[tc.Wname[0]]; !ok {
		req.RespondError(ninep.ENOENT)
	} else {
		req.Newfid.Aux.(*Fid).Name = tc.Wname[0]
		req.RespondRwalk([]ninep.Qid{*q})
	}
}

func (f *NullFS) Create(r *srv.Req) {
	r.RespondError(srv.Eperm)
}

func (f *NullFS) Open(r *srv.Req) {
	r.RespondRopen(&r.Tc.Qid, 0)
}

func (f *NullFS) Remove(r *srv.Req) {
	r.RespondError(srv.Eperm)
}

func (f *NullFS) Stat(req *srv.Req) {
	/*
		fid := req.Fid.Aux.(*Fid)
		st, err := dir2Dir(fid.path, fid.st, req.Conn.Dotu, req.Conn.Srv.Upool)
		if err != nil {
			req.RespondError(err)
			return
		}
		req.RespondRstat(st)
	*/
	req.RespondError(srv.Eperm)
}

func (f *NullFS) Wstat(r *srv.Req) {
	r.RespondError(srv.Eperm)
}

func (*NullFS) Attach(req *srv.Req) {

	if req.Afid != nil {
		req.RespondError(srv.Enoauth)
		return
	}

	req.Fid.Aux = &Fid{NullFile: NullFile{Name: "."}}

	req.RespondRattach(dirQids["."])
}

func NewNullFS(debug int) (*NullFS, error) {
	f := &NullFS{}
	//	f.user = ninep.OsUsers.Uid2User(os.Geteuid())
	//f.group = ninep.OsUsers.Gid2Group(os.Getegid())

	f.Dotu = true
	f.Debuglevel = debug
	return f, nil
}
