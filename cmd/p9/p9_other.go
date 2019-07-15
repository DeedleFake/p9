// +build !linux,!darwin

package main

func getNamespaceHost(network, host, port string) (string, string, string) {
	return network, host, port
}
