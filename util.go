package p9

import (
	"io"
	"os"
)

// LimitedReader is a reimplementation of io.LimitedReader with two
// main differences:
//
// * N is a uint32, allowing for larger sizes on 32-bit systems.
// * A custom error can be returned if N becomes zero.
type LimitedReader struct {
	R io.Reader
	N uint32
	E error
}

func (lr LimitedReader) err() error {
	if lr.E == nil {
		return io.EOF
	}

	return lr.E
}

func (lr *LimitedReader) Read(buf []byte) (int, error) {
	if lr.N <= 0 {
		return 0, lr.err()
	}

	if uint32(len(buf)) > lr.N {
		buf = buf[:lr.N]
	}

	n, err := lr.R.Read(buf)
	lr.N -= uint32(n)
	return n, err
}

func infoToEntry(fi os.FileInfo) DirEntry {
	return DirEntry{
		Mode:   ModeFromOS(fi.Mode()),
		MTime:  fi.ModTime(),
		Name:   fi.Name(),
		Length: uint64(fi.Size()),
	}
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
