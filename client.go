package p9

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/DeedleFake/p9/internal/debug"
)

var (
	// ErrUnsupportedVersion is returned from a handshake attempt that
	// fails due to a version mismatch.
	ErrUnsupportedVersion = errors.New("unsupported version")
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

	m     sync.RWMutex
	msize uint32
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

		msize: 1024,
	}
	go client.reader(ctx)
	go client.coord(ctx)

	return client
}

// Dial is a convenience function that dials and creates a client in
// the same step.
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
	for {
		err := c.c.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			log.Printf("Failed to set conn deadline: %v", err)
			return
		}

		msg, tag, err := Proto().Receive(c.c, c.msize)
		if err != nil {
			if (err == io.EOF) || (ctx.Err() != nil) {
				return
			}

			continue
		}

		if r, ok := msg.(Rversion); ok {
			c.m.Lock()
			c.msize = r.Msize
			c.m.Unlock()
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
	tags := make(map[uint16]chan interface{})

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
func (c *Client) Send(msg interface{}) (interface{}, error) {
	debug.Log("-> %T\n", msg)

	tag := NoTag
	if _, ok := msg.(Tversion); !ok {
		tag = <-c.nextTag
	}

	ret := make(chan interface{}, 1)
	c.sentMsg <- clientMsg{
		tag: tag,
		ret: ret,
	}

	err := Proto().Send(c.c, tag, msg)
	if err != nil {
		return nil, err
	}

	rsp := <-ret
	debug.Log("<- %T\n", rsp)

	if err, ok := rsp.(Rerror); ok {
		return nil, err
	}
	return rsp, nil
}

// Handshake performs an initial handshake to establish the maximum
// allowed message size. A handshake must be performed before any
// other request types may be sent.
func (c *Client) Handshake(msize uint32) (uint32, error) {
	rsp, err := c.Send(Tversion{
		Msize:   msize,
		Version: Version,
	})
	if err != nil {
		return 0, err
	}

	version := rsp.(Rversion)
	if version.Version != Version {
		return 0, ErrUnsupportedVersion
	}

	return version.Msize, nil
}

// Sometimes I think that some type of tuples would be nice...
type clientMsg struct {
	tag  uint16
	recv interface{}
	ret  chan interface{}
}
