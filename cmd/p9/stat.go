package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/internal/util"
)

type statCmd struct{}

func (cmd *statCmd) Name() string {
	return "stat"
}

func (cmd *statCmd) Desc() string {
	return "Gives detailed information about a file."
}

func (cmd *statCmd) Run(options GlobalOptions, args []string) error {
	fset := flag.NewFlagSet(cmd.Name(), flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(fset.Output(), "%v gives detailed information about a file.\n", args[0])
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Usage: %v [options] [path...]\n", cmd.Name())
		fmt.Fprintf(fset.Output(), "\n")
		fmt.Fprintf(fset.Output(), "Options:\n")
		fset.PrintDefaults()
	}
	format := fset.String("f", "text", "Output format. Supported formats are text and json.")
	err := fset.Parse(args[1:])
	if err != nil {
		return util.Errorf("parse flags: %w", err)
	}

	p := map[string]func(p9.DirEntry){
		"text": cmd.printText,
		"json": cmd.printJSON,
	}[*format]
	if p == nil {
		fmt.Fprintf(fset.Output(), "Unknown format: %q\n", *format)
		fmt.Fprintf(fset.Output(), "\n")
		return flag.ErrHelp
	}

	args = fset.Args()
	if len(args) == 0 {
		args = []string{""}
	}

	return attach(options, func(a *p9.Remote) error {
		switch *format {
		case "json":
			fmt.Println("[")
		}

		for i, arg := range args {
			fi, err := a.Stat(arg)
			if err != nil {
				return util.Errorf("stat %q: %w", arg, err)
			}

			p(fi)

			switch *format {
			case "json":
				c := ""
				if i < len(args)-1 {
					c = ","
				}

				fmt.Println(c)

			default:
				if len(args) > 1 {
					fmt.Println()
				}
			}
		}

		switch *format {
		case "json":
			fmt.Println("]")
		}

		return nil
	})
}

func (cmd *statCmd) printText(fi p9.DirEntry) {
	w := tabwriter.NewWriter(
		os.Stdout,
		0,
		4,
		1,
		' ',
		0,
	)
	defer func() {
		_ = w.Flush()
	}()

	const timeFormat = "03:04 PM, January 2, 2006"

	size := fi.Length
	suffix := "B"
	switch {
	case size >= 1000000000:
		size /= 1000000000
		suffix = "G"
	case size >= 1000000:
		size /= 1000000
		suffix = "M"
	case size >= 1000:
		size /= 1000
		suffix = "K"
	}

	fmt.Fprintf(w, "Mode:\t%v\n", fi.FileMode)
	fmt.Fprintf(w, "Last Accessed:\t%v\n", fi.ATime.Format(timeFormat))
	fmt.Fprintf(w, "Last Modified:\t%v\n", fi.MTime.Format(timeFormat))
	fmt.Fprintf(w, "Size:\t%v%v\n", size, suffix)
	fmt.Fprintf(w, "Name:\t%q\n", fi.EntryName)
	fmt.Fprintf(w, "User:\t%q\n", fi.UID)
	fmt.Fprintf(w, "Group:\t%q\n", fi.GID)
	fmt.Fprintf(w, "Last Modified By:\t%q\n", fi.MUID)
}

func (cmd *statCmd) printJSON(fi p9.DirEntry) {
	buf, _ := json.MarshalIndent(fi, "  ", "  ")
	fmt.Fprintf(os.Stdout, "  %s", buf)
}

func init() {
	RegisterCommand(&statCmd{})
}
