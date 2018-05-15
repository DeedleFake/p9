package p9

import (
	"errors"
	"os"
	"path/filepath"
)

type Dir string

func (d Dir) path(p string) string {
	return filepath.Join(string(d), filepath.FromSlash(p))
}

func (d Dir) Stat(p string) (DirEntry, error) {
	fi, err := os.Stat(d.path(p))
	if err != nil {
		return DirEntry{}, err
	}

	return infoToEntry(fi), nil
}

func (d Dir) WriteStat(p string, changes map[string]interface{}) error {
	panic("Not implemented.")
}

func (d Dir) Auth(user, aname string) (File, error) {
	return nil, errors.New("auth not supported")
}

func (d Dir) Open(p string, mode uint8) (File, error) {
	flag := toOSFlags(mode)

	file, err := os.OpenFile(d.path(p), flag, 0644)
	return &dirFile{
		File: file,
	}, err
}

func (d Dir) Create(p string, perm uint32, mode uint8) (File, error) {
	if perm&DMDIR != 0 {
		panic("Not implemented.")
	}

	flag := toOSFlags(mode)

	file, err := os.OpenFile(d.path(p), flag, os.FileMode(perm).Perm())
	return &dirFile{
		File: file,
	}, err
}

func (d Dir) Remove(p string) error {
	return os.Remove(d.path(p))
}

type dirFile struct {
	*os.File
}

func (f *dirFile) Stat() (DirEntry, error) {
	fi, err := f.File.Stat()
	if err != nil {
		return DirEntry{}, err
	}
	return infoToEntry(fi), nil
}

func (f *dirFile) Readdir() ([]DirEntry, error) {
	fi, err := f.File.Readdir(-1)
	if err != nil {
		return nil, err
	}

	entries := make([]DirEntry, 0, len(fi))
	for _, info := range fi {
		entries = append(entries, infoToEntry(info))
	}
	return entries, nil
}
