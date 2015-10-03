// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nullfs

import (
	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/srv"
)

const (
	Qroot     = 'r'
	Qnull     = 'n'
	Qzero     = 'z'
	Qnoaccess = 'N'
)

type NullFS struct {
	srv.Srv
}

type NullFile struct {
	Name string
	*ninep.Qid
}

type Fid struct {
	NullFile
}

var (
	rsrv    NullFS
	dirQids = map[string]*ninep.Qid{
		".":        &ninep.Qid{Type: ninep.QTDIR, Version: 0777, Path: Qroot},
		"null":     &ninep.Qid{Type: 0, Version: 0666, Path: Qnull},
		"zero":     &ninep.Qid{Type: 0, Version: 0444, Path: Qzero},
		"noaccess": &ninep.Qid{Type: 0, Version: 0, Path: Qnoaccess},
	}
	// Verify that we correctly implement ReqOps
	_ = srv.ReqOps(&NullFS{})
)

func (f *NullFS) Read(r *srv.Req) {
	var count int

	ninep.InitRread(r.Rc, r.Tc.Count)
	fid := r.Fid.Aux.(*Fid)

	if fid.Qid.Path == Qroot {
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

		// TODO: put this boilerplate into a helper function.
		switch {
		case r.Tc.Offset > uint64(len(dirents)):
			count = 0
		case len(dirents[r.Tc.Offset:]) > int(r.Tc.Count):
			count = int(r.Tc.Count)
		default:
			count = len(dirents[r.Tc.Offset:])
		}

		if count == 0 && int(r.Tc.Offset) < len(dirents) && len(dirents) > 0 {
			r.RespondError(&ninep.Error{Err: "too small read size for dir entry", Errornum: ninep.EINVAL})
			return
		}
		copy(r.Rc.Data, dirents[r.Tc.Offset:int(r.Tc.Offset)+count])
	} else {
		if fid.Qid.Path == Qzero {
			count = int(r.Tc.Count)
		}
	}
	ninep.SetRreadCount(r.Rc, uint32(count))
	r.Respond()
}

func (*NullFS) Clunk(req *srv.Req) { req.RespondRclunk() }

// Write handles writes for writeable files and always succeeds.
// Only the null files has w so we don't bother checking Path.
func (f *NullFS) Write(r *srv.Req) {
	var count uint32
	ninep.SetRreadCount(r.Rc, uint32(count))
	r.Respond()
}

func (f *NullFS) Walk(req *srv.Req) {
	tc := req.Tc

	if len(tc.Wname) > 1 && tc.Qid.Type != ninep.QTDIR {
		req.RespondError(ninep.ENOENT)
		return
	}

	// The most common case is walking from '.', so we initialize to '.' and fix it up
	// later if needed.
	if req.Newfid.Aux == nil {
		req.Newfid.Aux = &Fid{NullFile: NullFile{Name: ".", Qid: dirQids["."]}}
	}

	if len(tc.Wname) == 0 {
		req.RespondRwalk([]ninep.Qid{})
		return
	}

	if q, ok := dirQids[tc.Wname[0]]; !ok {
		req.RespondError(ninep.ENOENT)
	} else {
		req.Newfid.Aux.(*Fid).Name = tc.Wname[0]
		req.Newfid.Aux.(*Fid).Qid = q
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
	fid := req.Fid.Aux.(*Fid)
	d := &ninep.Dir{
		Qid:  *fid.Qid,
		Type: uint16(fid.Type),
		Mode: uint32(fid.Type) | fid.Version,
		Name: fid.Name,
		Uid:  "root",
		Gid:  "root",
	}
	req.RespondRstat(d)
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
	f.Dotu = true
	f.Debuglevel = debug
	return f, nil
}
