package p9

import (
	"fmt"
	"io"
	"os"
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

func (h *fsHandler) setFID(fid uint32, path string) {
	h.fids.Store(fid, path)
}

func (h *fsHandler) getQID(path string, t func() (QIDType, bool)) (QID, bool) {
	h.qidM.Lock()
	defer h.qidM.Unlock()

	n, ok := h.qids[path]
	if ok {
		return n, true
	}

	qt, ok := t()
	if !ok {
		return n, false
	}

	n = QID{
		Type: qt,
		Path: h.nextPath,
	}

	h.nextPath++
	h.qids[path] = n

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
		qid, ok := h.getQID(msg.Aname, func() (QIDType, bool) {
			return h.fs.Type(msg.Aname)
		})
		if !ok {
			return &Rerror{
				Ename: os.ErrNotExist.Error(),
			}
		}

		h.setFID(msg.FID, msg.Aname)
		return &Rattach{
			QID: qid,
		}

	default:
		panic(fmt.Errorf("Unexpected message type: %T", msg))
	}
}
