package p9

import (
	"encoding/binary"
	"fmt"
	"io"
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
func (e *encoder) Encode(v interface{}) error {
	if e.err != nil {
		return e.err
	}

	if v, ok := v.(encodable); ok {
		v.encode(e)
		return e.err
	}

	switch v := v.(type) {
	case uint8, uint16, uint32, uint64, int8, int16, int32, int64:
		e.err = e.mode(v)
		return e.err

	case []byte:
		err := e.mode(uint32(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		e.err = e.mode(v)
		return e.err

	case string:
		err := e.mode(uint16(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		e.err = e.mode([]byte(v))
		return e.err

	case []string:
		err := e.mode(uint16(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		for _, str := range v {
			err := e.Encode(str)
			if err != nil {
				e.err = err
				return err
			}
		}

		return e.err

	case []QID:
		err := e.mode(uint16(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		for _, qid := range v {
			err := e.Encode(qid)
			if err != nil {
				e.err = err
				return err
			}
		}

		return e.err

	default:
		panic(fmt.Errorf("Unexpected type: %T", v))
	}
}

// A decoder decodes the various types necessary for the protocal. It
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
func (d *decoder) Decode(v interface{}) error {
	if d.err != nil {
		return d.err
	}

	if v, ok := v.(decodable); ok {
		v.decode(d)
		return d.err
	}

	switch v := v.(type) {
	case *uint8, *uint16, *uint32, *uint64, *int8, *int16, *int32, *int64:
		d.err = binary.Read(d, binary.LittleEndian, v)
		return d.err

	case *[]byte:
		var size uint32
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return err
		}

		*v = make([]byte, size)
		d.err = binary.Read(d, binary.LittleEndian, v)
		return d.err

	case *string:
		var size uint16
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return err
		}

		buf := make([]byte, size)
		d.err = binary.Read(d, binary.LittleEndian, buf)
		*v = string(buf)
		return d.err

	case *[]string:
		var size uint16
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return err
		}

		*v = make([]string, size)
		for i := range *v {
			err := d.Decode(&(*v)[i])
			if err != nil {
				d.err = err
				return err
			}
		}

		return d.err

	case *[]QID:
		var size uint16
		err := binary.Read(d, binary.LittleEndian, &size)
		if err != nil {
			d.err = err
			return err
		}

		*v = make([]QID, size)
		for i := range *v {
			err := d.Decode(&(*v)[i])
			if err != nil {
				d.err = err
				return err
			}
		}

		return d.err

	default:
		panic(fmt.Errorf("Unexpected type: %T", v))
	}
}
