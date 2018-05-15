package p9

import (
	"errors"
	"os"
	"path/filepath"
	"time"
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
	p = d.path(p)
	base := filepath.Dir(p)

	mode, ok := changes["Mode"]
	if ok {
		perm := os.FileMode(mode.(uint32)).Perm()
		err := os.Chmod(p, perm)
		if err != nil {
			return err
		}
	}

	atime, ok1 := changes["ATime"]
	mtime, ok2 := changes["MTime"]
	if ok1 || ok2 {
		atime, _ := atime.(time.Time)
		mtime, _ := mtime.(time.Time)
		err := os.Chtimes(p, atime, mtime)
		if err != nil {
			return err
		}
	}

	length, ok := changes["Length"]
	if ok {
		err := os.Truncate(p, int64(length.(uint64)))
		if err != nil {
			return err
		}
	}

	name, ok := changes["Name"]
	if ok {
		err := os.Rename(p, filepath.Join(base, filepath.FromSlash(name.(string))))
		if err != nil {
			return err
		}
	}

	return nil
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
