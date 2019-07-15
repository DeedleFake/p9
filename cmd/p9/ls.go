package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/DeedleFake/p9"
)

type lsCmd struct {
	showDetails bool
}

func (cmd *lsCmd) Name() string {
	return "ls"
}

func (cmd *lsCmd) Desc() string {
	return "List the files in a directory."
}

func (cmd *lsCmd) Run(options GlobalOptions, args []string) error {
	fset := flag.NewFlagSet("ls", flag.ExitOnError)
	fset.BoolVar(&cmd.showDetails, "l", false, "Show details.")
	fset.Parse(args[1:])

	return attach(options, func(a *p9.Remote) error {
		d, err := a.Open(fset.Arg(0), p9.OREAD)
		if err != nil {
			return fmt.Errorf("Failed to open %q: %v", fset.Arg(0), err)
		}
		defer d.Close()

		fi, err := d.Stat("")
		if err != nil {
			return fmt.Errorf("Failed to stat: %v", err)
		}

		if fi.Type&p9.QTDir == 0 {
			cmd.printEntry(fi)
			return nil
		}

		fi.Name = "."
		cmd.printEntry(fi)

		entries, err := d.Readdir()
		if err != nil {
			return fmt.Errorf("Failed to read dir: %v", err)
		}

		for _, entry := range entries {
			cmd.printEntry(entry)
		}

		return nil
	})
}

func (cmd *lsCmd) printEntry(entry p9.DirEntry) {
	if cmd.showDetails {
		fmt.Printf("%v ", os.FileMode(entry.Mode))
	}
	fmt.Printf("%v\n", entry.Name)
}

func init() {
	RegisterCommand(&lsCmd{})
}
