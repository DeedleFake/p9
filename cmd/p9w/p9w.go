package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/DeedleFake/p9"
)

type ctxKey string

const (
	AddrKey   ctxKey = "addr"
	ClientKey ctxKey = "client"
	AttachKey ctxKey = "attach"
)

func Error(rw http.ResponseWriter, err error, status int) {
	log.Printf("Error (%v): %v", status, err)

	rw.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(rw)

	rw.WriteHeader(status)

	e.Encode(struct {
		Err error
	}{
		Err: err,
	})
}

func AttachHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()

		addr := q.Get("addr")
		if addr == "" {
			Error(rw, errors.New("addr not specified"), http.StatusBadRequest)
			return
		}
		if parts := strings.SplitN(addr, ":", 2); len(parts) < 2 {
			addr += ":564"
		}

		c, err := p9.Dial("tcp", addr)
		if err != nil {
			Error(rw, err, http.StatusBadRequest)
			return
		}
		defer c.Close()

		a, err := c.Attach(nil, q.Get("user"), q.Get("aname"))
		if err != nil {
			Error(rw, err, http.StatusBadRequest)
			return
		}
		defer a.Close()

		ctx := req.Context()
		ctx = context.WithValue(ctx, AddrKey, addr)
		ctx = context.WithValue(ctx, ClientKey, c)
		ctx = context.WithValue(ctx, AttachKey, a)
		h.ServeHTTP(rw, req.WithContext(ctx))
	})
}

func DispositionHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer h.ServeHTTP(rw, req)

		q := req.URL.Query()
		if q.Get("path") == "" {
			if q.Get("addr") == "" {
				return
			}

			rw.Header().Set(
				"Content-Disposition",
				fmt.Sprintf("filename=%q", q.Get("addr")),
			)
		}

		rw.Header().Set(
			"Content-Disposition",
			fmt.Sprintf("filename=%q", filepath.Base(q.Get("path"))),
		)
	})
}

func handleLS(rw http.ResponseWriter, req *http.Request) {
	addr := req.Context().Value(AddrKey).(string)
	a := req.Context().Value(AttachKey).(*p9.Remote)

	rw.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(rw)

	q := req.URL.Query()
	log.Printf("ls %v %v", addr, q.Get("path"))

	fi, err := a.Stat(q.Get("path"))
	if err != nil {
		Error(rw, err, http.StatusBadRequest)
		return
	}

	if fi.Mode&p9.ModeDir == 0 {
		e.Encode(fi)
		return
	}

	f, err := a.Open(q.Get("path"), p9.OREAD)
	if err != nil {
		Error(rw, err, http.StatusBadRequest)
		return
	}
	defer f.Close()

	entries, err := f.Readdir()
	if err != nil {
		Error(rw, err, http.StatusInternalServerError)
		return
	}

	e.Encode(entries)
}

func handleRead(rw http.ResponseWriter, req *http.Request) {
	addr := req.Context().Value(AddrKey).(string)
	a := req.Context().Value(AttachKey).(*p9.Remote)

	q := req.URL.Query()
	log.Printf("read %v %v", addr, q.Get("path"))

	f, err := a.Open(q.Get("path"), p9.OREAD)
	if err != nil {
		Error(rw, err, http.StatusBadRequest)
		return
	}
	defer f.Close()

	_, err = io.Copy(rw, f)
	if err != nil {
		Error(rw, err, http.StatusInternalServerError)
		return
	}
}

func handleMain(rw http.ResponseWriter, req *http.Request) {
	io.WriteString(rw, `<html>
	<body>
		<h3>Global Parameters</h3>
		<ul>
			<li>addr</li>
			<li>user</li>
			<li>aname</li>
		</ul>

		<h3>Endpoints</h3>
		<dl>
			<dt><a href='/ls'>ls</a></dt>
			<dd>List files. Parameters: path</dd>

			<dt><a href='/read'>read</a></dt>
			<dd>Read a file. Parameters: path</dd>
		</dl>
	</body>
</html>`)
}

func main() {
	addr := flag.String("addr", ":8080", "Address to listen on.")
	flag.Parse()

	handlers := func(h http.Handler) http.Handler {
		return DispositionHandler(AttachHandler(h))
	}

	http.Handle("/ls", handlers(http.HandlerFunc(handleLS)))
	http.Handle("/read", handlers(http.HandlerFunc(handleRead)))
	http.HandleFunc("/", handleMain)

	log.Printf("Starting server at %q", *addr)
	log.Fatalln(http.ListenAndServe(*addr, nil))
}
