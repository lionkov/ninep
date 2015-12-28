// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drivefs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/srv"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const (
	Qroot = 'r'
)

type DriveFS struct {
	srv.Srv
	Listener net.Listener
	Secret   string
	Cache    string
	Context  context.Context
	Config   *oauth2.Config
	Drive    *drive.Service
	Token    oauth2.Token
}

type config func(*DriveFS) error

type DriveFile struct {
	Name string
	*ninep.Qid
}

type Fid struct {
	DriveFile
}

var (
	dirQids = map[string]*ninep.Qid{
		".": &ninep.Qid{Type: ninep.QTDIR, Version: 0777, Path: Qroot},
	}
	// Verify that we correctly implement ReqOps
	_ = srv.ReqOps(&DriveFS{})
)

func (f *DriveFS) Read(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (*DriveFS) Clunk(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (f *DriveFS) Write(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (f *DriveFS) Walk(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (f *DriveFS) Create(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (f *DriveFS) Open(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (f *DriveFS) Remove(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (f *DriveFS) Stat(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

func (f *DriveFS) Wstat(r *srv.Req) {
	r.RespondError(&ninep.Error{Err: "NOT YET", Errornum: ninep.EINVAL})
}

// Attach responds to an attach message from the 9p client. It results in create a connection
// to drive.
func (fs *DriveFS) Attach(r *srv.Req) {
	if r.Afid != nil {
		r.RespondError(srv.Enoauth)
		return
	}

	ctx := context.Background()

	// We thought about putting the secret and cache info in the Aname. We might yet do it.
	// Tough call, but realistically, the user is starting the server, so we're going to let them
	// set up the server secret and cache when they start the server. This may change.
	// We don't print the Secret or the Cache when this fails for what I think are obvious reasons.
	config, err := google.ConfigFromJSON([]byte(fs.Secret), drive.DriveMetadataReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	if err := json.NewDecoder(bytes.NewReader([]byte(fs.Cache))).Decode(&fs.Token); err != nil {
		log.Fatalf("Unable to get a token from the string: fs.Cache %v", err)
	}
	client := config.Client(ctx, &fs.Token)

	drive, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}

	r.Fid.Aux = &Fid{DriveFile: DriveFile{Name: "."}}

	fs.Config, fs.Context, fs.Drive = config, ctx, drive

	r.RespondRattach(dirQids["."])
}

func NewDriveFS(c ...config) (*DriveFS, error) {
	f := new(DriveFS)

	if !f.Start(f) {
		return nil, fmt.Errorf("Can't happen: Starting the server failed")
	}

	for i := range c {
		if err := c[i](f); err != nil {
			return nil, err
		}
	}
	return f, nil
}
