package p9_test

import (
	"bytes"
	"testing"

	"github.com/DeedleFake/p9"
)

func TestWriteMessage(t *testing.T) {
	var buf bytes.Buffer
	err := p9.WriteMessage(&buf, 3, &p9.Tversion{
		Msize:   9,
		Version: "This is a test.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("(%x) %x", buf.Len(), buf.Bytes())
	t.Logf("%s", buf.Bytes())

	msg, tag, err := p9.ReadMessage(&buf, uint32(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", msg)
	t.Log(tag)
}
