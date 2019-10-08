package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"
	"unsafe"
)

var (
	ErrLargeMessage = errors.New("message larger than msize")
)

func Size(v interface{}) (uint32, error) {
	e := &encoder{}
	e.mode = e.size

	e.encode(reflect.ValueOf(v))
	return e.n, e.err
}

func Write(w io.Writer, v interface{}) error {
	e := &encoder{w: w}
	e.mode = e.write

	e.encode(reflect.ValueOf(v))
	return e.err
}

func Read(r io.Reader, v interface{}) error {
	d := &decoder{r: r}
	d.decode(reflect.ValueOf(v))
	return d.err
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

func (e *encoder) encode(v reflect.Value) {
	if e.err != nil {
		return
	}

	v = reflect.Indirect(v)

	switch v := v.Interface().(type) {
	case time.Time:
		e.mode(uint32(v.Unix()))
		return
	}

	switch v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uintptr:
		e.mode(v.Interface())

	case reflect.Array, reflect.Slice:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			e.mode(uint32(v.Len()))
		default:
			e.mode(uint16(v.Len()))
		}

		for i := 0; i < v.Len(); i++ {
			e.encode(v.Index(i))
		}

	case reflect.String:
		e.mode(uint16(v.Len()))
		e.mode([]byte(v.String()))

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			e.encode(v.Field(i))
		}

	default:
		e.err = fmt.Errorf("invalid type: %T", v)
	}
}

type decoder struct {
	r   io.Reader
	err error
}

func (d *decoder) read(v interface{}) {
	if d.err != nil {
		return
	}

	d.err = binary.Read(d.r, binary.LittleEndian, v)
}

func (d *decoder) decode(v reflect.Value) {
	if d.err != nil {
		return
	}

	v = reflect.Indirect(v)

	switch v.Interface().(type) {
	case time.Time:
		var unix uint32
		d.read(&unix)
		v.Set(reflect.ValueOf(time.Unix(int64(unix), 0)))
		return
	}

	switch v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uintptr:
		d.read(v.Interface())

	case reflect.Slice:
		var length uint32
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			d.read(&length)
		default:
			d.read((*uint16)(unsafe.Pointer(&length)))
		}

		if int(length) > v.Cap() {
			v.Set(reflect.MakeSlice(v.Type(), int(length), int(length)))
		}
		v.Set(v.Slice(0, int(length)))

		for i := 0; i < v.Len(); i++ {
			d.decode(v.Index(i))
		}

	case reflect.String:
		var length uint16
		d.read(&length)

		buf := make([]byte, int(length))
		d.read(buf)

		v.SetString(*(*string)(unsafe.Pointer(&buf)))

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			d.decode(v.Field(i))
		}

	default:
		d.err = fmt.Errorf("invalid type: %T", v)
	}
}
