package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"
)

var (
	ErrLargeMessage = errors.New("message larger than msize")
)

func Size(v interface{}) (uint32, error) {
	e := &encoder{}
	e.mode = e.size

	e.encode(v)
	return e.n, e.err
}

func Write(w io.Writer, v interface{}) error {
	e := &encoder{w: w}
	e.mode = e.write

	e.encode(v)
	return e.err
}

func Read(r io.Reader, v interface{}) error {
	panic("Not implemented.")
}

type encoder struct {
	w   io.Writer
	n   uint32
	err error

	mode func(interface{})
}

func (e *encoder) size(v interface{}) {
	e.n += uint32(binary.Size(v))
}

func (e *encoder) write(v interface{}) {
	if e.err != nil {
		return
	}

	e.err = binary.Write(e.w, binary.LittleEndian, v)
}

func (e *encoder) encode(v interface{}) {
	if e.err != nil {
		return
	}

	switch v := v.(type) {
	case time.Time:
		e.mode(uint32(v.Unix()))
		return
	case *time.Time:
		e.mode(uint32(v.Unix()))
		return
	}

	switch rv := reflect.Indirect(reflect.ValueOf(v)); rv.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uintptr:
		e.mode(rv.Interface())

	case reflect.Array, reflect.Slice:
		switch rv.Type().Elem().Kind() {
		case reflect.Uint8:
			e.mode(uint32(rv.Len()))
		default:
			e.mode(uint16(rv.Len()))
		}

		for i := 0; i < rv.Len(); i++ {
			e.encode(rv.Index(i))
		}

	case reflect.String:
		e.mode(uint16(rv.Len()))
		e.mode([]byte(rv.String()))

	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			e.encode(rv.Field(i))
		}

	default:
		e.err = fmt.Errorf("invalid type: %T", v)
	}
}
