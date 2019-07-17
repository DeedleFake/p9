// +build !linux,!darwin,!plan9,!windows

package p9

import "os"

func infoToEntry(fi os.FileInfo) DirEntry {
	return DirEntry{
		Mode:   ModeFromOS(fi.Mode()),
		MTime:  fi.ModTime(),
		Length: uint64(fi.Size()),
		Name:   fi.Name(),
	}
}
