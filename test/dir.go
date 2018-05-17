package main

import (
	"errors"
	"flag"

	"github.com/DeedleFake/p9"
)

func main() {
	root := flag.String("root", ".", "Directory to serve.")
	flag.Parse()

	fs := p9.AuthFS{
		FileSystem: p9.Dir(*root),

		AuthFunc: func(user, aname string) (p9.File, error) {
			return nil, errors.New("Actually, auth is still not supported. Ironic, huh?")
		},

		AttachFunc: func(afile p9.File, user, aname string) (p9.File, error) {
			return nil, nil
		},
	}

	err := p9.ListenAndServe("tcp", ":5640", p9.FSConnHandler(fs, 2048))
	if err != nil {
		panic(err)
	}
}
