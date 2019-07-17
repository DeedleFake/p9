package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/DeedleFake/p9"
)

type writeCmd struct{}

func (cmd *writeCmd) Name() string {
	return "write"
}

func (cmd *writeCmd) Desc() string {
	return "Write stdin to a file."
}

func (cmd *writeCmd) Run(options GlobalOptions, args []string) error {
	fset := flag.NewFlagSet(cmd.Name(), flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(fset.Output(), "%v copies the entirety of stdin into a file.", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Usage: %v [options] <path>\n", cmd.Name())
	}
	app := fset.Bool("a", false, "Append to the file instead of overwriting it.")
	create := fset.Uint(
		"c",
		0,
		"If non-zero, create the file, if it doesn't already exist, with the given permissions.",
	)
	err := fset.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("Failed to parse flags: %v", err)
	}

	args = fset.Args()
	if len(args) != 1 {
		fmt.Fprintf(fset.Output(), "Error: Only one file can be specified.\n")
		fmt.Fprintf(fset.Output(), "\n")
		return flag.ErrHelp
	}

	return attach(options, func(a *p9.Remote) error {
		open := func() (*p9.Remote, error) {
			trunc := uint8(p9.OTRUNC)
			if *app {
				trunc = 0
			}

			return a.Open(args[0], p9.OWRITE|trunc)
		}

		if *create != 0 {
			open = func() (*p9.Remote, error) {
				return a.Create(args[0], p9.FileMode(*create), p9.OWRITE)
			}
		}

		f, err := open()
		if err != nil {
			return fmt.Errorf("Failed to open %q: %v", args[0], err)
		}
		defer f.Close()

		if *app {
			_, err := f.Seek(0, io.SeekEnd)
			if err != nil {
				return fmt.Errorf("Failed to seek: %v", err)
			}
		}

		_, err = io.Copy(f, os.Stdin)
		if err != nil {
			return fmt.Errorf("Failed to write: %v", err)
		}

		return nil
	})
}

func init() {
	RegisterCommand(&writeCmd{})
}
