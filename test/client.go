package main

import (
	"fmt"
	"net"

	"github.com/DeedleFake/p9"
)

func main() {
	c, err := net.Dial("tcp", ":5640")
	if err != nil {
		panic(err)
	}

	client := p9.NewClient(c)
	defer client.Close()

	msize, err := client.Handshake(2048)
	if err != nil {
		panic(err)
	}
	fmt.Printf("msize: %v\n", msize)

	root, err := client.Attach(nil, "anyone", "/")
	if err != nil {
		panic(err)
	}
	fmt.Println(root.Type())

	test, err := root.Open("test.txt", p9.OREAD)
	if err != nil {
		panic(err)
	}
	defer test.Close()

	buf := make([]byte, 128)
	n, err := test.ReadAt(buf, 0)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%q\n", buf[:n])
}
