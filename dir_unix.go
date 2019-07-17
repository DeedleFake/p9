// +build linux darwin

package p9

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

func infoToEntry(fi os.FileInfo) DirEntry {
	sys, _ := fi.Sys().(*syscall.Stat_t)
	if sys == nil {
		return DirEntry{
			Mode:   ModeFromOS(fi.Mode()),
			MTime:  fi.ModTime(),
			Length: uint64(fi.Size()),
			Name:   fi.Name(),
		}
	}

	var uname string
	uid, err := user.LookupId(strconv.FormatUint(uint64(sys.Uid), 10))
	if err == nil {
		uname = uid.Username
	}

	var gname string
	gid, err := user.LookupGroupId(strconv.FormatUint(uint64(sys.Gid), 10))
	if err == nil {
		gname = gid.Name
	}

	return DirEntry{
		Mode:   ModeFromOS(fi.Mode()),
		ATime:  time.Unix(sys.Atim.Unix()),
		MTime:  fi.ModTime(),
		Length: uint64(fi.Size()),
		Name:   fi.Name(),
		UID:    uname,
		GID:    gname,
	}
}
