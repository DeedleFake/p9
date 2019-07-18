package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/DeedleFake/p9"
)

const StandardPort = "564"

type Command interface {
	Name() string
	Desc() string
	Run(GlobalOptions, []string) error
}

var commands = []Command{
	&helpCmd{},
	&versionCmd{},
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

func (helpCmd) Run(options GlobalOptions, args []string) (err error) {
	if len(args) > 1 {
		cmd := GetCommand(args[1])
		if cmd != nil {
			return cmd.Run(options, []string{args[1], "--help"})
		}

		fmt.Fprintf(os.Stderr, "Unknown help topic: %q\n", args[1])
		fmt.Fprintf(os.Stderr, "\n")
		err = flag.ErrHelp
	}

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

	return err
}

type versionCmd struct{}

func (versionCmd) Name() string {
	return "version"
}

func (versionCmd) Desc() string {
	return "Prints version information."
}

func (versionCmd) Run(options GlobalOptions, args []string) error {
	v := "failed to read build info"
	if bi, ok := debug.ReadBuildInfo(); ok {
		v = bi.Main.Version
	}

	fmt.Printf("Go: %v\n", runtime.Version())
	fmt.Printf("%v: %v\n", filepath.Base(os.Args[0]), v)
	fmt.Printf("9P: %v\n", p9.Version)

	return nil
}

func attach(options GlobalOptions, f func(*p9.Remote) error) error {
	c, err := p9.Dial(options.Network, options.Address)
	if err != nil {
		return fmt.Errorf("Failed to dial address: %v", err)
	}
	defer c.Close()

	_, err = c.Handshake(uint32(options.MSize))
	if err != nil {
		return fmt.Errorf("Handshake failed: %v", err)
	}

	a, err := c.Attach(nil, options.UName, options.AName)
	if err != nil {
		return fmt.Errorf("Failed to attach: %v", err)
	}
	defer a.Close()

	return f(a)
}

func parseAddr(addr string) (network, host, port string) {
	defer func() {
		if (len(host) > 0) && (host[0] == '$') {
			network, host, port = getNamespaceHost(network, host, port)
		}

		switch port {
		case "9fs", "9p":
			port = StandardPort
		}
	}()

	parts := strings.SplitN(addr, "!", 3)
	switch len(parts) {
	case 0:
		return "tcp", "localhost", StandardPort

	case 1:
		port = StandardPort
		sub := strings.SplitN(parts[0], ":", 2)
		host = sub[0]
		if len(sub) == 2 {
			port = sub[1]
		}
		return "tcp", host, port

	case 2:
		return parts[0], parts[1], ""

	case 3:
		return parts[0], parts[1], parts[2]
	}

	panic("This should never be reached.")
}

type GlobalOptions struct {
	Network string
	Address string
	MSize   uint
	UName   string
	AName   string
}

func main() {
	var options GlobalOptions
	flag.StringVar(
		&options.Address,
		"addr",
		"localhost:"+StandardPort,
		"When acting as a server, the address to bind to. When acting as a client, the address to connect to.",
	)
	flag.UintVar(
		&options.MSize,
		"msize",
		2048,
		"The message size to request from the server, or the size to report to a client.",
	)
	flag.StringVar(&options.UName, "uname", "root", "The user name to use for attaching.")
	flag.StringVar(&options.AName, "aname", "", "The filesystem root to attach to.")
	help := flag.Bool("help", false, "Show this help.")
	flag.Parse()

	n, a, p := parseAddr(options.Address)
	if (p == "") && strings.HasPrefix(a, "tcp") {
		p = StandardPort
	}
	options.Network = n
	options.Address = a
	if p != "" {
		options.Address += ":" + p
	}

	runCommand := func(c Command) {
		err := c.Run(options, flag.Args())
		if err != nil {
			if err == flag.ErrHelp {
				os.Exit(2)
			}

			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	cmd := flag.Arg(0)
	if (cmd == "") || (*help) {
		cmd = "help"
	}

	c := GetCommand(cmd)
	if c == nil {
		fmt.Fprintf(os.Stderr, "No such command: %q\n", cmd)
		runCommand(GetCommand("help"))
		os.Exit(2)
	}

	runCommand(c)
}
