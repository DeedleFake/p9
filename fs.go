package p9

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"
)

type FileSystem interface {
	Type(string) (QIDType, bool)
	Open(string, uint8) (File, error)
}

type File interface {
	io.ReaderAt
	io.WriterAt
	io.Closer

	Readdir() ([]Stat, error)
}

type fsHandler struct {
	fs    FileSystem
	msize uint32

	fids sync.Map

	qidM     sync.Mutex
	nextPath uint64
	qids     map[string]QID

	files sync.Map
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

func (h *fsHandler) getQID(p string, t func(string) (QIDType, bool)) (QID, bool) {
	h.qidM.Lock()
	defer h.qidM.Unlock()

	n, ok := h.qids[p]
	if ok {
		return n, true
	}

	qt, ok := t(p)
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

func (h *fsHandler) HandleMessage(msg Message) Message {
	fmt.Printf("%#v\n", msg)

	switch msg := msg.(type) {
	case *Tversion:
		if h.msize > msg.Msize {
			h.msize = msg.Msize
		}

		return &Rversion{
			Msize:   h.msize,
			Version: "9P2000",
		}

	case *Tattach:
		name := path.Clean(msg.Aname)

		qid, ok := h.getQID(name, h.fs.Type)
		if !ok {
			return &Rerror{
				Ename: os.ErrNotExist.Error(),
			}
		}

		h.setPath(msg.FID, name)
		return &Rattach{
			QID: qid,
		}

	case *Twalk:
		base, ok := h.getPath(msg.FID)
		if !ok {
			return &Rerror{
				Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
			}
		}

		qids := make([]QID, 0, len(msg.Wname))
		for i, name := range msg.Wname {
			next := path.Join(base, name)

			qid, ok := h.getQID(next, h.fs.Type)
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

	case *Topen:
		if _, ok := h.getFile(msg.FID); ok {
			return &Rerror{
				Ename: "file already open",
			}
		}

		p, ok := h.getPath(msg.FID)
		if !ok {
			return &Rerror{
				Ename: os.ErrNotExist.Error(),
			}
		}

		file, err := h.fs.Open(p, msg.Mode)
		if err != nil {
			return &Rerror{
				Ename: err.Error(),
			}
		}

		qid, ok := h.getQID(p, h.fs.Type)
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

	default:
		return &Rerror{
			Ename: fmt.Sprintf("Unexpected message type: %T", msg),
		}
	}
}
