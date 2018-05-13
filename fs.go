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

// FileSystem is an interface that allows high-level implementations
// of 9P servers by allowing the implementation to ignore the majority
// of the details of the protocol.
//
// With one exception, all paths passed to the methods of this system
// are absolute paths, use forward slashes as separators, and have
// been cleaned using path.Clean(). For the exception, see Auth().
type FileSystem interface {
	// Stat returns a DirEntry giving info about the file or directory
	// at the given path. If an error is returned, the text of the error
	// will be transmitted to the client.
	Stat(path string) (DirEntry, error)

	// WriteStat applies changes to the metadata of the file at path.
	// The changes argument contains a map containing key-value pairs
	// that correspond to the fields of the DirEntry struct. Any fields
	// that are in the struct but missing from the given map are fields
	// that should not be changed.
	//
	// If an error is returned, it will be transmitted to the client.
	WriteStat(path string, changes map[string]interface{}) error

	// Auth returns an authentication file. This file can be used to
	// send authentication information back and forth between the server
	// and the client.
	//
	// Because of the way that FileSystem hides protocol details, such
	// as FIDs, further calls assume that the Auth command created a
	// file with the same name as the user. References to this file are
	// the only references to a file that are not an absolute path.
	Auth(user, aname string) (File, error)

	// Open opens the file at path in the given mode. If an error is
	// returned, it will be transmitted to the client.
	Open(path string, mode uint8) (File, error)

	// Create creates and opens a file at path with the given perms and
	// mode. If an error is returned, it will be transmitted to the
	// client.
	Create(path string, perm uint32, mode uint8) (File, error)

	// Remove deletes the file at path, returning any errors encountered.
	Remove(path string) error
}

// IOUnitFS is implemented by FileSystems that want to report an
// IOUnit value to clients when open and create requests are made. An
// IOUnit value lets the client know the maximum amount of data during
// reads and writes that is guarunteed to be an atomic operation.
type IOUnitFS interface {
	IOUnit() uint32
}

// File is the interface implemented by files being dealt with by a
// FileSystem.
//
// Note that although this interface implements io.ReaderAt and
// io.WriterAt, a number of the restrictions placed on those
// interfaces may be ignored. The implementation need only follow
// restrictions placed by the 9P protocol specification.
type File interface {
	// Used to handle 9P read requests.
	io.ReaderAt

	// Used to handle 9P write requests.
	io.WriterAt

	// Used to handle 9P clunk requests.
	io.Closer

	// Stat returns the file's corresponding DirEntry.
	Stat() DirEntry

	// Readdir is called when an attempt is made to read a directory. It
	// should return a list of entries in the directory or an error. If
	// an error is returned, the error will be transmitted to the
	// client.
	Readdir() ([]DirEntry, error)
}

// DirEntry is a smaller version of Stat that eliminates unecessary or
// duplicate fields.
//
// Note that the top 8-bits of the Mode field are overwritten during
// transmission using the Type field.
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

// HandleFS returns a MessageHandler that provides a virtual
// filesystem using the provided FileSystem implementation. msize is
// the maximum size that messages from either the server or the client
// are allowed to be.
//
// BUG: Tflush requests are not currently handled at all by this
// implementation due to no clear method of stopping a pending call to
// ReadAt() or WriteAt().
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

