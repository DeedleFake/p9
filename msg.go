package p9

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"
)

var (
	bufPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

func WriteMessage(w io.Writer, tag uint16, msg Message) error {
	buf := bufPool.Get().(*bytes.Buffer)

	e := &encoder{w: buf}
	err := e.Encode(msg)
	if err != nil {
		return err
	}

	e.w = w
	e.Encode(4 + 1 + 2 + uint32(buf.Len()))
	e.Encode(msg.Type())
	e.Encode(tag)
	msg.encode(e)

	return e.err
}

// A Message is any 9P message, either T or R, minus the tag.
type Message interface {
	// Type returns the message type.
	Type() MessageType

	encodable
}

const (
	NoTag uint16 = math.MaxUint16
	NoFID uint32 = math.MaxUint32
)

type MessageType uint8

const (
	TversionType MessageType = 100 + iota
	RversionType
	TauthType
	RauthType
	TattachType
	RattachType
	terrorType // Not used.
	RerrorType
	TflushType
	RflushType
	TwalkType
	RwalkType
	TopenType
	RopenType
	TcreateType
	RcreateType
	TreadType
	RreadType
	TwriteType
	RwriteType
	TclunkType
	RclunkType
	TremoveType
	RremoveType
	TstatType
	RstatType
	TwstatType
	RwstatType
)

func (t MessageType) encode(e *encoder) {
	e.Encode(uint8(t))
}

type encoder struct {
	w   io.Writer
	err error
}

func (e *encoder) Write(data []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}

	n, err := e.w.Write(data)
	e.err = err
	return n, err
}

func (e *encoder) Encode(v interface{}) error {
	if e.err != nil {
		return e.err
	}

	if v, ok := v.(encodable); ok {
		v.encode(e)
		return e.err
	}

	switch v := v.(type) {
	case uint8, uint16, uint32, uint64, int8, int16, int32, int64:
		e.err = binary.Write(e, binary.LittleEndian, v)
		return e.err

	case []byte:
		err := binary.Write(e, binary.LittleEndian, uint32(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		e.err = binary.Write(e, binary.LittleEndian, v)
		return e.err

	case string:
		err := binary.Write(e, binary.LittleEndian, uint16(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		e.err = binary.Write(e, binary.LittleEndian, []byte(v))
		return e.err

	case []string:
		err := binary.Write(e, binary.LittleEndian, uint16(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		for _, str := range v {
			err := e.Encode(str)
			if err != nil {
				e.err = err
				return err
			}
		}

		return e.err

	case []QID:
		err := binary.Write(e, binary.LittleEndian, uint16(len(v)))
		if err != nil {
			e.err = err
			return err
		}

		for _, qid := range v {
			err := e.Encode(qid)
			if err != nil {
				e.err = err
				return err
			}
		}

		return e.err

	default:
		panic(fmt.Errorf("Unexpected type: %T", v))
	}
}

type encodable interface {
	encode(e *encoder)
}

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

type Stat struct {
	Type   uint16
	Dev    uint32
	QID    QID
	Mode   uint32 // TODO: Make a Mode type?
	ATime  uint32
	MTime  uint32
	Length uint64
	Name   string
	UID    string
	GID    string
	MUID   string
}

func (s *Stat) encode(e *encoder) {
	// size is the size of the data, not including the strings.
	const size = 41

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

type Tversion struct {
	Version string
}

func (msg Tversion) Type() MessageType {
	return TversionType
}

func (msg Tversion) encode(e *encoder) {
	e.Encode(msg.Version)
}

type Rversion struct {
	Version string
}

func (msg Rversion) Type() MessageType {
	return RversionType
}

func (msg Rversion) encode(e *encoder) {
	e.Encode(msg.Version)
}

type Tauth struct {
	AFID  uint32
	Uname string
	Aname string
}

func (msg Tauth) Type() MessageType {
	return TauthType
}

func (msg Tauth) encode(e *encoder) {
	e.Encode(msg.AFID)
	e.Encode(msg.Uname)
	e.Encode(msg.Aname)
}

type Rauth struct {
	AQID QID
}

func (msg Rauth) Type() MessageType {
	return RauthType
}

func (msg Rauth) encode(e *encoder) {
	e.Encode(msg.AQID)
}

type Tattach struct {
	FID   uint32
	AFID  uint32
	Uname string
	Aname string
}

func (msg Tattach) Type() MessageType {
	return TattachType
}

func (msg Tattach) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.AFID)
	e.Encode(msg.Uname)
	e.Encode(msg.Aname)
}

type Rattach struct {
	QID QID
}

func (msg Rattach) Type() MessageType {
	return RattachType
}

func (msg Rattach) encode(e *encoder) {
	e.Encode(msg.QID)
}

type Rerror struct {
	Ename string
}

func (msg Rerror) Type() MessageType {
	return RerrorType
}

func (msg Rerror) encode(e *encoder) {
	e.Encode(msg.Ename)
}

type Tflush struct {
	OldTag uint16
}

func (msg Tflush) Type() MessageType {
	return TflushType
}

func (msg Tflush) encode(e *encoder) {
	e.Encode(msg.OldTag)
}

type Rflush struct {
}

func (msg Rflush) Type() MessageType {
	return RflushType
}

func (msg Rflush) encode(e *encoder) {
}

type Twalk struct {
	FID    uint32
	NewFID uint32
	Wname  []string
}

func (msg Twalk) Type() MessageType {
	return TwalkType
}

func (msg Twalk) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.NewFID)
	e.Encode(msg.Wname)
}

type Rwalk struct {
	WQID []QID
}

func (msg Rwalk) Type() MessageType {
	return RwalkType
}

func (msg Rwalk) encode(e *encoder) {
	e.Encode(msg.WQID)
}

type Topen struct {
	FID  uint32
	Mode uint8 // TODO: Make a Mode type?
}

func (msg Topen) Type() MessageType {
	return TopenType
}

func (msg Topen) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.Mode)
}

type Ropen struct {
	QID    QID
	IOUnit uint32
}

func (msg Ropen) Type() MessageType {
	return RopenType
}

func (msg Ropen) encode(e *encoder) {
	e.Encode(msg.QID)
	e.Encode(msg.IOUnit)
}

type Tcreate struct {
	FID  uint32
	Name string
	Perm uint32 // TODO: Make a Perm type?
	Mode uint8  // TODO: Make a Mode type?
}

func (msg Tcreate) Type() MessageType {
	return TcreateType
}

func (msg Tcreate) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.Name)
	e.Encode(msg.Perm)
	e.Encode(msg.Mode)
}

