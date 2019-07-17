package p9

import (
	"errors"
	"os"
	"path/filepath"
)

// Dir is an implementation of FileSystem that serves from the local
// filesystem. It accepts attachments of either "" or "/", but rejects
// all others.
//
// Note that Dir does not support authentication, simply returning an
// error for any attempt to do so. If authentication is necessary,
// wrap a Dir in an AuthFS instance.
type Dir string

func (d Dir) path(p string) string {
	return filepath.Join(string(d), filepath.FromSlash(p))
}

func (d Dir) Stat(p string) (DirEntry, error) { // nolint
	fi, err := os.Stat(d.path(p))
	if err != nil {
		return DirEntry{}, err
	}

	return infoToEntry(fi), nil
}

func (d Dir) WriteStat(p string, changes StatChanges) error { // nolint
	// TODO: Add support for other values.

	p = d.path(p)
	base := filepath.Dir(p)

	mode, ok := changes.Mode()
	if ok {
		perm := mode.Perm()
		err := os.Chmod(p, os.FileMode(perm))
		if err != nil {
			return err
		}
	}

	atime, ok1 := changes.ATime()
	mtime, ok2 := changes.MTime()
	if ok1 || ok2 {
		err := os.Chtimes(p, atime, mtime)
		if err != nil {
			return err
		}
	}

	length, ok := changes.Length()
	if ok {
		err := os.Truncate(p, int64(length))
		if err != nil {
			return err
		}
	}

	name, ok := changes.Name()
	if ok {
		err := os.Rename(p, filepath.Join(base, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
	}

	return nil
}

func (d Dir) Auth(user, aname string) (File, error) { // nolint
	return nil, errors.New("auth not supported")
}

func (d Dir) Attach(afile File, user, aname string) (Attachment, error) { // nolint
	switch aname {
	case "", "/":
		return d, nil
	}

	return nil, errors.New("unknown attachment")
}

func (d Dir) Open(p string, mode uint8) (File, error) { // nolint
	flag := toOSFlags(mode)

	file, err := os.OpenFile(d.path(p), flag, 0644)
	return &dirFile{
		File: file,
	}, err
}

func (d Dir) Create(p string, perm FileMode, mode uint8) (File, error) { // nolint
	p = d.path(p)

	if perm&ModeDir != 0 {
		err := os.Mkdir(p, os.FileMode(perm.Perm()))
		if err != nil {
			return nil, err
		}
	}

	flag := toOSFlags(mode)

	file, err := os.OpenFile(p, flag|os.O_CREATE, os.FileMode(perm.Perm()))
	return &dirFile{
		File: file,
	}, err
}

func (d Dir) Remove(p string) error { // nolint
	return os.Remove(d.path(p))
}

type dirFile struct {
	*os.File
}

func (f *dirFile) Readdir() ([]DirEntry, error) { // nolint
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

// AuthFS allows simple wrapping and overwriting of the Auth() and
// Attach() methods of an existing FileSystem implementation, allowing
// the user to add authentication support to a FileSystem that does
// not have it, or to change the implementation of that support for
// FileSystems that do.
type AuthFS struct {
	FileSystem

	// AuthFunc is the function called when the Auth() method is called.
	AuthFunc func(user, aname string) (File, error)

	// AttachFunc is the function called when the Attach() method is
	// called. Note that this function, unlike the method, does not
	// return an Attachment. Instead, if this function returns a nil
	// error, the underlying implementation's Attach() method is called
	// with the returned file as its afile argument.
	AttachFunc func(afile File, user, aname string) (File, error)
}

func (a AuthFS) Auth(user, aname string) (File, error) { // nolint
	return a.AuthFunc(user, aname)
}

func (a AuthFS) Attach(afile File, user, aname string) (Attachment, error) { // nolint
	file, err := a.AttachFunc(afile, user, aname)
	if err != nil {
		return nil, err
	}
	return a.FileSystem.Attach(file, user, aname)
}

func toOSFlags(mode uint8) (flag int) {
	if mode&OREAD != 0 {
		flag |= os.O_RDONLY
	}
	if mode&OWRITE != 0 {
		flag |= os.O_WRONLY
	}
	if mode&ORDWR != 0 {
		flag |= os.O_RDWR
	}
	if mode&OTRUNC != 0 {
		flag |= os.O_TRUNC
	}

	return flag
}
