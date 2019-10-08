package p9

import (
	"errors"
	"io"

	"github.com/DeedleFake/p9/proto"
)

// ReadDir decodes a series of directory entries from a reader. It
// reads until EOF, so it doesn't return io.EOF as a possible error.
//
// It is recommended that the reader passed to ReadDir have some form
// of buffering, as some servers will silently mishandle attempts to
// read pieces of a directory. Wrapping the reader with a bufio.Reader
// is often sufficient.
func ReadDir(r io.Reader) ([]DirEntry, error) {
	var entries []DirEntry
	for {
		var stat Stat
		err := proto.Read(r, &stat)
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return entries, err
		}

		entries = append(entries, stat.dirEntry())
	}
}

// WriteDir writes a series of directory entries to w. It uses getPath
// to lookup the QID path of each entry by name. If getPath returns an
// error, that error is immediately returned.
func WriteDir(w io.Writer, entries []DirEntry, getPath func(string) (uint64, error)) error {
	for _, entry := range entries {
		p, err := getPath(entry.Name)
		if err != nil {
			return err
		}

		err = proto.Write(w, entry.stat(p))
		if err != nil {
			return err
		}
	}

	return nil
}
