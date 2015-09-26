// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clnt

import "github.com/lionkov/ninep"

// Removes the file associated with the Fid. Returns nil if the
// operation is successful.
func (clnt *Clnt) Remove(fid *Fid) error {
	tc := clnt.NewFcall()
	err := ninep.PackTremove(tc, fid.Fid)
	if err != nil {
		return err
	}

	_, err = clnt.Rpc(tc)
	clnt.fidpool.putId(fid.Fid)
	fid.Fid = ninep.NOFID

	return err
}

// Removes the named file. Returns nil if the operation is successful.
func (clnt *Clnt) FRemove(path string) error {
	var err error
	fid, err := clnt.FWalk(path)
	if err != nil {
		return err
	}

	err = clnt.Remove(fid)
	return err
}
