package p9

import (
	"io"
	"log"
	"net"
)

func Serve(lis net.Listener, connHandler ConnHandler) (err error) {
	for {
		c, err := lis.Accept()
		if err != nil {
			return err
		}

		go func() {
			defer c.Close()

			if h, ok := connHandler.(handleConn); ok {
				h.HandleConn(c)
			}
			if h, ok := connHandler.(handleDisconnect); ok {
				defer h.HandleDisconnect(c)
			}

			handleMessages(c, connHandler.MessageHandler())
		}()
	}
}

func handleMessages(c net.Conn, handler MessageHandler) {
	var msize uint32
	mode := func(f func()) {
		f()
	}

	for {
		tmsg, tag, err := ReadMessage(c, msize)
		if err != nil {
			if err == io.EOF {
				return
			}

			log.Printf("Error reading message: %v", err)
		}

		mode(func() {
			rmsg := handler.HandleMessage(tmsg)
			if rmsg, ok := rmsg.(*Rversion); ok {
				if msize > 0 {
					panic("Attempted to set msize twice")
				}

				msize = rmsg.Msize
				mode = func(f func()) {
					go f()
				}
			}

			err := WriteMessage(c, tag, rmsg)
			if err != nil {
				log.Printf("Error writing message: %v", err)
			}
		})
	}
}

type ConnHandler interface {
	MessageHandler() MessageHandler

	// Optional methods:
	// HandleConn(c net.Conn)
	// HandleDisconnect(c net.Conn)
}

type handleConn interface {
	HandleConn(c net.Conn)
}

type handleDisconnect interface {
	HandleDisconnect(c net.Conn)
}

type ConnHandlerFunc func() MessageHandler

func (h ConnHandlerFunc) MessageHandler() MessageHandler {
	return h()
}

type MessageHandler interface {
	HandleMessage(Message) Message
}

type MessageHandlerFunc func(Message) Message

func (h MessageHandlerFunc) HandleMessage(msg Message) Message {
	return h(msg)
}
