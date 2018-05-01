package p9

import (
	"fmt"
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
			defer func() {
				err := recover()
				if err != nil {
					log.Printf("Panic in connection handler: %v", err)
				}
			}()

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
	for {
		tmsg, tag, err := ReadMessage(c)
		if err != nil {
			if err == io.EOF {
				return
			}

			panic(fmt.Errorf("Error reading message: %v", err))
		}

		go func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Printf("Panic in message handler: %v", err)
				}
			}()

			rmsg := handler.HandleMessage(tmsg)
			err := WriteMessage(c, tag, rmsg)
			if err != nil {
				panic(fmt.Errorf("Error writing message: %v", err))
			}
		}()
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
