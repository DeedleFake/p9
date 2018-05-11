package p9

import "time"

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

func (s Stat) encode(e *encoder) {
	// size is the size of the data, not including the strings
	// themselves but including their length prefixes.
	const size = 47

	e.Encode(uint16(size + len(s.Name) + len(s.UID) + len(s.GID) + len(s.MUID)))
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
