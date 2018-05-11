package main

import (
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/DeedleFake/p9"
)

type FS map[string]*File

func (fs FS) Type(path string) (p9.QIDType, bool) {
	file, ok := fs[path]
	if !ok {
		return 0, false
	}
	return file.Type, true
}

func (fs FS) Open(path string, mode uint8) (p9.File, error) {
	file, ok := fs[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return file, nil
}

type File struct {
	m sync.RWMutex

	Type p9.QIDType
	Data []byte
}

func (file *File) ReadAt(buf []byte, off int64) (int, error) {
	file.m.RLock()
	defer file.m.RUnlock()

	if off >= int64(len(file.Data)) {
		return 0, io.EOF
	}

	n := copy(buf, file.Data[off:])
	if n < len(buf) {
		return n, io.EOF
	}
	return n, nil
}

func (file *File) WriteAt(buf []byte, off int64) (int, error) {
	file.m.Lock()
	defer file.m.Unlock()

	file.Data = append(file.Data[:off], append(buf[:len(buf):len(buf)], file.Data[int(off)+len(buf):]...)...)
	return len(buf), nil
}

func (file File) Close() error {
	return nil
}

func (file *File) Readdir() ([]p9.Stat, error) {
	panic("Not implemented.")
}

var (
	fs = FS{
		"/": &File{
			Type: p9.QTDir,
		},

		"/test": &File{
			Type: p9.QTFile,
			Data: []byte("This is a test."),
		},
	}
)

func connHandler() p9.MessageHandler {
	return p9.HandleFS(fs, 1024)
}

func main() {
	lis, err := net.Listen("tcp", "localhost:5640")
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer lis.Close()

	err = p9.Serve(lis, p9.ConnHandlerFunc(connHandler))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
