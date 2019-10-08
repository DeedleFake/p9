package proto_test

import (
	"bytes"
	"testing"

	"github.com/DeedleFake/p9"
)

func TestReadWrite(t *testing.T) {
	var buf bytes.Buffer
	err := p9.Proto.Send(&buf, 3, &p9.Tversion{
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
