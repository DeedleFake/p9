package main

import (
	"fmt"
	"log"
	"net"

	"github.com/DeedleFake/p9"
)

func connHandler() p9.MessageHandler {
	return p9.MessageHandlerFunc(func(msg p9.Message) p9.Message {
		log.Printf("%#v", msg)

		switch msg := msg.(type) {
		case *p9.Tversion:
			return &p9.Rversion{
				Msize:   msg.Msize,
				Version: "9P2000",
			}

		default:
			return &p9.Rerror{
				Ename: fmt.Sprintf("Unsupported message type: %T", msg),
			}
		}
	})
}

func main() {
	lis, err := net.Listen("tcp", "localhost:5640")
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer lis.Close()

	err = p9.Serve(lis, p9.ConnHandlerFunc(connHandler))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
