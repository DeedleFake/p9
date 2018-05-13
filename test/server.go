package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"sort"
	"sync"

	"github.com/DeedleFake/p9"
)

type FS map[string]p9.File

func (fs FS) Type(p string) (p9.QIDType, bool) {
	file, ok := fs[p]
	if !ok {
		return 0, false
	}

	switch file.(type) {
	case *File:
		return p9.QTFile, true
	case Dir:
		return p9.QTDir, true

	default:
		panic(fmt.Errorf("Unexpected type: %T", file))
	}
}

func (fs FS) Stat(p string) (p9.DirEntry, error) {
	dir, name := path.Split(p)

	d, ok := fs[dir].(Dir)
	if !ok {
		return p9.DirEntry{}, errors.New("No such directory")
	}

	f, ok := d[name]
	if !ok {
		return p9.DirEntry{}, errors.New("No such file")
	}

	return f, nil
}

func (fs FS) Open(p string, mode uint8) (p9.File, error) {
	file, ok := fs[p]
	if !ok {
		return nil, os.ErrNotExist
	}
	return file, nil
}

type File struct {
	m sync.RWMutex

	t    p9.QIDType
	Data []byte
}

func (file *File) ReadAt(buf []byte, off int64) (int, error) {
	file.m.RLock()
	defer file.m.RUnlock()

	if off >= int64(len(file.Data)) {
		return 0, io.EOF
	}

	n := copy(buf, file.Data[off:])
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

func (file *File) WriteAt(buf []byte, off int64) (int, error) {
	file.m.Lock()
	defer file.m.Unlock()

	file.Data = append(file.Data[:off], append(buf[:len(buf):len(buf)], file.Data[int(off)+len(buf):]...)...)
	return len(buf), nil
}

func (file File) Close() error {
	return nil
}

func (file File) Type() p9.QIDType {
	return file.t
}

func (file File) Readdir() ([]p9.DirEntry, error) {
	return nil, errors.New("Not a directory")
}

type Dir map[string]p9.DirEntry

func (d Dir) ReadAt(buf []byte, off int64) (int, error) {
	panic("Not implemented.")
}

func (d Dir) WriteAt(buf []byte, off int64) (int, error) {
	return 0, errors.New("can't write to directory")
}

func (d Dir) Close() error {
	return nil
}

func (d Dir) Type() p9.QIDType {
	return p9.QTDir
}

func (d Dir) Readdir() ([]p9.DirEntry, error) {
	entries := make([]p9.DirEntry, 0, len(d))
	for _, entry := range d {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i1, i2 int) bool {
		return entries[i1].Name < entries[i2].Name
	})

	return entries, nil
}

var (
	fs = FS{
		"/": Dir{
			"test": p9.DirEntry{
				Type: p9.QTFile,
				Name: "test",
			},
		},

		"/test": &File{
			Data: []byte("This is a test."),
		},
	}
)

func connHandler() p9.MessageHandler {
	return p9.HandleFS(fs, 1024)
}

func main() {
	lis, err := net.Listen("tcp", "localhost:5640")
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer lis.Close()

	err = p9.Serve(lis, p9.ConnHandlerFunc(connHandler))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
