package main

import (
	"net"
	"os"
	"path/filepath"

	"github.com/DeedleFake/p9"
)

func infoToEntry(fi os.FileInfo) p9.DirEntry {
	t := p9.QTFile
	if fi.IsDir() {
		t = p9.QTDir
	}

	return p9.DirEntry{
		Type:   t,
		Mode:   uint32(fi.Mode()),
		MTime:  fi.ModTime(),
		Name:   fi.Name(),
		Length: uint64(fi.Size()),
	}
}

type Dir string

func (d Dir) path(p string) string {
	return filepath.Join(string(d), filepath.FromSlash(p))
}

func (d Dir) Stat(p string) (p9.DirEntry, error) {
	fi, err := os.Stat(d.path(p))
	if err != nil {
		return p9.DirEntry{}, err
	}

	return infoToEntry(fi), nil
}

func (d Dir) WriteStat(p string, changes map[string]interface{}) error {
	panic("Not implemented.")
}

func (d Dir) Auth(user, aname string) (p9.File, error) {
	panic("Not implemented.")
}

func (d Dir) Open(p string, mode uint8) (p9.File, error) {
	var flag int
	switch mode {
	case p9.OREAD:
		flag = os.O_RDONLY
	case p9.OWRITE:
		flag = os.O_WRONLY
	case p9.ORDWR:
		flag = os.O_RDWR
	}
	if mode&p9.OTRUNC != 0 {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(d.path(p), flag, 0644)
	return &File{
		File: file,
	}, err
}

func (d Dir) Create(p string, perm uint32, mode uint8) (p9.File, error) {
	panic("Not implemented.")
}

func (d Dir) Remove(p string) error {
	panic("Not implemented.")
}

type File struct {
	*os.File
}

func (f *File) Stat() p9.DirEntry {
	fi, err := f.File.Stat()
	if err != nil {
		panic(err)
	}
	return infoToEntry(fi)
}

func (f *File) Readdir() ([]p9.DirEntry, error) {
	fi, err := f.File.Readdir(-1)
	if err != nil {
		return nil, err
	}

	entries := make([]p9.DirEntry, 0, len(fi))
	for _, info := range fi {
		entries = append(entries, infoToEntry(info))
	}
	return entries, nil
}

func main() {
	lis, err := net.Listen("unix", "test.sock")
	if err != nil {
		panic(err)
	}
	defer lis.Close()

	err = p9.Serve(lis, p9.ConnHandlerFunc(func() p9.MessageHandler {
		return p9.HandleFS(Dir("."), 2048)
	}))
	if err != nil {
		panic(err)
	}
}
