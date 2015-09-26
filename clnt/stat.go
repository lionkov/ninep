// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clnt

import "github.com/lionkov/ninep"

// Returns the metadata for the file associated with the Fid, or an Error.
func (clnt *Clnt) Stat(fid *Fid) (*ninep.Dir, error) {
	tc := clnt.NewFcall()
	err := ninep.PackTstat(tc, fid.Fid)
	if err != nil {
		return nil, err
	}

	rc, err := clnt.Rpc(tc)
	if err != nil {
		return nil, err
	}

	return &rc.Dir, nil
}

// Returns the metadata for a named file, or an Error.
func (clnt *Clnt) FStat(path string) (*ninep.Dir, error) {
	fid, err := clnt.FWalk(path)
	if err != nil {
		return nil, err
	}

	d, err := clnt.Stat(fid)
	clnt.Clunk(fid)
	return d, err
}

// Modifies the data of the file associated with the Fid, or an Error.
func (clnt *Clnt) Wstat(fid *Fid, dir *ninep.Dir) error {
	tc := clnt.NewFcall()
	err := ninep.PackTwstat(tc, fid.Fid, dir, clnt.Dotu)
	if err != nil {
		return err
	}

	_, err = clnt.Rpc(tc)
	return err
}

// FSync syncs the file for a fid. It does this by sending a NewWstatDir, i.e. a
// Dir with all fields set to 'not set'.
func (clnt *Clnt) FSync(fid *Fid) error {
	return clnt.Wstat(fid, ninep.NewWstatDir())
}

// Rename renames the file for a fid.
func (clnt *Clnt) Rename(fid *Fid, name string) error {
	d := ninep.NewWstatDir()
	d.Name = name
	return clnt.Wstat(fid, d)
}
