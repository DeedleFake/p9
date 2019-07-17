// +build linux darwin

package main

import (
	"os"
	"os/user"
	"path/filepath"
)

func getNamespaceHost(network, host, port string) (string, string, string) {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	tmpdir := os.Getenv("TMPDIR")
	if tmpdir == "" {
		tmpdir = "/tmp"
	}

	nsdir := filepath.Join(tmpdir, "ns."+u.Username+".:0")
	_ = os.MkdirAll(nsdir, 0700)

	return "unix", filepath.Join(nsdir, host[1:]), ""
}
