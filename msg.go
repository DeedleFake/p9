package p9

import (
	"errors"
	"io"
	"math"
)

var (
	ErrLargeMessage = errors.New("Message larger than msize")
	ErrLargeStat    = errors.New("Stat larger that declared size")
)

// WriteMessage writes msg to w with the given tag. It returns an
// error if any are encountered.
func WriteMessage(w io.Writer, tag uint16, msg Message) error {
	e := &encoder{
		w: w,
	}

	e.mode = e.size
	e.Encode(msg)

	e.mode = e.write
	e.Encode(4 + 1 + 2 + e.n)
	e.Encode(msg.Type())
	e.Encode(tag)
	e.Encode(msg)

	return e.err
}

// ReadMessage reads the next message from r, returning both it and
// its tag. It also returns an error if any were encountered.
//
// If msize is positive and the message read is greater than it then
// ErrLargeMessage is returned.
func ReadMessage(r io.Reader, msize uint32) (Message, uint16, error) {
	d := &decoder{
		r: r,
	}

	var size uint32
	d.Decode(&size)
	d.r = &LimitedReader{
		R: d.r,
		N: size,
		E: ErrLargeMessage,
	}

	var msgType MessageType
	d.Decode(&msgType)

	tag := NoTag
	d.Decode(&tag)

	if d.err != nil {
		return nil, tag, d.err
	}

	var msg Message
	switch msgType {
	case TversionType:
		msg = new(Tversion)
	case RversionType:
		msg = new(Rversion)
	case TauthType:
		msg = new(Tauth)
	case RauthType:
		msg = new(Rauth)
	case TattachType:
		msg = new(Tattach)
	case RattachType:
		msg = new(Rattach)
	case RerrorType:
		msg = new(Rerror)
	case TflushType:
		msg = new(Tflush)
	case RflushType:
		msg = new(Rflush)
	case TwalkType:
		msg = new(Twalk)
	case RwalkType:
		msg = new(Rwalk)
	case TopenType:
		msg = new(Topen)
	case RopenType:
		msg = new(Ropen)
	case TcreateType:
		msg = new(Tcreate)
	case RcreateType:
		msg = new(Rcreate)
	case TreadType:
		msg = new(Tread)
	case RreadType:
		msg = new(Rread)
	case TwriteType:
		msg = new(Twrite)
	case RwriteType:
		msg = new(Rwrite)
	case TclunkType:
		msg = new(Tclunk)
	case RclunkType:
		msg = new(Rclunk)
	case TremoveType:
		msg = new(Tremove)
	case RremoveType:
		msg = new(Rremove)
	case TstatType:
		msg = new(Tstat)
	case RstatType:
		msg = new(Rstat)
	case TwstatType:
		msg = new(Twstat)
	case RwstatType:
		msg = new(Rwstat)
	}

	d.Decode(msg)
	return msg, tag, d.err
}

func EncodeDir(w io.Writer, dir []Stat) error {
	e := &encoder{
		w: w,
	}
	e.mode = e.write

	for _, stat := range dir {
		e.Encode(stat)
	}
	return e.err
}

// A Message is any 9P message, either T or R, minus the tag.
type Message interface {
	// Type returns the message type.
	Type() MessageType

	encodable
	decodable
}

const (
	// NoTag is a special tag that is used when the tag is unimportant.
	NoTag uint16 = math.MaxUint16

	// NoFID is a special FID that is used to signal a lack of an FID.
	NoFID uint32 = math.MaxUint32
)

// MessageType is the 8-bit identifier of a message's type.
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

const (
	OREAD uint8 = iota
	OWRITE
	ORDWR
	OEXEC

	OTRUNC  uint8 = 0x10
	ORCLOSE uint8 = 0x40
)

func (t MessageType) encode(e *encoder) {
	e.Encode(uint8(t))
}

func (t *MessageType) decode(d *decoder) {
	d.Decode((*uint8)(t))
}

type Tversion struct {
	Msize   uint32
	Version string
}

func (msg Tversion) Type() MessageType {
	return TversionType
}

func (msg Tversion) encode(e *encoder) {
	e.Encode(msg.Msize)
	e.Encode(msg.Version)
}

func (msg *Tversion) decode(d *decoder) {
	d.Decode(&msg.Msize)
	d.Decode(&msg.Version)
}

type Rversion struct {
	Msize   uint32
	Version string
}

func (msg Rversion) Type() MessageType {
	return RversionType
}

func (msg Rversion) encode(e *encoder) {
	e.Encode(msg.Msize)
	e.Encode(msg.Version)
}

