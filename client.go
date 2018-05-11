package p9

import (
	"context"
	"io"
	"log"
	"net"
	"time"
)

type Client struct {
	cancel func()

	c   net.Conn
	tag uint16

	sentMsg chan clientMsg
	recvMsg chan clientMsg
	nextTag chan uint16
}

func NewClient(c net.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		cancel: cancel,

		c: c,

		sentMsg: make(chan clientMsg),
		recvMsg: make(chan clientMsg),
		nextTag: make(chan uint16),
	}
	go client.reader(ctx)
	go client.coord(ctx)

	return client
}

func (c *Client) Close() error {
	c.cancel()
	return nil
}

func (c *Client) reader(ctx context.Context) {
	msize := uint32(1024)

	for {
		err := c.c.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			log.Printf("Failed to set conn deadline: %v", err)
			return
		}

		msg, tag, err := ReadMessage(c.c, msize)
		if err != nil {
			if (err == io.EOF) || (ctx.Err() != nil) {
				return
			}

			continue
		}

		if r, ok := msg.(*Rversion); ok {
			msize = r.Msize
		}

		select {
		case <-ctx.Done():
			return

		case c.recvMsg <- clientMsg{
			tag:  tag,
			recv: msg,
		}:
		}
	}
}

func (c *Client) coord(ctx context.Context) {
	var nextTag uint16
	tags := make(map[uint16]chan Message)

	for {
		select {
		case <-ctx.Done():
			close(c.nextTag)
			return

		case cm := <-c.sentMsg:
			tags[cm.tag] = cm.ret

		case cm := <-c.recvMsg:
			rcm, ok := tags[cm.tag]
			if !ok {
				continue
			}

			rcm <- cm.recv

		case c.nextTag <- nextTag:
			nextTag++
		}
	}
}

func (c *Client) Send(msg Message) (Message, error) {
	tag := NoTag
	if _, ok := msg.(*Tversion); !ok {
		tag = <-c.nextTag
	}

	err := WriteMessage(c.c, tag, msg)
	if err != nil {
		return nil, err
	}

	ret := make(chan Message, 1)
	c.sentMsg <- clientMsg{
		tag: tag,
		ret: ret,
	}

	rsp := <-ret
	if err, ok := rsp.(*Rerror); ok {
		return nil, err
	}
	return rsp, nil
}

type clientMsg struct {
	tag  uint16
	recv Message
	ret  chan Message
}
