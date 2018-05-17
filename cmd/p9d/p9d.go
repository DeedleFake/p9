package main

import (
	"flag"
	"log"

	"github.com/DeedleFake/p9"
)

func main() {
	net := flag.String("net", "tcp", "Network type to use.")
	addr := flag.String("addr", ":5640", "Address to listen on.")
	root := flag.String("root", ".", "Directory to serve.")
	msize := flag.Uint("msize", 4096, "Maximum size of messages. May be smaller if client requests it.")
	flag.Parse()

	log.Printf("Starting server on %q", *net+"!"+*addr)
	err := p9.ListenAndServe(*net, *addr, p9.FSConnHandler(p9.Dir(*root), uint32(*msize)))
	if err != nil {
		panic(err)
	}
}
