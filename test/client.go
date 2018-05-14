package main

import (
	"fmt"
	"io"
	"os"

	"github.com/DeedleFake/p9"
)

func main() {
	c, err := p9.Dial("tcp", ":5640")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	msize, err := c.Handshake(2048)
	if err != nil {
		panic(err)
	}
	fmt.Printf("msize: %v\n", msize)

	root, err := c.Attach(nil, "anyone", "")
	if err != nil {
		panic(err)
	}
	fmt.Println(root.Type())

	test, err := root.Open("test.txt", p9.OREAD)
	if err != nil {
		panic(err)
	}
	defer test.Close()

	_, err = io.Copy(os.Stdout, test)
	if err != nil {
		panic(err)
	}
}
