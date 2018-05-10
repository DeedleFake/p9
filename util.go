package p9

import (
	"io"
)

// LimitedReader is a reimplementation of io.LimitedReader with two
// main differences:
//
// * N is a uint32, allowing for larger sizes on 32-bit systems.
// * A custom error can be returned if N becomes less than zero.
type LimitedReader struct {
	R io.Reader
	N uint32

	// The error to return if the limit is crossed. If this is nil,
	// io.EOF is returned.
	Over error
}

func (lr LimitedReader) err() error {
	if lr.Over == nil {
		return io.EOF
	}

	return lr.Over
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
	if err != nil {
		return n, err
	}
	if lr.N < 0 {
		return n, lr.err()
	}
	return n, nil
}
