package p9_test

import (
	"bytes"
	"testing"

	"github.com/DeedleFake/p9"
)

func TestWriteMessage(t *testing.T) {
	var buf bytes.Buffer
	err := p9.WriteMessage(&buf, 3, p9.Tversion{
		Version: "This is a test.",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(buf.Bytes())
	t.Log(buf.Len())

	msg, tag, err := p9.ReadMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", msg)
	t.Log(tag)
}
