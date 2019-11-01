// +build linux darwin freebsd

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"path"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/internal/util"
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
		c, err := fuse.Mount(
			args[0],
			fuse.FSName("9p!"+options.Address),
			fuse.Subtype("9p"),
			fuse.VolumeName("9p!"+options.Address),
		)
		if err != nil {
			return util.Errorf("mount: %w", err)
		}
		defer c.Close()

		err = fs.Serve(c, &fuseFS{root: a})
		if err != nil {
			return util.Errorf("serve: %w", err)
		}

		<-c.Ready
		if err := c.MountError; err != nil {
			return util.Errorf("ready: %w", err)
		}

		return nil
	})
}

type fuseFS struct {
	root *p9.Remote
}

func (fs *fuseFS) Root() (fs.Node, error) {
	return &fuseNode{n: fs.root}, nil
}

type fuseNode struct {
	n *p9.Remote
	p string
}

func (node *fuseNode) flags(f fuse.OpenFlags) (flags uint8) {
	switch {
	case f.IsReadOnly():
		flags = p9.OREAD
	case f.IsWriteOnly():
		flags = p9.OWRITE
	case f.IsReadWrite():
		flags = p9.ORDWR
	}
	if f&fuse.OpenTruncate != 0 {
		flags |= p9.OTRUNC
	}

	return flags
}

func (node *fuseNode) Attr(ctx context.Context, attr *fuse.Attr) error {
	s, err := node.n.Stat(node.p)
	if err != nil {
		log.Printf("Error statting file: %v", err)
		return err
	}

	attr.Inode = s.Path
	attr.Size = s.Length
	attr.Atime = s.ATime
	attr.Mtime = s.MTime
	attr.Mode = s.FileMode.OS()

	return nil
}

func (node *fuseNode) Lookup(ctx context.Context, name string) (fs.Node, error) {
	p := path.Join(node.p, name)
	_, err := node.n.Stat(p)
	if err != nil {
		return nil, fuse.ENOENT
	}

	return &fuseNode{n: node.n, p: p}, nil
}

func (node *fuseNode) Open(ctx context.Context, req *fuse.OpenRequest, rsp *fuse.OpenResponse) (fs.Handle, error) {
	n, err := node.n.Open(node.p, node.flags(req.Flags))
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return nil, err
	}
	return &fuseNode{n: n}, nil
}

func (node *fuseNode) Create(ctx context.Context, req *fuse.CreateRequest, rsp *fuse.CreateResponse) (fs.Handle, error) {
	n, err := node.n.Create(path.Join(node.p, req.Name), p9.ModeFromOS(req.Mode), node.flags(req.Flags))
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return nil, err
	}
	return &fuseNode{n: n}, nil
}

func (node *fuseNode) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	p := path.Join(node.p, req.Name)

	n, err := node.n.Create(p, p9.ModeFromOS(req.Mode)|p9.ModeDir, 0)
	if err != nil {
		log.Printf("Error creating directory: %v", err)
		return nil, err
	}

	err = n.Close()
	if err != nil {
		log.Printf("Error closing newly-created directory: %v", err)
		return nil, err
	}

	return &fuseNode{n: n, p: p}, nil
}

func (node *fuseNode) direntType(m p9.FileMode) fuse.DirentType {
	switch {
	case m&p9.ModeDir != 0:
		return fuse.DT_Dir
	case m&p9.ModeSymlink != 0:
		return fuse.DT_Link
	case m&p9.ModeSocket != 0:
		return fuse.DT_Socket
	default:
		return fuse.DT_Unknown
	}
}

func (node *fuseNode) Read(ctx context.Context, req *fuse.ReadRequest, rsp *fuse.ReadResponse) error {
	if req.Dir {
		log.Printf("Tried to read file as a directory")
		return fmt.Errorf("%#v", req)
	}

	buf := make([]byte, req.Size)
	n, err := node.n.ReadAt(buf, req.Offset)
	rsp.Data = buf[:n]
	if (err != nil) && !errors.Is(err, io.EOF) {
		log.Printf("Error reading file: %v", err)
		return err
	}
	return nil
}

func (node *fuseNode) Write(ctx context.Context, req *fuse.WriteRequest, rsp *fuse.WriteResponse) error {
	n, err := node.n.WriteAt(req.Data, req.Offset)
	rsp.Size = n
	if err != nil {
		log.Printf("Error writing file: %v", err)
		return err
	}
	return nil
}

func (node *fuseNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	e, err := node.n.Readdir()
	if err != nil {
		log.Printf("Error reading directory: %v", err)
		return nil, err
	}

	r := make([]fuse.Dirent, len(e))
	for i := range e {
		r[i] = fuse.Dirent{
			Inode: e[i].Path,
			Type:  node.direntType(e[i].FileMode),
			Name:  e[i].EntryName,
		}
	}

	return r, nil
}

func (node *fuseNode) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return node.n.Close()
}

func init() {
	RegisterCommand(&mountCmd{})
}
