// +build linux darwin

package main

import (
	"os"
	"os/user"
	"path/filepath"
)

func getNamespace(name string) (network, addr string) {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	nsdir := filepath.Join("/", "tmp", "ns."+u.Username+".:0")
	os.MkdirAll(nsdir, 0700)

	return "unix", filepath.Join(nsdir, name)
}
