package proto_test

import (
	"bytes"
	"testing"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/proto"
)

func TestReadWrite(t *testing.T) {
	var buf bytes.Buffer
	err := proto.Send(&buf, 3, uint8(p9.TversionType), &p9.Tversion{
		Msize:   9,
		Version: "This is a test.",
	})
	if err != nil {
		t.Error(err)
	}
	t.Logf("(%x) %x", buf.Len(), buf.Bytes())
	t.Logf("%s", buf.Bytes())

	msg, tag, err := p9.Proto.Receive(&buf, 0)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%#v", msg)
	t.Log(tag)
}
