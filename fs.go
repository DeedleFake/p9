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
	Open(string) (File, error)
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
}

func HandleFS(fs FileSystem, msize uint32) MessageHandler {
	return &fsHandler{
		fs:    fs,
		msize: msize,

		qids: make(map[string]QID),
	}
}

func (h *fsHandler) setFID(fid uint32, p string) {
	h.fids.Store(fid, p)
}

func (h *fsHandler) getFID(fid uint32) (string, bool) {
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

		h.setFID(msg.FID, name)
		return &Rattach{
			QID: qid,
		}

	case *Twalk:
		base, ok := h.getFID(msg.FID)
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

		h.setFID(msg.NewFID, base)
		return &Rwalk{
			WQID: qids,
		}

	default:
		panic(fmt.Errorf("Unexpected message type: %T", msg))
	}
}
