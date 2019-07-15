package p9

import (
	"os"
	"time"
)

// Stat is a stat value.
type Stat struct {
	Type   uint16
	Dev    uint32
	QID    QID
	Mode   os.FileMode
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

// DirEntry is a smaller version of Stat that eliminates unnecessary
// or duplicate fields.
//
// Note that the top 8-bits of the Mode field are overwritten during
// transmission using the Type field.
type DirEntry struct {
	Type   QIDType
	Mode   os.FileMode
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
		Mode:   d.Mode | (os.FileMode(d.Type) << 24),
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

func (c StatChanges) Type() (QIDType, bool) { // nolint
	return c.DirEntry.Type, c.DirEntry.Type != 0xFF
}

func (c StatChanges) Mode() (os.FileMode, bool) { // nolint
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
