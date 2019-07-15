package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/DeedleFake/p9"
)

type readCmd struct{}

func (cmd *readCmd) Name() string {
	return "read"
}

func (cmd *readCmd) Desc() string {
	return "Read the contents of a file and print them to stdout."
}

func (cmd *readCmd) Run(options GlobalOptions, args []string) error {
	fset := flag.NewFlagSet(cmd.Name(), flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, "%v reads the raw contents of a file and prints them to stdout.\n", cmd.Name())
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Usage: %v <path>\n", cmd.Name())
	}
	fset.Parse(args[1:])

	return attach(options, func(a *p9.Remote) error {
		f, err := a.Open(fset.Arg(0), p9.OREAD)
		if err != nil {
			return fmt.Errorf("Failed to open %q: %v", fset.Arg(0), err)
		}
		defer f.Close()

		_, err = io.Copy(os.Stdout, f)
		if err != nil {
			return fmt.Errorf("Failed to read: %v", err)
		}

		return nil
	})
}

func init() {
	RegisterCommand(&readCmd{})
}
