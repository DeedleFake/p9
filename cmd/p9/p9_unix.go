// +build linux darwin

package main

import (
	"os/user"
	"path/filepath"
)

func getNamespaceHost(network, host, port string) (string, string, string) {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	network = "unix"
	host = filepath.Join("/", "tmp", "ns."+u.Username+".:0", host[1:])
	port = ""

	return network, host, port
}
