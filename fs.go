package p9

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"
)

type FileSystem interface {
	Type(string) (QIDType, bool)

	Stat(string) (DirEntry, error)
	Open(string, uint8) (File, error)
}

type File interface {
	io.ReaderAt
	io.WriterAt
	io.Closer

	Type() QIDType
	Readdir() ([]DirEntry, error)
}

type DirEntry struct {
	Type   QIDType
	Mode   uint32
	ATime  time.Time
	MTime  time.Time
	Length uint64
	Name   string
	UID    string
	GID    string
	MUID   string
}

func (d DirEntry) stat(path uint64) Stat {
	return Stat{
		QID: QID{
			Type: d.Type,
			Path: path,
		},
		Mode:   d.Mode | (uint32(d.Type) << 24),
		ATime:  d.ATime,
		MTime:  d.MTime,
		Length: d.Length,
		Name:   d.Name,
		UID:    d.UID,
		GID:    d.GID,
		MUID:   d.MUID,
	}
}

type fsHandler struct {
	fs    FileSystem
	msize uint32

	fids sync.Map

	qidM     sync.Mutex
	nextPath uint64
	qids     map[string]QID

	files sync.Map
	dirs  sync.Map
}

func HandleFS(fs FileSystem, msize uint32) MessageHandler {
	return &fsHandler{
		fs:    fs,
		msize: msize,

		qids: make(map[string]QID),
	}
}

func (h *fsHandler) setPath(fid uint32, p string) {
	h.fids.Store(fid, p)
}

func (h *fsHandler) getPath(fid uint32) (string, bool) {
	v, ok := h.fids.Load(fid)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (h *fsHandler) getQID(p string) (QID, bool) {
	h.qidM.Lock()
	defer h.qidM.Unlock()

	n, ok := h.qids[p]
	if ok {
		return n, true
	}

	qt, ok := h.fs.Type(p)
	if !ok {
		return n, false
	}

	n = QID{
		Type: qt,
		Path: h.nextPath,
	}

	h.nextPath++
	h.qids[p] = n

	return n, true
}

func (h *fsHandler) setFile(fid uint32, file File) {
	h.files.Store(fid, file)
}

func (h *fsHandler) getFile(fid uint32) (File, bool) {
	v, ok := h.files.Load(fid)
	if !ok {
		return nil, false
	}
	return v.(File), true
}

func (h *fsHandler) setDir(fid uint32, entries []DirEntry) (io.Reader, error) {
	base, ok := h.getPath(fid)
	if !ok {
		return nil, fmt.Errorf("Unknown FID: %v", fid)
	}

	var buf bytes.Buffer
	e := &encoder{
		w: &buf,
	}
	e.mode = e.write

	for _, entry := range entries {
		qid, ok := h.getQID(path.Join(base, entry.Name))
		if !ok {
			return nil, os.ErrNotExist
		}

		e.Encode(entry.stat(qid.Path))
	}

	h.dirs.Store(fid, &buf)
	return &buf, nil
}

func (h *fsHandler) getDir(fid uint32) (io.Reader, bool) {
	v, ok := h.dirs.Load(fid)
	if !ok {
		return nil, false
	}
	return v.(io.Reader), true
}

func (h *fsHandler) largeCount(count uint32) bool {
	return 4+1+2+4+count > h.msize
}

func (h *fsHandler) version(msg *Tversion) Message {
	if h.msize > msg.Msize {
		h.msize = msg.Msize
	}

	return &Rversion{
		Msize:   h.msize,
		Version: "9P2000",
	}
}

func (h *fsHandler) attach(msg *Tattach) Message {
	name := path.Clean(msg.Aname)

	qid, ok := h.getQID(name)
	if !ok {
		return &Rerror{
			Ename: os.ErrNotExist.Error(),
		}
	}

	h.setPath(msg.FID, name)
	return &Rattach{
		QID: qid,
	}
}

func (h *fsHandler) walk(msg *Twalk) Message {
	base, ok := h.getPath(msg.FID)
	if !ok {
		return &Rerror{
			Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
		}
	}

	qids := make([]QID, 0, len(msg.Wname))
	for i, name := range msg.Wname {
		next := path.Join(base, name)

		qid, ok := h.getQID(next)
		if !ok {
			if i == 0 {
				return &Rerror{
					Ename: os.ErrNotExist.Error(),
				}
			}

			return &Rwalk{
				WQID: qids,
			}
		}

		qids = append(qids, qid)
		base = next
	}

	h.setPath(msg.NewFID, base)
	return &Rwalk{
		WQID: qids,
	}
}

func (h *fsHandler) open(msg *Topen) Message {
	if _, ok := h.getFile(msg.FID); ok {
		return &Rerror{
			Ename: "file already open",
		}
	}

	p, ok := h.getPath(msg.FID)
	if !ok {
		return &Rerror{
			Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
		}
	}

	file, err := h.fs.Open(p, msg.Mode)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	qid, ok := h.getQID(p)
	if !ok {
		// If everything else works, this should never happen.
		return &Rerror{
			Ename: "file opened but QID not found",
		}
	}

	h.setFile(msg.FID, file)
	return &Ropen{
		QID: qid,

		// What is IOUnit for?
	}
}

func (h *fsHandler) read(msg *Tread) Message {
	file, ok := h.getFile(msg.FID)
	if !ok {
		return &Rerror{
			Ename: "file not open",
		}
	}

	if h.largeCount(msg.Count) {
		return &Rerror{
			Ename: "read too large",
		}
	}

	var n int
	buf := make([]byte, msg.Count)

	switch {
	case file.Type()&QTDir != 0:
		var r io.Reader
		switch msg.Offset {
		case 0:
			dir, err := file.Readdir()
			if err != nil {
				return &Rerror{
					Ename: err.Error(),
				}
			}

			r, err = h.setDir(msg.FID, dir)
			if err != nil {
				return &Rerror{
					Ename: err.Error(),
				}
			}

		default:
			r, ok = h.getDir(msg.FID)
			if !ok {
				return &Rerror{
					Ename: "Dir read with invalid offset",
				}
			}
		}

		tmp, err := r.Read(buf)
		if err != nil {
			return &Rerror{
				Ename: err.Error(),
			}
		}
		n = tmp

	default:
		tmp, err := file.ReadAt(buf, int64(msg.Offset))
		if (err != nil) && (err != io.EOF) {
			return &Rerror{
				Ename: err.Error(),
			}
		}
		n = tmp
	}

	return &Rread{
		Data: buf[:n],
	}
}

func (h *fsHandler) stat(msg *Tstat) Message {
	p, ok := h.getPath(msg.FID)
	if !ok {
		return &Rerror{
			Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
		}
	}

	stat, err := h.fs.Stat(p)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	qid, ok := h.getQID(p)
	if !ok {
		return &Rerror{
			Ename: os.ErrNotExist.Error(),
		}
	}

	return &Rstat{
		Stat: stat.stat(qid.Path),
	}
}

func (h *fsHandler) HandleMessage(msg Message) Message {
	fmt.Printf("%#v\n", msg)

	switch msg := msg.(type) {
	case *Tversion:
		return h.version(msg)

	case *Tattach:
		return h.attach(msg)

	case *Twalk:
		return h.walk(msg)

	case *Topen:
		return h.open(msg)

	case *Tread:
		return h.read(msg)

	case *Tstat:
		return h.stat(msg)

	default:
		return &Rerror{
			Ename: fmt.Sprintf("Unexpected message type: %T", msg),
		}
	}
}
