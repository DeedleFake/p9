package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/DeedleFake/p9"
)

type lsCmd struct{}

func (lsCmd) Name() string {
	return "ls"
}

func (lsCmd) Desc() string {
	return "Lists the files in a directory."
}

func (lsCmd) Run(options GlobalOptions, args []string) {
	fset := flag.NewFlagSet("ls", flag.ExitOnError)
	fset.Parse(args[1:])

	c, err := p9.Dial("tcp", options.Address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to dial address: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	_, err = c.Handshake(uint32(options.MSize))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Handshake failed: %v\n", err)
		os.Exit(1)
	}

	a, err := c.Attach(nil, options.UName, options.AName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to attach: %v\n", err)
		os.Exit(1)
	}
	defer a.Close()

	d, err := a.Open(fset.Arg(0), p9.OREAD)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open %q: %v\n", fset.Arg(0), err)
		os.Exit(1)
	}
	defer d.Close()

	entries, err := d.Readdir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read dir: %v\n", err)
		os.Exit(1)
	}

	for _, entry := range entries {
		fmt.Printf("%v\n", entry.Name)
	}
}

func init() {
	RegisterCommand(&lsCmd{})
}
