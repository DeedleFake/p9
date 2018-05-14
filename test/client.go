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
	defer root.Close()
	fmt.Println(root.Type())

	test, err := root.Open("test", p9.OREAD)
	if err != nil {
		panic(err)
	}
	defer test.Close()

	_, err = io.Copy(os.Stdout, test)
	if err != nil {
		panic(err)
	}

	_, err = test.Seek(-5, io.SeekEnd)
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(os.Stdout, test)
	if err != nil {
		panic(err)
	}

	dir, err := root.Open("/", p9.OREAD)
	if err != nil {
		panic(err)
	}
	defer dir.Close()

	stat, err := dir.Stat()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", stat)

	entries, err := dir.Readdir()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v entries:\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("\t%v\n", entry.Name)
	}
}
