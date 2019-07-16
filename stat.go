package p9

import (
	"os"
	"time"
	"unsafe"
)

type FileMode uint32

const (
	ModeDir FileMode = 1 << (31 - iota)
	ModeAppend
	ModeExclusive
	ModeMount
	ModeAuth
	ModeTemporary
	ModeSymlink
	_
	ModeDevice
	ModeNamedPipe
	ModeSocket
	ModeSetuid
	ModeSetgid
)

func ModeFromOS(m os.FileMode) FileMode {
	r := FileMode(m.Perm())

	if m&os.ModeDir != 0 {
		r |= ModeDir
	}
	if m&os.ModeAppend != 0 {
		r |= ModeAppend
	}
	if m&os.ModeExclusive != 0 {
		r |= ModeExclusive
	}
	if m&os.ModeTemporary != 0 {
		r |= ModeTemporary
	}
	if m&os.ModeSymlink != 0 {
		r |= ModeSymlink
	}
	if m&os.ModeDevice != 0 {
		r |= ModeDevice
	}
	if m&os.ModeNamedPipe != 0 {
		r |= ModeNamedPipe
	}
	if m&os.ModeSocket != 0 {
		r |= ModeSocket
	}
	if m&os.ModeSetuid != 0 {
		r |= ModeSetuid
	}
	if m&os.ModeSetgid != 0 {
		r |= ModeSetgid
	}

	return r
}

func (m FileMode) OS() os.FileMode {
	r := os.FileMode(m.Perm())

	if m&ModeDir != 0 {
		r |= os.ModeDir
	}
	if m&ModeAppend != 0 {
		r |= os.ModeAppend
	}
	if m&ModeExclusive != 0 {
		r |= os.ModeExclusive
	}
	if m&ModeTemporary != 0 {
		r |= os.ModeTemporary
	}
	if m&ModeSymlink != 0 {
		r |= os.ModeSymlink
	}
	if m&ModeDevice != 0 {
		r |= os.ModeDevice
	}
	if m&ModeNamedPipe != 0 {
		r |= os.ModeNamedPipe
	}
	if m&ModeSocket != 0 {
		r |= os.ModeSocket
	}
	if m&ModeSetuid != 0 {
		r |= os.ModeSetuid
	}
	if m&ModeSetgid != 0 {
		r |= os.ModeSetgid
	}

	return r
}

func (m FileMode) QIDType() QIDType {
	return QIDType(m >> 24)
}

func (m FileMode) Type() FileMode {
	return m & 0xFFFF0000
}

func (m FileMode) Perm() FileMode {
	return m & 0777
}

func (m FileMode) String() string {
	buf := []byte("----------")

	const types = "dalMATL!DpSug"
	for i := range types {
		if m&(1<<uint(31-i)) != 0 {
			buf[0] = types[i]
		}
	}

	const perms = "rwx"
	for i := 1; i < len(buf); i++ {
		if m&(1<<uint32(len(buf)-1-i)) != 0 {
			buf[i] = perms[(i-1)%len(perms)]
		}
	}

	return *(*string)(unsafe.Pointer(&buf))
}

func (m FileMode) encode(e *encoder) {
	e.Encode(uint32(m))
}

func (m *FileMode) decode(d *decoder) {
	d.Decode((*uint32)(m))
}

// Stat is a stat value.
type Stat struct {
	Type   uint16
	Dev    uint32
	QID    QID
	Mode   FileMode
	ATime  time.Time
	MTime  time.Time
	Length uint64
	Name   string
	UID    string
	GID    string
	MUID   string
}

func (s Stat) dirEntry() DirEntry {
	return DirEntry{
		Mode:   s.Mode,
		ATime:  s.ATime,
		MTime:  s.MTime,
		Length: s.Length,
		Name:   s.Name,
		UID:    s.UID,
		GID:    s.GID,
		MUID:   s.MUID,
	}
}

func (s Stat) size() uint16 {
	return uint16(47 + len(s.Name) + len(s.UID) + len(s.GID) + len(s.MUID))
}

func (s Stat) encode(e *encoder) {
	e.Encode(s.size())
	e.Encode(s.Type)
	e.Encode(s.Dev)
	e.Encode(s.QID)
	e.Encode(s.Mode)
	e.Encode(s.ATime)
	e.Encode(s.MTime)
	e.Encode(s.Length)
	e.Encode(s.Name)
	e.Encode(s.UID)
	e.Encode(s.GID)
	e.Encode(s.MUID)
}

func (s *Stat) decode(d *decoder) {
	var size uint16
	d.Decode(&size)

	r := d.r
	d.r = &LimitedReader{
		R: r,
		N: uint32(size),
		E: ErrLargeStat,
	}
	defer func() {
		d.r = r
	}()

	d.Decode(&s.Type)
	d.Decode(&s.Dev)
	d.Decode(&s.QID)
	d.Decode(&s.Mode)
	d.Decode(&s.ATime)
	d.Decode(&s.MTime)
	d.Decode(&s.Length)
	d.Decode(&s.Name)
	d.Decode(&s.UID)
	d.Decode(&s.GID)
	d.Decode(&s.MUID)
}

// DirEntry is a smaller version of Stat that eliminates unnecessary
// or duplicate fields.
type DirEntry struct {
	Mode   FileMode
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
		Type: uint16(d.Mode >> 16),
		QID: QID{
			Type: QIDType(d.Mode >> 24),
			Path: path,
		},
		Mode:   d.Mode,
		ATime:  d.ATime,
		MTime:  d.MTime,
		Length: d.Length,
		Name:   d.Name,
		UID:    d.UID,
		GID:    d.GID,
		MUID:   d.MUID,
	}
}

// StatChanges is a wrapper around DirEntry that is used in wstat
// requests. If one of its methods returns false, that field should be
// considered unset in the DirEntry.
type StatChanges struct {
	DirEntry
}

func (c StatChanges) Mode() (FileMode, bool) { // nolint
	return c.DirEntry.Mode, c.DirEntry.Mode != 0xFFFFFFFF
}

func (c StatChanges) ATime() (time.Time, bool) { // nolint
	return c.DirEntry.ATime, c.DirEntry.ATime.Unix() != -1
}

func (c StatChanges) MTime() (time.Time, bool) { // nolint
	return c.DirEntry.MTime, c.DirEntry.MTime.Unix() != -1
}

func (c StatChanges) Length() (uint64, bool) { // nolint
	return c.DirEntry.Length, c.DirEntry.Length != 0xFFFFFFFFFFFFFFFF
}

func (c StatChanges) Name() (string, bool) { // nolint
	return c.DirEntry.Name, c.DirEntry.Name != ""
}

func (c StatChanges) UID() (string, bool) { // nolint
	return c.DirEntry.UID, c.DirEntry.UID != ""
}

func (c StatChanges) GID() (string, bool) { // nolint
	return c.DirEntry.GID, c.DirEntry.GID != ""
}

func (c StatChanges) MUID() (string, bool) { // nolint
	return c.DirEntry.MUID, c.DirEntry.MUID != ""
}
