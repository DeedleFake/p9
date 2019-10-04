package proto

import (
	"io"
)

// limitedReader is a reimplementation of io.limitedReader with two
// main differences:
//
// * N is a uint32, allowing for larger sizes on 32-bit systems.
// * A custom error can be returned if N becomes zero.
type limitedReader struct {
	R io.Reader
	N uint32
	E error
}

func (lr limitedReader) err() error {
	if lr.E == nil {
		return io.EOF
	}

	return lr.E
}

func (lr *limitedReader) Read(buf []byte) (int, error) {
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
