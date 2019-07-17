// +build plan9

package p9

import (
	"os"
	"syscall"
	"time"
)

func infoToEntry(fi os.FileInfo) DirEntry {
	sys, _ := fi.Sys().(*syscall.Dir)
	if sys == nil {
		return DirEntry{
			Mode:   ModeFromOS(fi.Mode()),
			MTime:  fi.ModTime(),
			Length: uint64(fi.Size()),
			Name:   fi.Name(),
		}
	}

	return DirEntry{
		Mode:   ModeFromOS(fi.Mode()),
		ATime:  time.Unix(int64(sys.Atime), 0),
		MTime:  fi.ModTime(),
		Length: uint64(fi.Size()),
		Name:   fi.Name(),
		UID:    sys.Uid,
		GID:    sys.Gid,
		MUID:   sys.Muid,
	}
}