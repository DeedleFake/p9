package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/DeedleFake/p9"
)

type readCmd struct{}

func (readCmd) Name() string {
	return "read"
}

func (readCmd) Desc() string {
	return "Read the contents of a file and print them to stdout."
}

func (readCmd) Run(options GlobalOptions, args []string) error {
	fset := flag.NewFlagSet("read", flag.ExitOnError)
	fset.Parse(args[1:])

	return attach(options, func(a *p9.Remote) error {
		f, err := a.Open(fset.Arg(0), p9.OREAD)
		if err != nil {
			return fmt.Errorf("Failed to open %q: %v\n", fset.Arg(0), err)
		}
		defer f.Close()

		_, err = io.Copy(os.Stdout, f)
		if err != nil {
			return fmt.Errorf("Failed to read: %v\n", err)
		}

		return nil
	})
}

func init() {
	RegisterCommand(&readCmd{})
}
