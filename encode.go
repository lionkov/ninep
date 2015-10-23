// Copyright 2015 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ninep

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// MsgEncode writes a 9p message to a buffer form passed in arguments.
// It prepends the required 4 bytes of length and fills it in at the end.
// It gets ugly here in places but show me a Marshal'ing package that doesn't have ugliness in places.
func msgEncode(v ...interface{}) (*bytes.Buffer, error) {
	var adjust int
	var b = &bytes.Buffer{}
	b.Write([]byte{0, 0, 0, 0})
	for i := range v {
		switch s := v[i].(type) {
		default:
			if err := binary.Write(b, binary.LittleEndian, s); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case *Dir:
			// avoid reflection. Oh and, well, it doesn't work anyway in this case.
			// The binary package can't write a Dir.
			var d *bytes.Buffer
			var l uint16
			var err error
			if d, err = msgEncode([]interface{}{
				//s.Size, Dir includes size on wire but really should not IMHO. FIX ME
				s.Type,
				s.Dev,
				s.Qid.Type, s.Qid.Version, s.Qid.Path,
				s.Mode,
				s.Atime,
				s.Mtime,
				s.Length,
				s.Name,
				s.Uid,
				s.Gid,
				s.Muid}...); err != nil {
				return nil, err
			}
			// The msgEncode pushes 32 bits of length for us, which we use as two 16-bit fields.
			// Note that the embedded length does not include itself.
			l = uint16(len(d.Bytes())) - 2
			fmt.Printf("dir: l is %v\n", l)
			d.Bytes()[0] = uint8(l)
			d.Bytes()[1] = uint8(l >> 8)
			d.Bytes()[2] = uint8(l - 2)
			d.Bytes()[3] = uint8((l - 2) >> 8)
			b.Write(d.Bytes())

		case *DirDotu:
			// avoid reflection. Oh and, well, it doesn't work anyway in this case.
			// The binary package can't write a Dir.
			var d *bytes.Buffer
			var l uint16
			var err error
			if d, err = msgEncode([]interface{}{
				//s.Size, Dir includes size on wire but really should not IMHO. FIX ME
				s.Type,
				s.Dev,
				s.Qid.Type, s.Qid.Version, s.Qid.Path,
				s.Mode,
				s.Atime,
				s.Mtime,
				s.Length,
				s.Name,
				s.Uid,
				s.Gid,
				s.Muid,
				s.Ext,
				s.Uidnum,
				s.Gidnum,
				s.Muidnum}...); err != nil {
				return nil, err
			}
			// The msgEncode pushes 32 bits of length for us, which we use as two 16-bit fields.
			// Note that the embedded length does not include itself.
			l = uint16(len(d.Bytes())) - 2
			fmt.Printf("dir: l is %v\n", l)
			d.Bytes()[0] = uint8(l)
			d.Bytes()[1] = uint8(l >> 8)
			d.Bytes()[2] = uint8(l - 2)
			d.Bytes()[3] = uint8((l - 2) >> 8)
			b.Write(d.Bytes())

		case MType:
			if err := binary.Write(b, binary.LittleEndian, uint8(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case Mode:
			if err := binary.Write(b, binary.LittleEndian, uint8(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case Tag:
			if err := binary.Write(b, binary.LittleEndian, uint16(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case NumEntries:
			if err := binary.Write(b, binary.LittleEndian, uint16(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case FID:
			if err := binary.Write(b, binary.LittleEndian, uint32(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case Count:
			if err := binary.Write(b, binary.LittleEndian, uint32(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case Perm:
			if err := binary.Write(b, binary.LittleEndian, uint32(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case Offset:
			if err := binary.Write(b, binary.LittleEndian, uint64(s)); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
		case string:
			if err := binary.Write(b, binary.LittleEndian, uint16(len(s))); err != nil {
				return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
			}
			b.WriteString(s)
		case []string:
			for _, v := range s {
				if err := binary.Write(b, binary.LittleEndian, uint16(len(v))); err != nil {
					return nil, fmt.Errorf("failed to encode %v: %v\n", v, err)
				}
				b.WriteString(v)
			}
		case WriteData:
			adjust = len(s)
		}

	}
	l := b.Len() + adjust
	b.Bytes()[0] = uint8(l)
	b.Bytes()[1] = uint8(l >> 8)
	b.Bytes()[2] = uint8(l >> 16)
	b.Bytes()[3] = uint8(l >> 24)
	return b, nil
}
