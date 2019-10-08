package p9

import (
	"io"
	"log"
	"net"
)

// Serve serves a 9P server, listening for new connection on lis and
// handling them using the provided handler.
//
// Note that to avoid a data race, messages from a single client are
// handled entirely sequentially until an msize has been established,
// at which point they will be handled concurrently.
func Serve(lis net.Listener, connHandler ConnHandler) (err error) {
	for {
		c, err := lis.Accept()
		if err != nil {
			return err
		}

		go func() {
			defer c.Close() // nolint

			if h, ok := connHandler.(handleConn); ok {
				h.HandleConn(c)
			}
			if h, ok := connHandler.(handleDisconnect); ok {
				defer h.HandleDisconnect(c)
			}

			mh := connHandler.MessageHandler()
			if c, ok := mh.(io.Closer); ok {
				defer c.Close() // nolint
			}

			handleMessages(c, mh)
		}()
	}
}

// ListenAndServe is a convenience function that establishes listener,
// via net.Listen(), and then calls Serve().
func ListenAndServe(network, addr string, connHandler ConnHandler) (rerr error) {
	lis, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	defer func() {
		err := lis.Close()
		if (err != nil) && (rerr == nil) {
			rerr = err
		}
	}()

	return Serve(lis, connHandler)
}

func handleMessages(c net.Conn, handler MessageHandler) {
	var msize uint32
	mode := func(f func()) {
		f()
	}

	for {
		tmsg, tag, err := Proto().Receive(c, msize)
		if err != nil {
			if err == io.EOF {
				return
			}

			log.Printf("Error reading message: %v", err)
		}

		mode(func() {
			rmsg := handler.HandleMessage(tmsg)
			if rmsg, ok := rmsg.(Rversion); ok {
				if msize > 0 {
					panic("Attempted to set msize twice")
				}

				msize = rmsg.Msize
				mode = func(f func()) {
					go f()
				}
			}

			err := Proto().Send(c, tag, rmsg)
			if err != nil {
				log.Printf("Error writing message: %v", err)
			}
		})
	}
}

// ConnHandler initializes new MessageHandlers for incoming
// connections. Unlike HTTP, which is a connectionless protocol, 9P
// requires that each connection be handled as a unique client session
// with a stored state, hence this two-step process.
//
// If a ConnHandler provides a HandleConn(net.Conn) method, that
// method will be called when a new connection is made. Similarly, if
// it provides a HandleDisconnect(net.Conn) method, that method will
// be called when a connection is ended.
type ConnHandler interface {
	MessageHandler() MessageHandler
}

type handleConn interface {
	HandleConn(c net.Conn)
}

type handleDisconnect interface {
	HandleDisconnect(c net.Conn)
}

// ConnHandlerFunc allows a function to be used as a ConnHandler.
type ConnHandlerFunc func() MessageHandler

func (h ConnHandlerFunc) MessageHandler() MessageHandler { // nolint
	return h()
}

// MessageHandler handles messages for a single client connection.
//
// If a MessageHandler also implements io.Closer, then Close() will be
// called when the connection ends. Its return value is ignored.
type MessageHandler interface {
	// HandleMessage is passed received messages from the client. Its
	// return value is then sent back to the client with the same tag.
	HandleMessage(interface{}) interface{}
}

// MessageHandlerFunc allows a function to be used as a MessageHandler.
type MessageHandlerFunc func(interface{}) interface{}

func (h MessageHandlerFunc) HandleMessage(msg interface{}) interface{} { // nolint
	return h(msg)
}
