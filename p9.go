package p9

import (
	"math"
	"time"
)

const (
	// Version is the 9P version implemented by this package, both for
	// server and client.
	Version = "9P2000"
)

const (
	// NoTag is a special tag that is used when the tag is unimportant.
	NoTag uint16 = math.MaxUint16

	// NoFID is a special FID that is used to signal a lack of an FID.
	NoFID uint32 = math.MaxUint32
)

// File open modes and flags.
const (
	OREAD uint8 = iota
	OWRITE
	ORDWR
	OEXEC

	OTRUNC  uint8 = 0x10
	ORCLOSE uint8 = 0x40

	// OWALK is a special flag that is not part of the 9P specification.
	// For more information, see Remote.Open().
	OWALK uint8 = 0xFF

	DMDIR uint32 = 0x80000000
)

// QID represents a QID value.
type QID struct {
	Type    QIDType
	Version uint32
	Path    uint64
}

func (qid QID) encode(e *encoder) {
	e.Encode(qid.Type)
	e.Encode(qid.Version)
	e.Encode(qid.Path)
}

func (qid *QID) decode(d *decoder) {
	d.Decode(&qid.Type)
	d.Decode(&qid.Version)
	d.Decode(&qid.Path)
}

// QIDType represents an 8-bit QID type identifier.
type QIDType uint8

// Valid types of files.
const (
	QTFile    QIDType = 0
	QTSymlink QIDType = 1 << iota
	QTTmp
	QTAuth
	QTMount
	QTExcl
	QTAppend
	QTDir
)

func (t QIDType) encode(e *encoder) {
	e.Encode(uint8(t))
}

func (t *QIDType) decode(d *decoder) {
	d.Decode((*uint8)(t))
}

// Stat is a stat value.
type Stat struct {
	Type   uint16
	Dev    uint32
	QID    QID
	Mode   uint32 // TODO: Make a Mode type?
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
		Type:   s.QID.Type,
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
