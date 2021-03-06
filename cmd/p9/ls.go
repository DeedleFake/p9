package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/internal/util"
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
	fset := flag.NewFlagSet(cmd.Name(), flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(fset.Output(), "%v lists the files in a directory.\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Usage: %v [-l] [path...]\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Options:\n")
		fset.PrintDefaults()
	}
	fset.BoolVar(&cmd.showDetails, "l", false, "Show details.")
	err := fset.Parse(args[1:])
	if err != nil {
		return util.Errorf("parse flags: %w", err)
	}

	args = fset.Args()
	if len(args) == 0 {
		args = []string{""}
	}

	return attach(options, func(a *p9.Remote) error {
		for i, arg := range args {
			if len(args) > 1 {
				fmt.Printf("%v:\n", arg)
			}

			d, err := a.Open(arg, p9.OREAD)
			if err != nil {
				return util.Errorf("open %q: %w", arg, err)
			}
			defer d.Close()

			fi, err := d.Stat("")
			if err != nil {
				return util.Errorf("stat %q: %w", arg, err)
			}

			if !fi.IsDir() {
				cmd.printEntries([]p9.DirEntry{fi})
				return nil
			}

			entries, err := d.Readdir()
			if err != nil {
				return util.Errorf("read dir %q: %w", arg, err)
			}
			sort.Slice(entries, func(i1, i2 int) bool {
				return entries[i1].EntryName < entries[i2].EntryName
			})

			fi.EntryName = "."
			cmd.printEntries(append([]p9.DirEntry{fi}, entries...))

			if i < len(args)-1 {
				fmt.Println()
			}
		}

		return nil
	})
}

func (cmd *lsCmd) printEntries(entries []p9.DirEntry) {
	w := tabwriter.NewWriter(
		os.Stdout,
		0,
		4,
		1,
		' ',
		0,
	)
	defer w.Flush()

	for _, entry := range entries {
		if cmd.showDetails {
			yd := "15:04"
			if entry.MTime.Year() != time.Now().Year() {
				yd = "2006"
			}

			fmt.Fprintf(
				w,
				"%v\t%v\t%v\t%v\t%v\t",
				entry.FileMode,
				entry.UID,
				entry.GID,
				entry.Length, // TODO: Right-align this column.
				entry.MTime.Format("Jan 02 "+yd),
			)
		}
		fmt.Fprintf(w, "%v\n", entry.EntryName)
	}
}

func init() {
	RegisterCommand(&lsCmd{})
}
