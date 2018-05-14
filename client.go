package p9

import (
	"context"
	"io"
	"log"
	"net"
	"time"
)

// Client provides functionality for sending requests to and receiving
// responses from a 9P server. It automatically handles message tags,
// properly blocking until a matching tag response has been received.
type Client struct {
	cancel func()

	c net.Conn

	sentMsg chan clientMsg
	recvMsg chan clientMsg

	nextTag chan uint16
	nextFID chan uint32
}

// NewClient initializes a client that communicates using c. The
// Client will close c when the Client is closed.
func NewClient(c net.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		cancel: cancel,

		c: c,

		sentMsg: make(chan clientMsg),
		recvMsg: make(chan clientMsg),

		nextTag: make(chan uint16),
		nextFID: make(chan uint32),
	}
	go client.reader(ctx)
	go client.coord(ctx)

	return client
}

// Dial is a convience function that dials and creates a client in the
// same step.
func Dial(network, addr string) (*Client, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	return NewClient(c), nil
}

// Close cleans up resources created by the client as well as closing
// the underlying connection.
func (c *Client) Close() error {
	c.cancel()
	return c.c.Close()
}

// reader reads messages from the connection, sending them to the
// coordinator to be sent to waiting Send calls.
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

// coord coordinates between Send calls and the reader.
func (c *Client) coord(ctx context.Context) {
	var nextTag uint16
	tags := make(map[uint16]chan Message)

	var nextFID uint32

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

		case c.nextFID <- nextFID:
			nextFID++
		}
	}
}

// Send sends a message to the server, blocking until a response has
// been received. It is safe to place multiple Send calls
// concurrently, and each will return when the response to that
// request has been received.
func (c *Client) Send(msg Message) (Message, error) {
	tag := NoTag
	if _, ok := msg.(*Tversion); !ok {
		tag = <-c.nextTag
	}

	ret := make(chan Message, 1)
	c.sentMsg <- clientMsg{
		tag: tag,
		ret: ret,
	}

	err := WriteMessage(c.c, tag, msg)
	if err != nil {
		return nil, err
	}

	rsp := <-ret
	if err, ok := rsp.(*Rerror); ok {
		return nil, err
	}
	return rsp, nil
}

// Sometimes I think that some type of tuples would be nice...
type clientMsg struct {
	tag  uint16
	recv Message
	ret  chan Message
}
