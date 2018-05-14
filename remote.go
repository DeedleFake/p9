package p9

import (
	"errors"
	"io"
	"path"
	"strings"
)

var (
	// ErrUnsupportedVersion is returned from a handshake attempt that
	// fails due to a version mismatch.
	ErrUnsupportedVersion = errors.New("unsupported version")
)

// Handshake performs an initial handshake to establish the maximum
// allowed message size. A handshake must be performed before any
// other request types may be sent.
func (c *Client) Handshake(msize uint32) (uint32, error) {
	rsp, err := c.Send(&Tversion{
		Msize:   msize,
		Version: "9P2000",
	})
	if err != nil {
		return 0, err
	}

	version := rsp.(*Rversion)
	if version.Version != "9P2000" {
		return 0, ErrUnsupportedVersion
	}

	return version.Msize, nil
}

// Remote provides a file-like interface for performing operations on
// files presented by a 9P server.
type Remote struct {
	client *Client

	fid uint32
	qid QID

	pos uint64
}

func (c *Client) Auth(user, aname string) (*Remote, error) {
	fid := <-c.nextFID

	rsp, err := c.Send(&Tauth{
		AFID:  fid,
		Uname: user,
		Aname: aname,
	})
	if err != nil {
		return nil, err
	}
	rauth := rsp.(*Rauth)

	return &Remote{
		client: c,
		fid:    fid,
		qid:    rauth.AQID,
	}, nil
}

func (c *Client) Attach(afile *Remote, user, aname string) (*Remote, error) {
	fid := <-c.nextFID

	var afid uint32
	if afile != nil {
		afid = afile.fid
	}

	rsp, err := c.Send(&Tattach{
		FID:   fid,
		AFID:  afid,
		Uname: user,
		Aname: aname,
	})
	if err != nil {
		return nil, err
	}
	attach := rsp.(*Rattach)

	return &Remote{
		client: c,
		fid:    fid,
		qid:    attach.QID,
	}, nil
}

func (file *Remote) Type() QIDType {
	return file.qid.Type
}

func (file *Remote) walk(p string) (*Remote, error) {
	fid := <-file.client.nextFID

	w := strings.Split(path.Clean(p), "/")
	_, err := file.client.Send(&Twalk{
		FID:    file.fid,
		NewFID: fid,
		Wname:  w,
	})
	if err != nil {
		return nil, err
	}

	return &Remote{
		client: file.client,
		fid:    fid,
	}, nil
}

func (file *Remote) Open(p string, mode uint8) (*Remote, error) {
	next, err := file.walk(p)
	if err != nil {
		return nil, err
	}

	rsp, err := file.client.Send(&Topen{
		FID:  next.fid,
		Mode: mode,
	})
	if err != nil {
		return nil, err
	}
	open := rsp.(*Ropen)

	file.qid = open.QID

	return file, nil
}

func (file *Remote) Read(buf []byte) (int, error) {
	n, err := file.ReadAt(buf, int64(file.pos))
	file.pos += uint64(n)
	return n, err
}

func (file *Remote) ReadAt(buf []byte, off int64) (int, error) {
	rsp, err := file.client.Send(&Tread{
		FID:    file.fid,
		Offset: uint64(off),
		Count:  uint32(len(buf)),
	})
	if err != nil {
		return 0, err
	}
	read := rsp.(*Rread)

	n := copy(buf, read.Data)
	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func (file *Remote) Close() error {
	_, err := file.client.Send(&Tclunk{
		FID: file.fid,
	})
	return err
}
