package main

import (
	"flag"

	"github.com/DeedleFake/p9"
)

func main() {
	root := flag.String("root", ".", "Directory to serve.")
	flag.Parse()

	err := p9.ListenAndServe("tcp", ":5640", p9.FSConnHandler(p9.Dir(*root), 2048))
	if err != nil {
		panic(err)
	}
}
