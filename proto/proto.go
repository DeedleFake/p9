// Package proto provides a usage-inspecific wrapper around 9P's data
// serialization and communication scheme.
package proto

import (
	"fmt"
	"io"
)

const (
	// NoTag is a special tag that is used when a tag is unimportant.
	NoTag uint16 = 0xFFFF
)

// Proto maps integer type IDs to functions that create the correct
// message type for that ID. In other words, if the value sent over
// the network to indication a specific message type is 1, then index
// 1 in the Proto should be a function that returns that message type.
//
// Note that message types are serialized as unsigned 8-bit integers,
// so the use of any indices outside of the range that can be
// represented by those is undefined.
type Proto []func() interface{}

func (p Proto) Receive(r io.Reader, msize uint32) (msg interface{}, tag uint16, err error) {
	var size uint32
	err = Read(r, &size)
	if err != nil {
		return nil, NoTag, fmt.Errorf("receive: %w", err)
	}

	lr := &limitedReader{
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
	if (msgType < 0) || (int(msgType) > len(p)) || (p[int(msgType)] == nil) {
		if err != nil {
			return nil, NoTag, err
		}

		return nil, NoTag, fmt.Errorf("receive: invalid message type: %v", msgType)
	}

	tag = NoTag
	read(&tag)

	msg = p[int(msgType)]()
	read(msg)

	return msg, tag, err
}

func Send(w io.Writer, tag uint16, msgType uint8, msg interface{}) (err error) {
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
