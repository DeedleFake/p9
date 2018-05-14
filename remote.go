package p9

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
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
//
// Remote implements File, allowing it to be itself served using
// FileSystem.
type Remote struct {
	client *Client

	fid uint32
	qid QID

	m   sync.Mutex
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

	w := []string{path.Clean(p)}
	if w[0] != "/" {
		w = strings.Split(w[0], "/")
	}
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

	next.qid = open.QID

	return next, nil
}

func (file *Remote) Seek(offset int64, whence int) (int64, error) {
	file.m.Lock()
	defer file.m.Unlock()

	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return int64(file.pos), errors.New("negative offset")
		}

		file.pos = uint64(offset)
		return offset, nil

	case io.SeekCurrent:
		npos := int64(file.pos) + offset
		if npos < 0 {
			return int64(file.pos), errors.New("negative offset")
		}

		file.pos = uint64(npos)
		return npos, nil

	case io.SeekEnd:
		stat, err := file.Stat()
		if err != nil {
			return int64(file.pos), err
		}

		npos := int64(stat.Length) + offset
		if npos < 0 {
			return int64(file.pos), errors.New("negative offset")
		}

		file.pos = uint64(npos)
		return npos, nil
	}

	panic(fmt.Errorf("Invalid whence: %v", whence))
}

func (file *Remote) Read(buf []byte) (int, error) {
	file.m.Lock()
	defer file.m.Unlock()

	n, err := file.ReadAt(buf, int64(file.pos))
	file.pos += uint64(n)
	return n, err
}

func (file *Remote) maxBufSize() int {
	file.client.m.RLock()
	defer file.client.m.RUnlock()

	return int(file.client.msize - uint32(4+1+2+4))
}

func (file *Remote) readPart(buf []byte, off int64) (int, error) {
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
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

func (file *Remote) ReadAt(buf []byte, off int64) (int, error) {
	size := len(buf)
	if size > file.maxBufSize() {
		size = file.maxBufSize()
	}

	var total int
	for start := 0; start < len(buf); start += size {
		end := start + size
		if end > len(buf) {
			end = len(buf)
		}

		n, err := file.readPart(buf[start:end], off+int64(start))
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (file *Remote) Write(data []byte) (int, error) {
	file.m.Lock()
	defer file.m.Unlock()

	n, err := file.WriteAt(data, int64(file.pos))
	file.pos += uint64(n)
	return n, err
}

func (file *Remote) writePart(data []byte, off int64) (int, error) {
	rsp, err := file.client.Send(&Twrite{
		FID:    file.fid,
		Offset: uint64(off),
		Data:   data,
	})
	if err != nil {
		return 0, err
	}
	write := rsp.(*Rwrite)

	if write.Count < uint32(len(data)) {
		return int(write.Count), io.EOF
	}
	return int(write.Count), nil
}

func (file *Remote) WriteAt(data []byte, off int64) (int, error) {
	size := len(data)
	if size > file.maxBufSize() {
		size = file.maxBufSize()
	}

	var total int
	for start := 0; start < len(data); start += size {
		end := start + size
		if end > len(data) {
			end = len(data)
		}

		n, err := file.writePart(data[start:end], off+int64(start))
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (file *Remote) Close() error {
	_, err := file.client.Send(&Tclunk{
		FID: file.fid,
	})
	return err
}

func (file *Remote) Stat() (DirEntry, error) {
	rsp, err := file.client.Send(&Tstat{
		FID: file.fid,
	})
	if err != nil {
		return DirEntry{}, err
	}
	stat := rsp.(*Rstat)

	return stat.Stat.dirEntry(), nil
}

func (file *Remote) Readdir() ([]DirEntry, error) {
	d := &decoder{
		r: file,
	}

	var entries []DirEntry
	for {
		var stat Stat
		d.Decode(&stat)
		if d.err != nil {
			if d.err == io.EOF {
				d.err = nil
			}
			return entries, d.err
		}

		entries = append(entries, stat.dirEntry())
	}
}
