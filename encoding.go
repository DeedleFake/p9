package p9

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// encodable is a type that knows how to encode itself using an
// encoder.
type encodable interface {
	encode(e *encoder)
}

// decodable is a type that knows how to decode itself using a
// decoder.
type decodable interface {
	decode(d *decoder)
}

// An encoder encodes the various types necessary for the protocol. It
// only handles those types, however. Attempting to use it with other
// types will panic.
type encoder struct {
	w   io.Writer
	n   uint32
	err error

	mode func(interface{}) error
}

// size is an encoder mode that calculates the total size of an
// encoded message.
func (e *encoder) size(v interface{}) error {
	e.n += uint32(binary.Size(v))
	return nil
}

// write is an encoder mode that actually writes messages.
func (e *encoder) write(v interface{}) error {
	return binary.Write(e, binary.LittleEndian, v)
}

func (e *encoder) Write(data []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}

	n, err := e.w.Write(data)
	e.err = err
	return n, err
}

// Encode handles v using the current encoder mode.
func (e *encoder) Encode(v interface{}) {
	if e.err != nil {
		return
	}

	switch v := v.(type) {
	case encodable:
		v.encode(e)

	case uint8, uint16, uint32, uint64, int8, int16, int32, int64:
		e.err = e.mode(v)

	case []byte:
		err := e.mode(uint32(len(v)))
		if err != nil {
			e.err = err
			return
		}

		e.err = e.mode(v)

	case string:
		err := e.mode(uint16(len(v)))
		if err != nil {
			e.err = err
			return
		}

		e.err = e.mode([]byte(v))

	case []string:
		err := e.mode(uint16(len(v)))
		if err != nil {
			e.err = err
			return
		}

		for _, str := range v {
			e.Encode(str)
			if e.err != nil {
				return
			}
		}

	case []QID:
		err := e.mode(uint16(len(v)))
		if err != nil {
			e.err = err
			return
		}

		for _, qid := range v {
			e.Encode(qid)
			if e.err != nil {
				return
			}
		}

	case time.Time:
		e.err = e.mode(uint32(v.Unix()))

	default:
		panic(fmt.Errorf("Unexpected type: %T", v))
	}
}

// A decoder decodes the various types necessary for the protocol. It
// only handles those types. Attempting to use a value of another type
// with it will cause a panic.
type decoder struct {
	r   io.Reader
	err error
}

func (d *decoder) Read(buf []byte) (int, error) {
	if d.err != nil {
		return 0, d.err
	}

	n, err := d.r.Read(buf)
	d.err = err
	return n, err
}

// Decode reads and decodes a value from the underlying io.Reader into
// v.
func (d *decoder) Decode(v interface{}) {
	if d.err != nil {
		return
	}

	switch v := v.(type) {
	case decodable:
		v.decode(d)

	case *uint8, *uint16, *uint32, *uint64, *int8, *int16, *int32, *int64:
		d.err = binary.Read(d, binary.LittleEndian, v)

	case *[]byte:
		var size uint32
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return
		}

		*v = make([]byte, size)
		d.err = binary.Read(d, binary.LittleEndian, v)

	case *string:
		var size uint16
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return
		}

		buf := make([]byte, size)
		d.err = binary.Read(d, binary.LittleEndian, buf)
		*v = string(buf)

	case *[]string:
		var size uint16
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return
		}

		*v = make([]string, size)
		for i := range *v {
			d.Decode(&(*v)[i])
			if d.err != nil {
				return
			}
		}

	case *[]QID:
		var size uint16
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return
		}

		*v = make([]QID, size)
		for i := range *v {
			d.Decode(&(*v)[i])
			if d.err != nil {
				return
			}
		}

	case *time.Time:
		var sec uint32
		err := binary.Read(d, binary.LittleEndian, &sec)
		if err != nil {
			d.err = err
			return
		}

		*v = time.Unix(int64(sec), 0)

	default:
		panic(fmt.Errorf("Unexpected type: %T", v))
	}
}

// ReadDir decodes a series of directory entries from a reader. It
// reads until EOF, so it doesn't return io.EOF as a possible error.
//
// It is recommended that the reader passed to ReadDir have some form
// of buffering, as some servers will silently mishandle attempts to
// read pieces of a directory. Wrapping the reader with a bufio.Reader
// is often sufficient.
func ReadDir(r io.Reader) ([]DirEntry, error) {
	d := &decoder{
		r: r,
	}

	var entries []DirEntry
	for {
		var stat Stat
		d.Decode(&stat)
		if d.err != nil {
			if d.err == io.EOF {
				d.err = nil
			}
			return entries, d.err
		}

		entries = append(entries, stat.dirEntry())
	}
}

// WriteDir writes a series of directory entries to w. It uses getPath
// to lookup the QID path of each entry by name. If getPath returns an
// error, that error is immediately returned.
func WriteDir(w io.Writer, entries []DirEntry, getPath func(string) (uint64, error)) error {
	e := &encoder{
		w: w,
	}
	e.mode = e.write

	for _, entry := range entries {
		p, err := getPath(entry.Name)
		if err != nil {
			return err
		}

		e.Encode(entry.stat(p))
	}

	return e.err
}
