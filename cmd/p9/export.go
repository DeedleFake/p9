package main

import (
	"flag"
	"fmt"

	"github.com/DeedleFake/p9"
)

type exportCmd struct{}

func (cmd *exportCmd) Name() string {
	return "export"
}

func (cmd *exportCmd) Desc() string {
	return "Serves a directory over 9P."
}

func (cmd *exportCmd) Run(options GlobalOptions, args []string) error {
	fset := flag.NewFlagSet(cmd.Name(), flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(fset.Output(), "%v serves a directory over 9P.", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Usage: %v <path>\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Options:\n")
		fset.PrintDefaults()
	}
	err := fset.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("Failed to parse flags: %v", err)
	}

	args = fset.Args()
	if len(args) != 1 {
		fmt.Fprintf(fset.Output(), "Error: Need exactly one path.\n")
		fmt.Fprintf(fset.Output(), "\n")
		return flag.ErrHelp
	}

	err = p9.ListenAndServe(
		options.Network,
		options.Address,
		p9.FSConnHandler(p9.Dir(args[0]), uint32(options.MSize)),
	)
	if err != nil {
		return fmt.Errorf("Failed to start server: %v", err)
	}

	return nil
}

func init() {
	RegisterCommand(&exportCmd{})
}