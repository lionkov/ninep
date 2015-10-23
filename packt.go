// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ninep

import "bytes"

// Create a Tversion message in the specified Fcall.
func PackTversion(fc *Fcall, msize uint32, version string) error {
	b, err := msgEncode(Tversion, NOTAG, Count(msize), version)
	if err != nil {
		return err
	}

	// I'm leaving this annoying little line in each function because I think I can
	// kill it later.
	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tversion, NOTAG
	return nil
}

// Create a Tauth message in the specified Fcall.
// TODO: change fid type to FID
func PackTauth(fc *Fcall, fid uint32, uname string, aname string, unamenum uint32, dotu bool) error {
	var b *bytes.Buffer
	var err error
	if !dotu {
		b, err = msgEncode(Tauth, NOTAG, FID(fid), uname, aname)
	} else {
		b, err = msgEncode(Tauth, NOTAG, FID(fid), uname, aname, unamenum)
	}
	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tauth, NOTAG
	fc.Fid, fc.Uname, fc.Aname = fid, uname, aname
	return nil
}

// Create a Tflush message in the specified Fcall.
func PackTflush(fc *Fcall, oldtag uint16) error {
	b, err := msgEncode(Tflush, NOTAG, Tag(oldtag))
	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tflush, NOTAG
	fc.Oldtag = oldtag
	return nil
}

// Create a Tattach message in the specified Fcall. If dotu is true,
// the function will create 9P2000.u including the nuname value, otherwise
// nuname is ignored.
// NOTE: dotu is going away. We hope.
func PackTattach(fc *Fcall, fid uint32, afid uint32, uname string, aname string, unamenum uint32, dotu bool) error {
	var b *bytes.Buffer
	var err error

	if !dotu {
		b, err = msgEncode(Tattach, NOTAG, FID(fid), FID(afid), uname, aname)
	} else {
		b, err = msgEncode(Tattach, NOTAG, FID(fid), FID(afid), uname, aname, unamenum)
	}
	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tattach, NOTAG
	fc.Fid, fc.Afid, fc.Uname, fc.Aname, fc.Unamenum = fid, afid, uname, aname, unamenum
	return nil
}

// Create a Twalk message in the specified Fcall.
func PackTwalk(fc *Fcall, fid uint32, newfid uint32, wnames []string) error {
	b, err := msgEncode(Twalk, NOTAG, FID(fid), FID(newfid), NumEntries(len(wnames)), wnames)

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Twalk, NOTAG
	fc.Fid, fc.Newfid = fid, newfid
	return nil
}

// Create a Topen message in the specified Fcall.
func PackTopen(fc *Fcall, fid uint32, mode uint8) error {
	b, err := msgEncode(Topen, NOTAG, FID(fid), Mode(mode))

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Topen, NOTAG
	fc.Fid, fc.Mode = fid, mode
	return nil
}

// Create a Tcreate message in the specified Fcall. If dotu is true,
// the function will create a 9P2000.u message that includes ext.
// Otherwise the ext value is ignored.
func PackTcreate(fc *Fcall, fid uint32, name string, perm uint32, mode uint8, ext string, dotu bool) error {
	var err error
	var b *bytes.Buffer
	if ! dotu {
		b, err = msgEncode(Tcreate, NOTAG, FID(fid), name, Perm(perm), Mode(mode))
	} else {
		b, err = msgEncode(Tcreate, NOTAG, FID(fid), name, Perm(perm), Mode(mode), ext)
	}

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tcreate, NOTAG
	fc.Fid, fc.Name, fc.Perm, fc.Mode = fid, name, perm, mode

	return nil
}

// Create a Tread message in the specified Fcall.
func PackTread(fc *Fcall, fid uint32, offset uint64, count uint32) error {
	b, err := msgEncode(Tread, NOTAG, FID(fid), Offset(offset), Count(count))

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tread, NOTAG
	fc.Fid, fc.Offset, fc.Count = fid, offset, count
	return nil
}

// Create a Twrite message in the specified Fcall.
func PackTwrite(fc *Fcall, fid uint32, offset uint64, count uint32, data []byte) error {
	b, err := msgEncode(Twrite, NOTAG, FID(fid), Offset(offset), Count(count), WriteData(data))

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Twrite, NOTAG
	fc.Fid, fc.Offset, fc.Count, fc.Data = fid, offset, count, data
	return nil
}

// Create a Tclunk message in the specified Fcall.
func PackTclunk(fc *Fcall, fid uint32) error {
	b, err := msgEncode(Tclunk, NOTAG, FID(fid))

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tclunk, NOTAG
	fc.Fid = fid
	return nil
}

// Create a Tremove message in the specified Fcall.
func PackTremove(fc *Fcall, fid uint32) error {
	b, err := msgEncode(Tremove, NOTAG, FID(fid))

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tremove, NOTAG
	fc.Fid = fid
	return nil
}

// Create a Tstat message in the specified Fcall.
func PackTstat(fc *Fcall, fid uint32) error {
	b, err := msgEncode(Tstat, NOTAG, FID(fid))

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Tstat, NOTAG
	fc.Fid = fid
	return nil
}

// Create a Twstat message in the specified Fcall. If dotu is true
// the function will create 9P2000.u message, otherwise the 9P2000.u
// specific fields from the Stat value will be ignored.
func PackTwstat(fc *Fcall, fid uint32, d *Dir, dotu bool) error {
	var b *bytes.Buffer
	var err error
	if !dotu {
		b, err = msgEncode(Twstat, NOTAG, FID(fid), d)
	} else {
		var du = DirDotu(*d)
		b, err = msgEncode(Twstat, NOTAG, FID(fid), &du)
	}

	if err != nil {
		return err
	}

	fc.Pkt, fc.Size, fc.Type, fc.Tag = b.Bytes(), uint32(b.Len()), Twstat, NOTAG
	fc.Fid, fc.Dir = fid, *d

	return nil
}
