// Package proto provides a usage-inspecific wrapper around 9P's data
// serialization and communication scheme.
package proto

import (
	"fmt"
	"io"
	"reflect"

	"github.com/DeedleFake/p9/internal/misc"
)

const (
	// NoTag is a special tag that is used when a tag is unimportant.
	NoTag uint16 = 0xFFFF
)

type Proto struct {
	rmap map[uint8]reflect.Type
	smap map[reflect.Type]uint8
}

func NewProto(mapping map[uint8]reflect.Type) Proto {
	smap := make(map[reflect.Type]uint8, len(mapping))
	for k, v := range mapping {
		smap[v] = k
	}

	return Proto{
		rmap: mapping,
		smap: smap,
	}
}

func (p Proto) TypeFromID(id uint8) reflect.Type {
	return p.rmap[id]
}

func (p Proto) IDFromType(t reflect.Type) (uint8, bool) {
	id, ok := p.smap[t]
	return id, ok
}

func (p Proto) Receive(r io.Reader, msize uint32) (msg interface{}, tag uint16, err error) {
	var size uint32
	err = Read(r, &size)
	if err != nil {
		return nil, NoTag, fmt.Errorf("receive: %w", err)
	}

	if (msize > 0) && (size > msize) {
		return nil, NoTag, fmt.Errorf("receive: %w", ErrLargeMessage)
	}

	lr := &misc.LimitedReader{
		R: r,
		N: size,
		E: ErrLargeMessage,
	}

	read := func(v interface{}) {
		if err != nil {
			return
		}

		err = Read(lr, v)
		if err != nil {
			err = fmt.Errorf("receive: %w", err)
		}
	}

	var msgType uint8
	read(&msgType)

	t := p.TypeFromID(msgType)
	if t == nil {
		if err != nil {
			return nil, NoTag, err
		}

		return nil, NoTag, fmt.Errorf("receive: invalid message type: %v", msgType)
	}

	tag = NoTag
	read(&tag)

	m := reflect.New(t)
	read(m.Interface())

	return m.Elem().Interface(), tag, err
}

func (p Proto) Send(w io.Writer, tag uint16, msg interface{}) (err error) {
	msgType, ok := p.IDFromType(reflect.Indirect(reflect.ValueOf(msg)).Type())
	if !ok {
		return fmt.Errorf("send: invalid message type: %T", msg)
	}

	write := func(v interface{}) {
		if err != nil {
			return
		}

		err = Write(w, v)
		if err != nil {
			err = fmt.Errorf("send: %w", err)
		}
	}

	n, err := Size(msg)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}

	write(4 + 1 + 2 + n)
	write(msgType)
	write(tag)
	write(msg)

	return err
}
