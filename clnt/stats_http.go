// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build httpstats

package clnt

import (
	"fmt"
	"io"
	"net/http"

	"github.com/lionkov/ninep"
)

func (clnt *Clnt) ServeHTTP(c http.ResponseWriter, r *http.Request) {
	io.WriteString(c, fmt.Sprintf("<html><body><h1>Client %s</h1>", clnt.Id))
	defer io.WriteString(c, "</body></html>")

	// fcalls
	if clnt.Debuglevel&DbgLogFcalls != 0 {
		fs := clnt.Log.Filter(clnt, DbgLogFcalls)
		io.WriteString(c, fmt.Sprintf("<h2>Last %d 9P messages</h2>", len(fs)))
		for _, l := range fs {
			fc := l.Data.(*ninep.Fcall)
			if fc.Type != 0 {
				io.WriteString(c, fmt.Sprintf("<br>%s", fc))
			}
		}
	}
}

func clntServeHTTP(c http.ResponseWriter, r *http.Request) {
	io.WriteString(c, fmt.Sprintf("<html><body>"))
	defer io.WriteString(c, "</body></html>")

	clnts.Lock()
	if clnts.clntList == nil {
		io.WriteString(c, "no clients")
	}

	for clnt := clnts.clntList; clnt != nil; clnt = clnt.next {
		io.WriteString(c, fmt.Sprintf("<a href='/ninep/clnt/%s'>%s</a><br>", clnt.Id, clnt.Id))
	}
	clnts.Unlock()
}

func (clnt *Clnt) statsRegister() {
	http.Handle("/ninep/clnt/"+clnt.Id, clnt)
}

func (clnt *Clnt) statsUnregister() {
	http.Handle("/ninep/clnt/"+clnt.Id, nil)
}

func (c *ClntList) statsRegister() {
	http.HandleFunc("/ninep/clnt", clntServeHTTP)
}

func (c *ClntList) statsUnregister() {
	http.HandleFunc("/ninep/clnt", nil)
}
