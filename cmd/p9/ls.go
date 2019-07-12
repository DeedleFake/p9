package main

import (
	"flag"
	"fmt"

	"github.com/DeedleFake/p9"
)

func init() {
	RegisterCommand(NewRemoteCommand(
		"ls",
		"List the files in a directory.",
		func(a *p9.Remote, args []string) error {
			fset := flag.NewFlagSet("ls", flag.ExitOnError)
			fset.Parse(args[1:])

			d, err := a.Open(fset.Arg(0), p9.OREAD)
			if err != nil {
				return fmt.Errorf("Failed to open %q: %v\n", fset.Arg(0), err)
			}
			defer d.Close()

			entries, err := d.Readdir()
			if err != nil {
				return fmt.Errorf("Failed to read dir: %v\n", err)
			}

			for _, entry := range entries {
				fmt.Printf("%v\n", entry.Name)
			}

			return nil
		},
	))
}