func (h *fsHandler) getQID(p string) (QID, error) {
	h.qidM.Lock()
	defer h.qidM.Unlock()

	n, ok := h.qids[p]
	if ok {
		return n, nil
	}

	qt := QTAuth
	if path.IsAbs(p) {
		stat, err := h.fs.Stat(p)
		if !ok {
			return n, err
		}
		qt = stat.Type
	}

	n = QID{
		Type: qt,
		Path: h.nextPath,
	}

	h.nextPath++
	h.qids[p] = n

	return n, nil
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
		qid, err := h.getQID(path.Join(base, entry.Name))
		if err != nil {
			return nil, err
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
	if msg.Version != "9P2000" {
		return &Rerror{
			Ename: "Unsupported version",
		}
	}

	if h.msize > msg.Msize {
		h.msize = msg.Msize
	}

	return &Rversion{
		Msize:   h.msize,
		Version: "9P2000",
	}
}

func (h *fsHandler) auth(msg *Tauth) Message {
	aname := path.Clean(msg.Aname)
	if aname == "." {
		aname = "/"
	}

	file, err := h.fs.Auth(msg.Uname, aname)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	if path.IsAbs(msg.Uname) {
		return &Rerror{
			Ename: "Invalid uname",
		}
	}

	qid, err := h.getQID(msg.Uname)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	h.setFile(msg.AFID, file)

	return &Rauth{
		AQID: qid,
	}
}

func (h *fsHandler) flush(msg *Tflush) Message {
	// TODO: Implement this.

	fmt.Fprintln(os.Stderr, "Warning: Flush support is not implemented.")
	return &Rerror{
		Ename: "Tflush is not supported",
	}
}

func (h *fsHandler) attach(msg *Tattach) Message {
	name := path.Clean(msg.Aname)
	if name == "." {
		name = "/"
	}

	qid, err := h.getQID(name)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
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

		qid, err := h.getQID(next)
		if err != nil {
			if i == 0 {
				return &Rerror{
					Ename: err.Error(),
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

	qid, err := h.getQID(p)
	if err != nil {
		// If everything else works, this should never happen.
		return &Rerror{
			Ename: fmt.Sprintf("File opened but QID not found: %v", err),
		}
	}

	var iounit uint32
	if unit, ok := h.fs.(IOUnitFS); ok {
		iounit = unit.IOUnit()
	}

	h.setFile(msg.FID, file)
	return &Ropen{
		QID:    qid,
		IOUnit: iounit,
	}
}

func (h *fsHandler) create(msg *Tcreate) Message {
	if _, ok := h.getFile(msg.FID); ok {
		return &Rerror{
			Ename: "file already open",
		}
	}

	base, ok := h.getPath(msg.FID)
	if !ok {
		return &Rerror{
			Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
		}
	}
	p := path.Join(base, msg.Name)

	file, err := h.fs.Create(p, msg.Perm, msg.Mode)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	qid, err := h.getQID(p)
	if err != nil {
		return &Rerror{
			Ename: fmt.Sprintf("File created but QID not found: %v", err),
		}
	}

	var iounit uint32
	if unit, ok := h.fs.(IOUnitFS); ok {
		iounit = unit.IOUnit()
	}

	h.setFile(msg.FID, file)
	return &Rcreate{
		QID:    qid,
		IOUnit: iounit,
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
	case file.Stat().Type&QTDir != 0:
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

func (h *fsHandler) write(msg *Twrite) Message {
	file, ok := h.getFile(msg.FID)
	if !ok {
		return &Rerror{
			Ename: "file not open",
		}
	}

	n, err := file.WriteAt(msg.Data, int64(msg.Offset))
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	return &Rwrite{
		Count: uint32(n),
	}
}

func (h *fsHandler) clunk(msg *Tclunk) Message {
	rsp := Message(new(Rclunk))

	file, ok := h.getFile(msg.FID)
	switch ok {
	case true:
		err := file.Close()
		if err != nil {
			rsp = &Rerror{
				Ename: err.Error(),
			}
		}
	case false:
		rsp = &Rerror{
			Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
		}
	}

	h.fids.Delete(msg.FID)
	h.files.Delete(msg.FID)
	h.dirs.Delete(msg.FID)

	return rsp
}

func (h *fsHandler) remove(msg *Tremove) Message {
	p, ok := h.getPath(msg.FID)
	if !ok {
		return &Rerror{
			Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
		}
	}

	err := h.fs.Remove(p)
	h.clunk(&Tclunk{
		FID: msg.FID,
	})
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	return new(Rremove)
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

	qid, err := h.getQID(p)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	return &Rstat{
		Stat: stat.stat(qid.Path),
	}
}

func (h *fsHandler) wstat(msg *Twstat) Message {
	p, ok := h.getPath(msg.FID)
	if !ok {
		return &Rerror{
			Ename: fmt.Sprintf("Unknown FID: %v", msg.FID),
		}
	}

	changes := make(map[string]interface{})

	if msg.Stat.Mode != 0xFFFFFFFF {
		changes["Mode"] = msg.Stat.Mode
	}

	if msg.Stat.ATime.Unix() != -1 {
		changes["ATime"] = msg.Stat.ATime
	}

	if msg.Stat.MTime.Unix() != -1 {
		changes["MTime"] = msg.Stat.MTime
	}

	if msg.Stat.Length != 0xFFFFFFFFFFFFFFFF {
		changes["Length"] = msg.Stat.Length
	}

	if msg.Stat.Name != "" {
		changes["Name"] = msg.Stat.Name
	}

	if msg.Stat.UID != "" {
		changes["UID"] = msg.Stat.UID
	}

	if msg.Stat.GID != "" {
		changes["GID"] = msg.Stat.GID
	}

	if msg.Stat.MUID != "" {
		changes["MUID"] = msg.Stat.MUID
	}

	err := h.fs.WriteStat(p, changes)
	if err != nil {
		return &Rerror{
			Ename: err.Error(),
		}
	}

	return new(Rwstat)
}

func (h *fsHandler) HandleMessage(msg Message) Message {
	switch msg := msg.(type) {
	case *Tversion:
		return h.version(msg)

	case *Tauth:
		return h.auth(msg)

	case *Tflush:
		return h.flush(msg)

	case *Tattach:
		return h.attach(msg)

	case *Twalk:
		return h.walk(msg)

	case *Topen:
		return h.open(msg)

	case *Tcreate:
		return h.create(msg)

	case *Tread:
		return h.read(msg)

	case *Twrite:
		return h.write(msg)

	case *Tclunk:
		return h.clunk(msg)

	case *Tremove:
		return h.remove(msg)

	case *Tstat:
		return h.stat(msg)

	case *Twstat:
		return h.wstat(msg)

	default:
		return &Rerror{
			Ename: fmt.Sprintf("Unexpected message type: %T", msg),
		}
	}
}

func (h *fsHandler) Close() error {
	h.files.Range(func(k, v interface{}) bool {
		v.(File).Close()
		return true
	})

	return nil
}
