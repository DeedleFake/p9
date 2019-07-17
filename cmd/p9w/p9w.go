package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/DeedleFake/p9"
)

type ctxKey string

const (
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

		c, err := p9.Dial("tcp", q.Get("addr"))
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
		ctx = context.WithValue(ctx, AttachKey, a)
		ctx = context.WithValue(ctx, ClientKey, c)
		h.ServeHTTP(rw, req.WithContext(ctx))
	})
}

func handleLS(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(rw)

	q := req.URL.Query()
	log.Printf("ls %v %v", q.Get("addr"), q.Get("path"))

	a := req.Context().Value(AttachKey).(*p9.Remote)

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
	q := req.URL.Query()
	log.Printf("read %v %v", q.Get("addr"), q.Get("path"))

	a := req.Context().Value(AttachKey).(*p9.Remote)

	f, err := a.Open(q.Get("path"), p9.OREAD)
	if err != nil {
		Error(rw, err, http.StatusBadRequest)
		return
	}
	defer f.Close()

	rw.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("filename=%q", filepath.Base(q.Get("path"))),
	)
	_, err = io.Copy(rw, f)
	if err != nil {
		Error(rw, err, http.StatusInternalServerError)
		return
	}
}

func main() {
	addr := flag.String("addr", ":8080", "Address to listen on.")
	flag.Parse()

	http.Handle("/ls", AttachHandler(http.HandlerFunc(handleLS)))
	http.Handle("/read", AttachHandler(http.HandlerFunc(handleRead)))

	log.Println("Starting server at %v", *addr)
	log.Fatalln(http.ListenAndServe(*addr, nil))
}
