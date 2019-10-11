package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/internal/util"
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
		return err
	}
	defer c.Close()

	_, err = c.Handshake(uint32(options.MSize))
	if err != nil {
		return util.Errorf("handshake: %w", err)
	}

	a, err := c.Attach(nil, options.UName, options.AName)
	if err != nil {
		return util.Errorf("attach %q: %w", options.AName, err)
	}
	defer a.Close()

	return f(a)
}

func parseAddr(addr string) (network, address string) {
	switch {
	case strings.HasPrefix(addr, "$"):
		return getNamespace(addr[1:])

	case strings.HasPrefix(addr, "./"), strings.HasPrefix(addr, "/"):
		return "unix", addr
	}

	parts := strings.SplitN(addr, ":", 2)
	if len(parts) == 2 {
		if (parts[1] == "9p") || (parts[1] == "9fs") {
			parts[1] = StandardPort
		}

		return "tcp", strings.Join(parts, ":")
	}

	parts = strings.SplitN(addr, "!", 3)
	switch len(parts) {
	case 2:
		return parts[0], parts[1] + ":" + StandardPort
	case 3:
		if (parts[2] == "9p") || (parts[2] == "9fs") {
			parts[2] = StandardPort
		}
		return parts[0], strings.Join(parts[1:], ":")
	}

	return "tcp", addr + ":" + StandardPort
}

type GlobalOptions struct {
	Network string
	Address string
	MSize   uint
	UName   string
	AName   string
}

func getUsername() string {
	u, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get username: %v\n", err)
		return ""
	}

	return u.Username
}

func main() {
	var options GlobalOptions
	flag.StringVar(
		&options.Address,
		"addr",
		"",
		"When acting as a server, the address to bind to. When acting as a client, the address to connect to.",
	)
	flag.UintVar(
		&options.MSize,
		"msize",
		2048,
		"The message size to request from the server, or the size to report to a client.",
	)
	flag.StringVar(&options.UName, "uname", getUsername(), "The user name to use for attaching. Default is the current user.")
	flag.StringVar(&options.AName, "aname", "", "The filesystem root to attach to.")
	help := flag.Bool("help", false, "Show this help.")
	flag.Parse()

	options.Network, options.Address = parseAddr(options.Address)

	runCommand := func(c Command) {
		err := c.Run(options, flag.Args())
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
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
