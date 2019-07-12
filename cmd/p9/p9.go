package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Command interface {
	Name() string
	Desc() string
	Run(GlobalOptions, []string)
}

var commands = []Command{
	&helpCmd{},
}

func GetCommand(name string) Command {
	for _, cmd := range commands {
		if cmd.Name() == name {
			return cmd
		}
	}

	return nil
}

func RegisterCommand(c Command) {
	commands = append(commands, c)
}

type helpCmd struct{}

func (helpCmd) Name() string {
	return "help"
}

func (helpCmd) Desc() string {
	return "Displays this help message."
}

func (helpCmd) Run(options GlobalOptions, args []string) {
	arg0 := filepath.Base(os.Args[0])

	fmt.Fprintf(os.Stderr, "%v is a command-line tool for both accessing and serving 9P filesystems.\n", arg0)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Usage: %v [global options] <command> [command options]\n", arg0)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Global Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "\t%v\t\t%v\n", cmd.Name(), cmd.Desc())
	}
}

type GlobalOptions struct {
	Address string
	MSize   uint
	UName   string
	AName   string
}

func main() {
	var options GlobalOptions
	flag.StringVar(&options.Address, "addr", "localhost:564", "When acting as a server, the address to bind to. When acting as a client, the address to connect to.")
	flag.UintVar(&options.MSize, "msize", 2048, "The message size to request from the server, or the size to report to a client.")
	flag.StringVar(&options.UName, "uname", "root", "The user name to use for attaching.")
	flag.StringVar(&options.AName, "aname", "", "The filesystem root to attach to.")
	help := flag.Bool("help", false, "Show this help.")
	flag.Parse()

	if parts := strings.Split(options.Address, ":"); len(parts) == 1 {
		options.Address += ":564"
	}

	cmd := flag.Arg(0)
	if (cmd == "") || (*help) {
		cmd = "help"
	}

	c := GetCommand(cmd)
	if c == nil {
		fmt.Fprintf(os.Stderr, "No such command: %q", cmd)
		GetCommand("help").Run(options, flag.Args())
		os.Exit(2)
	}

	c.Run(options, flag.Args())
}
