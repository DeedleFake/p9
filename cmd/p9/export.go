package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/internal/util"
	"github.com/DeedleFake/p9/proto"
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
		fmt.Fprintf(fset.Output(), "%v serves a directory over 9P.\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Usage: %v <path>\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Options:\n")
		fset.PrintDefaults()
	}
	rw := fset.Bool("rw", false, "Make exported FS writable.")
	err := fset.Parse(args[1:])
	if err != nil {
		return util.Errorf("parse flags: %w", err)
	}

	args = fset.Args()
	if len(args) != 1 {
		fmt.Fprintf(fset.Output(), "Error: Need exactly one path.\n")
		fmt.Fprintf(fset.Output(), "\n")
		return flag.ErrHelp
	}

	fs := p9.FileSystem(p9.Dir(args[0]))
	if !*rw {
		fs = p9.ReadOnlyFS(fs)
	}

	lis, err := net.Listen(options.Network, options.Address)
	if err != nil {
		return util.Errorf("listen: %w", err)
	}
	defer lis.Close()

	errC := make(chan error, 1)
	go func() {
		err = proto.Serve(
			lis,
			p9.Proto(),
			p9.FSConnHandler(fs, uint32(options.MSize)),
		)
		if err != nil {
			errC <- util.Errorf("serve: %w", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)

	select {
	case err := <-errC:
		return err
	case <-c:
		return nil
	}
}

func init() {
	RegisterCommand(&exportCmd{})
}
