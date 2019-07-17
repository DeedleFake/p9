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
		fmt.Fprintf(fset.Output(), "%v reads the raw contents of a file and prints them to stdout.\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Usage: %v <path...>\n", cmd.Name())
	}
	err := fset.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("Failed to parse flags: %v", err)
	}

	args = fset.Args()
	if len(args) == 0 {
		fmt.Fprintf(fset.Output(), "Error: Need at least one path.\n")
		fmt.Fprintf(fset.Output(), "\n")
		return flag.ErrHelp
	}

	return attach(options, func(a *p9.Remote) error {
		for _, arg := range args {
			f, err := a.Open(arg, p9.OREAD)
			if err != nil {
				return fmt.Errorf("Failed to open %q: %v", arg, err)
			}
			defer f.Close()

			_, err = io.Copy(os.Stdout, f)
			if err != nil {
				return fmt.Errorf("Failed to read: %v", err)
			}
		}

		return nil
	})
}

func init() {
	RegisterCommand(&readCmd{})
}
