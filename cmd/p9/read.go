package main

import (
	"archive/tar"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/internal/util"
)

type readCmd struct {
	tar bool
}

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
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Options:\n")
		fset.PrintDefaults()
	}
	fset.BoolVar(&cmd.tar, "tar", false, "Output files as tar.")
	err := fset.Parse(args[1:])
	if err != nil {
		return util.Errorf("parse flags: %v", err)
	}

	args = fset.Args()
	if len(args) == 0 {
		fmt.Fprintf(fset.Output(), "Error: Need at least one path.\n")
		fmt.Fprintf(fset.Output(), "\n")
		return flag.ErrHelp
	}

	writeFile := func(arg string, f *p9.Remote) error {
		defer f.Close()

		_, err = io.Copy(os.Stdout, f)
		if err != nil {
			return util.Errorf("read %q: %w", arg, err)
		}

		return nil
	}

	if cmd.tar {
		out := tar.NewWriter(os.Stdout)
		defer out.Close()

		writeFile = func(arg string, f *p9.Remote) error {
			defer f.Close()

			fi, err := f.Stat("")
			if err != nil {
				return util.Errorf("stat %q: %w", arg, err)
			}

			switch {
			case fi.FileMode&(p9.ModeAppend|p9.ModeExclusive|p9.ModeMount|p9.ModeAuth) != 0:

			case fi.IsDir():
				children, err := f.Readdir()
				if err != nil {
					return util.Errorf("read dir %q: %w", arg, err)
				}

				for _, c := range children {
					cf, err := f.Open(c.EntryName, p9.OREAD)
					if err != nil {
						return util.Errorf("open %q: %w", path.Join(arg, c.EntryName), err)
					}

					err = writeFile(path.Join(arg, c.EntryName), cf)
					if err != nil {
						return err
					}
				}

			default:
				hdr, err := tar.FileInfoHeader(fi, "")
				if err != nil {
					return util.Errorf("file info header for %q: %w", arg, err)
				}
				hdr.Name = arg
				hdr.Uname = fi.UID
				hdr.Gname = fi.GID

				err = out.WriteHeader(hdr)
				if err != nil {
					return util.Errorf("write header for %q: %w", arg, err)
				}

				_, err = io.Copy(out, f)
				if err != nil {
					return util.Errorf("read %q: %w", arg, err)
				}
			}

			return nil
		}
	}

	return attach(options, func(a *p9.Remote) error {
		for _, arg := range args {
			f, err := a.Open(arg, p9.OREAD)
			if err != nil {
				return util.Errorf("open %q: %w", arg, err)
			}

			err = writeFile(arg, f)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func init() {
	RegisterCommand(&readCmd{})
}
