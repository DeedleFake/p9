package main

import (
	"errors"
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

func (fs FS) WriteStat(p string, changes map[string]interface{}) error {
	return errors.New("wstat is not implemented")
}

func (fs FS) Auth(user, aname string) (p9.File, error) {
	return &File{
		stat: p9.DirEntry{
			Type: p9.QTAuth,
			Name: aname,
		},
	}, nil
}

func (fs FS) Open(p string, mode uint8) (p9.File, error) {
	file, ok := fs[p]
	if !ok {
		return nil, os.ErrNotExist
	}
	return file, nil
}

func (fs FS) Create(p string, perm uint32, mode uint8) (p9.File, error) {
	dir, name := path.Split(p)

	entry := p9.DirEntry{
		Type: p9.QTFile,
		Name: name,
	}

	fs[dir].(Dir)[name] = entry

	fs[name] = &File{
		stat: entry,
	}

	return fs[name], nil
}

func (fs FS) Remove(p string) error {
	delete(fs, p)

	dir, name := path.Split(p)
	delete(fs[dir].(Dir), name)

	return nil
}

type File struct {
	m sync.RWMutex

	stat p9.DirEntry
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

	if off > int64(len(file.Data)) {
		return 0, io.EOF
	}

	ss := int(off) + len(buf)
	if ss > len(file.Data) {
		ss = len(file.Data)
	}

	file.Data = append(file.Data[:off], append(buf[:len(buf):len(buf)], file.Data[ss:]...)...)
	return len(buf), nil
}

func (file File) Close() error {
	return nil
}

func (file File) Stat() p9.DirEntry {
	return file.stat
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

func (d Dir) Stat() p9.DirEntry {
	return d[""]
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
			"": p9.DirEntry{
				Type: p9.QTDir,
				Name: ".",
			},

			"test": p9.DirEntry{
				Type: p9.QTFile,
				Name: "test",
			},
		},

		"/test": &File{
			stat: p9.DirEntry{
				Type: p9.QTFile,
				Name: "test",
			},
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
