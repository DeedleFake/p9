package main

import (
	"archive/tar"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/DeedleFake/p9"
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
		return fmt.Errorf("Failed to parse flags: %v", err)
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
			return fmt.Errorf("read %q: %w", arg, err)
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
				return fmt.Errorf("stat %q: %w", arg, err)
			}

			switch {
			case fi.Mode&(p9.ModeAppend|p9.ModeExclusive|p9.ModeMount|p9.ModeAuth) != 0:

			case fi.Mode&p9.ModeDir != 0:
				children, err := f.Readdir()
				if err != nil {
					return fmt.Errorf("read dir %q: %w", arg, err)
				}

				for _, c := range children {
					cf, err := f.Open(c.Name, p9.OREAD)
					if err != nil {
						return fmt.Errorf("open %q: %w", path.Join(arg, c.Name), err)
					}

					err = writeFile(path.Join(arg, c.Name), cf)
					if err != nil {
						return err
					}
				}

			default:
				err = out.WriteHeader(&tar.Header{
					Name:       arg,
					Size:       int64(fi.Length),
					Mode:       int64(fi.Mode.OS()),
					Uname:      fi.UID,
					Gname:      fi.GID,
					ModTime:    fi.MTime,
					AccessTime: fi.ATime,
				})
				if err != nil {
					return fmt.Errorf("write header for %q: %w", arg, err)
				}

				_, err = io.Copy(out, f)
				if err != nil {
					return fmt.Errorf("read %q: %w", arg, err)
				}
			}

			return nil
		}
	}

	return attach(options, func(a *p9.Remote) error {
		for _, arg := range args {
			f, err := a.Open(arg, p9.OREAD)
			if err != nil {
				return fmt.Errorf("open %q: %w", arg, err)
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
