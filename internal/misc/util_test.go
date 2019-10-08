package misc_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/DeedleFake/p9/internal/misc"
)

func TestLimitedReader(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		out  []byte
		buf  []byte
	}{
		{
			name: "Exact",
			in:   []byte("this is a test"),
			out:  []byte("this"),
			buf:  make([]byte, 4),
		},
		{
			name: "Inexact",
			in:   []byte("this is a test"),
			out:  []byte("this"),
			buf:  make([]byte, 3),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			r := &misc.LimitedReader{
				R: bytes.NewReader(test.in),
				N: uint32(len(test.out)),
				E: io.ErrUnexpectedEOF,
			}

			var out []byte
			var err error

			for i := 0; i < 5; i++ {
				n, rerr := r.Read(test.buf)
				out = append(out, test.buf[:n]...)

				if rerr != nil {
					err = rerr
					break
				}
			}

			if !bytes.Equal(out, test.out) {
				t.Errorf("Got output %q but expected %q", out, test.out)
			}
			if err != io.ErrUnexpectedEOF {
				t.Errorf("Got error %v", err)
			}
		})
	}
}
