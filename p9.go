package p9

import (
	"math"
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

// Other constants.
const (
	IOHeaderSize = 24
)