type Rcreate struct {
	QID    QID
	IOUnit uint32
}

func (msg Rcreate) Type() MessageType {
	return RcreateType
}

func (msg Rcreate) encode(e *encoder) {
	e.Encode(msg.QID)
	e.Encode(msg.IOUnit)
}

type Tread struct {
	FID    uint32
	Offset uint64
	Count  uint32
}

func (msg Tread) Type() MessageType {
	return TreadType
}

func (msg Tread) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.Offset)
	e.Encode(msg.Count)
}

// TODO: Figure out a clean way to allow handlers to send responses
// via an io.Writer?
type Rread struct {
	Data []byte
}

func (msg Rread) Type() MessageType {
	return RreadType
}

func (msg Rread) encode(e *encoder) {
	e.Encode(msg.Data)
}

// TODO: Figure out a clean way to allow clients request writes via an
// io.Writer?
type Twrite struct {
	FID    uint32
	Offset uint64
	Data   []byte
}

func (msg Twrite) Type() MessageType {
	return TwriteType
}

func (msg Twrite) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.Offset)
	e.Encode(msg.Data)
}

type Rwrite struct {
	Count uint32
}

func (msg Rwrite) Type() MessageType {
	return RwriteType
}

func (msg Rwrite) encode(e *encoder) {
	e.Encode(msg.Count)
}

type Tclunk struct {
	FID uint32
}

func (msg Tclunk) Type() MessageType {
	return TclunkType
}

func (msg Tclunk) encode(e *encoder) {
	e.Encode(msg.FID)
}

type Rclunk struct {
}

func (msg Rclunk) Type() MessageType {
	return RclunkType
}

func (msg Rclunk) encode(e *encoder) {
}

type Tremove struct {
	FID uint32
}

func (msg Tremove) Type() MessageType {
	return TremoveType
}

func (msg Tremove) encode(e *encoder) {
	e.Encode(msg.FID)
}

type Rremove struct {
}

func (msg Rremove) Type() MessageType {
	return RremoveType
}

func (msg Rremove) encode(e *encoder) {
}

type Tstat struct {
	FID uint32
}

func (msg Tstat) Type() MessageType {
	return TstatType
}

func (msg Tstat) encode(e *encoder) {
	e.Encode(msg.FID)
}

type Rstat struct {
	Stat []Stat
}

func (msg Rstat) Type() MessageType {
	return RstatType
}

func (msg Rstat) encode(e *encoder) {
	e.Encode(msg.Stat)
}

type Twstat struct {
	FID  uint32
	Stat []Stat
}

func (msg Twstat) Type() MessageType {
	return TwstatType
}

func (msg Twstat) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.Stat)
}

type Rwstat struct {
}

func (msg Rwstat) Type() MessageType {
	return RwstatType
}

func (msg Rwstat) encode(e *encoder) {
}
