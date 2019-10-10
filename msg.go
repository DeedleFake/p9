package p9

import (
	"bytes"
	"io"
	"reflect"

	"github.com/DeedleFake/p9/internal/util"
	"github.com/DeedleFake/p9/proto"
)

const (
	TversionType uint8 = 100 + iota
	RversionType
	TauthType
	RauthType
	TattachType
	RattachType
	_ // Terror isn't used, but the slot is skipped over.
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

var protocol = proto.NewProto(map[uint8]reflect.Type{
	TversionType: reflect.TypeOf(Tversion{}),
	RversionType: reflect.TypeOf(Rversion{}),
	TauthType:    reflect.TypeOf(Tauth{}),
	RauthType:    reflect.TypeOf(Rauth{}),
	TattachType:  reflect.TypeOf(Tattach{}),
	RattachType:  reflect.TypeOf(Rattach{}),
	RerrorType:   reflect.TypeOf(Rerror{}),
	TflushType:   reflect.TypeOf(Tflush{}),
	RflushType:   reflect.TypeOf(Rflush{}),
	TwalkType:    reflect.TypeOf(Twalk{}),
	RwalkType:    reflect.TypeOf(Rwalk{}),
	TopenType:    reflect.TypeOf(Topen{}),
	RopenType:    reflect.TypeOf(Ropen{}),
	TcreateType:  reflect.TypeOf(Tcreate{}),
	RcreateType:  reflect.TypeOf(Rcreate{}),
	TreadType:    reflect.TypeOf(Tread{}),
	RreadType:    reflect.TypeOf(Rread{}),
	TwriteType:   reflect.TypeOf(Twrite{}),
	RwriteType:   reflect.TypeOf(Rwrite{}),
	TclunkType:   reflect.TypeOf(Tclunk{}),
	RclunkType:   reflect.TypeOf(Rclunk{}),
	TremoveType:  reflect.TypeOf(Tremove{}),
	RremoveType:  reflect.TypeOf(Rremove{}),
	TstatType:    reflect.TypeOf(Tstat{}),
	RstatType:    reflect.TypeOf(Rstat{}),
	TwstatType:   reflect.TypeOf(Twstat{}),
	RwstatType:   reflect.TypeOf(Rwstat{}),
})

func Proto() proto.Proto {
	return protocol
}

type Tversion struct { // nolint
	Msize   uint32
	Version string
}

func (Tversion) P9NoTag() {}

type Rversion struct { // nolint
	Msize   uint32
	Version string
}

type Tauth struct { // nolint
	AFID  uint32
	Uname string
	Aname string
}

type Rauth struct { // nolint
	AQID QID
}

type Tattach struct { // nolint
	FID   uint32
	AFID  uint32
	Uname string
	Aname string
}

type Rattach struct { // nolint
	QID QID
}

// Rerror is a special response that represents an error. As a special
// case, this type also implements error for convenience.
type Rerror struct {
	Ename string
}

func (msg Rerror) Error() string { // nolint
	return msg.Ename
}

type Tflush struct { // nolint
	OldTag uint16
}

type Rflush struct { // nolint
}

type Twalk struct { // nolint
	FID    uint32
	NewFID uint32
	Wname  []string
}

type Rwalk struct { // nolint
	WQID []QID
}

type Topen struct { // nolint
	FID  uint32
	Mode uint8 // TODO: Make a Mode type?
}

type Ropen struct { // nolint
	QID    QID
	IOUnit uint32
}

type Tcreate struct { // nolint
	FID  uint32
	Name string
	Perm FileMode
	Mode uint8 // TODO: Make a Mode type?
}

type Rcreate struct { // nolint
	QID    QID
	IOUnit uint32
}

type Tread struct { // nolint
	FID    uint32
	Offset uint64
	Count  uint32
}

type Rread struct { // nolint
	Data []byte
}

type Twrite struct { // nolint
	FID    uint32
	Offset uint64
	Data   []byte
}

type Rwrite struct { // nolint
	Count uint32
}

type Tclunk struct { // nolint
	FID uint32
}

type Rclunk struct { // nolint
}

type Tremove struct { // nolint
	FID uint32
}

type Rremove struct { // nolint
}

type Tstat struct { // nolint
	FID uint32
}

type Rstat struct { // nolint
	Stat Stat
}

func (stat Rstat) P9Encode() ([]byte, error) {
	var buf bytes.Buffer

	err := proto.Write(&buf, stat.Stat.size()+2)
	if err != nil {
		return nil, err
	}

	err = proto.Write(&buf, stat.Stat)
	return buf.Bytes(), err
}

func (stat *Rstat) P9Decode(r io.Reader) error {
	var size uint16
	err := proto.Read(r, &size)
	if err != nil {
		return err
	}

	r = &util.LimitedReader{
		R: r,
		N: uint32(size),
		E: ErrLargeStat,
	}

	return proto.Read(r, &stat.Stat)
}

type Twstat struct { // nolint
	FID  uint32
	Stat Stat
}

type Rwstat struct { // nolint
}