func (msg *Rversion) decode(d *decoder) {
	d.Decode(&msg.Msize)
	d.Decode(&msg.Version)
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

func (msg *Tauth) decode(d *decoder) {
	d.Decode(&msg.AFID)
	d.Decode(&msg.Uname)
	d.Decode(&msg.Aname)
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

func (msg *Rauth) decode(d *decoder) {
	d.Decode(&msg.AQID)
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

func (msg *Tattach) decode(d *decoder) {
	d.Decode(&msg.FID)
	d.Decode(&msg.AFID)
	d.Decode(&msg.Uname)
	d.Decode(&msg.Aname)
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

func (msg *Rattach) decode(d *decoder) {
	d.Decode(&msg.QID)
}

type Rerror struct {
	Ename string
}

func (msg Rerror) Type() MessageType {
	return RerrorType
}

func (msg Rerror) Error() string {
	return msg.Ename
}

func (msg Rerror) encode(e *encoder) {
	e.Encode(msg.Ename)
}

func (msg *Rerror) decode(d *decoder) {
	d.Decode(&msg.Ename)
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

func (msg *Tflush) decode(d *decoder) {
	d.Decode(&msg.OldTag)
}

type Rflush struct {
}

func (msg Rflush) Type() MessageType {
	return RflushType
}

func (msg Rflush) encode(e *encoder) {
}

func (msg *Rflush) decode(d *decoder) {
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

func (msg *Twalk) decode(d *decoder) {
	d.Decode(&msg.FID)
	d.Decode(&msg.NewFID)
	d.Decode(&msg.Wname)
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

func (msg *Rwalk) decode(d *decoder) {
	d.Decode(&msg.WQID)
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

func (msg *Topen) decode(d *decoder) {
	d.Decode(&msg.FID)
	d.Decode(&msg.Mode)
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

func (msg *Ropen) decode(d *decoder) {
	d.Decode(&msg.QID)
	d.Decode(&msg.IOUnit)
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

func (msg *Tcreate) decode(d *decoder) {
	d.Decode(&msg.FID)
	d.Decode(&msg.Name)
	d.Decode(&msg.Perm)
	d.Decode(&msg.Mode)
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

func (msg *Rcreate) decode(d *decoder) {
	d.Decode(&msg.QID)
	d.Decode(&msg.IOUnit)
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

func (msg *Tread) decode(d *decoder) {
	d.Decode(&msg.FID)
	d.Decode(&msg.Offset)
	d.Decode(&msg.Count)
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

func (msg *Rread) decode(d *decoder) {
	d.Decode(&msg.Data)
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

func (msg *Twrite) decode(d *decoder) {
	d.Decode(&msg.FID)
	d.Decode(&msg.Offset)
	d.Decode(&msg.Data)
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

func (msg *Rwrite) decode(d *decoder) {
	d.Decode(&msg.Count)
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

func (msg *Tclunk) decode(d *decoder) {
	d.Decode(&msg.FID)
}

type Rclunk struct {
}

func (msg Rclunk) Type() MessageType {
	return RclunkType
}

func (msg Rclunk) encode(e *encoder) {
}

func (msg *Rclunk) decode(d *decoder) {
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

func (msg *Tremove) decode(d *decoder) {
	d.Decode(&msg.FID)
}

type Rremove struct {
}

func (msg Rremove) Type() MessageType {
	return RremoveType
}

func (msg Rremove) encode(e *encoder) {
}

func (msg *Rremove) decode(d *decoder) {
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

func (msg *Tstat) decode(d *decoder) {
	d.Decode(&msg.FID)
}

type Rstat struct {
	Stat Stat
}

func (msg Rstat) Type() MessageType {
	return RstatType
}

func (msg Rstat) encode(e *encoder) {
	e.Encode(msg.Stat.size() + 2)
	e.Encode(msg.Stat)
}

func (msg *Rstat) decode(d *decoder) {
	d.Decode(new(uint16)) // size
	d.Decode(&msg.Stat)
}

type Twstat struct {
	FID  uint32
	Stat Stat
}

func (msg Twstat) Type() MessageType {
	return TwstatType
}

func (msg Twstat) encode(e *encoder) {
	e.Encode(msg.FID)
	e.Encode(msg.Stat.size())
	e.Encode(msg.Stat)
}

func (msg *Twstat) decode(d *decoder) {
	d.Decode(&msg.FID)
	d.Decode(new(uint16))
	d.Decode(&msg.Stat)
}

type Rwstat struct {
}

func (msg Rwstat) Type() MessageType {
	return RwstatType
}

func (msg Rwstat) encode(e *encoder) {
}

func (msg *Rwstat) decode(d *decoder) {
}
