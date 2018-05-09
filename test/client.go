package main

import (
	"log"
	"net"

	"github.com/DeedleFake/p9"
)

func main() {
	c, err := net.Dial("tcp", ":5640")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	client := p9.NewClient(c)
	defer client.Close()

	msg, err := client.Send(&p9.Tversion{
		Msize:   1000,
		Version: "9P2000",
	})
	if err != nil {
		panic(err)
	}
	log.Printf("%#v", msg)
}
