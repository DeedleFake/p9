package main

import (
	"flag"
	"fmt"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/internal/util"
	"github.com/hanwen/go-fuse/v2/fs"
)

type mountCmd struct {
}

func (cmd *mountCmd) Name() string {
	return "mount"
}

func (cmd *mountCmd) Desc() string {
	return "Mount a 9P filesystem."
}

func (cmd *mountCmd) Run(options GlobalOptions, args []string) error {
	fset := flag.NewFlagSet(cmd.Name(), flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(fset.Output(), "%v mounts a 9P filesystem.\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Usage: %v <mount point>\n", cmd.Name())
		fset.PrintDefaults()
	}
	err := fset.Parse(args[1:])
	if err != nil {
		return util.Errorf("parse flags: %w", err)
	}

	args = fset.Args()
	if len(args) != 1 {
		return flag.ErrHelp
	}

	return attach(options, func(a *p9.Remote) error {
		s, err := fs.Mount(args[0], &fuse{}, nil)
		if err != nil {
			return err
		}

		s.Wait()
		return nil
	})
}

type fuse struct {
	fs.Inode
}

func init() {
	RegisterCommand(&mountCmd{})
}
