// +build linux

package fs

import (
	"encoding/hex"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sequix/dequash-snapshotter/pkg/log"

	"golang.org/x/sys/unix"
)

// IsPathExist returns whether the given path exists.
func IsPathExist(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.WithError(err).With("path", path).Error("cannot stat")
	}
	return true
}

// IsEmptyDir returns whether the given path is a empty dir.
func IsEmptyDir(path string) bool {
	d, err := os.Open(path)
	if err != nil {
		log.WithError(err).With("path", path).Error("cannot open")
		return false
	}
	ds, err := d.Stat()
	if err != nil {
		log.WithError(err).With("path", path).Error("cannot stat")
		return false
	}
	if !ds.IsDir() {
		return false
	}
	fs, err := d.Readdir(1)
	if err != nil {
		if err == io.EOF {
			return true
		}
		log.WithError(err).With("path", path).Error("cannot readdir")
		return false
	}
	return len(fs) == 0
}

// IsReadable returns whether the given path is a regular file.
func IsRegularFile(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		log.WithError(err).With("path", path).Error("cannot stat")
		return false
	}
	return st.Mode().IsRegular()
}

// IsReadable returns whether the given path is readable.
func IsReadable(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		log.WithError(err).With("path", path).Error("cannot stat")
		return false
	}
	perm := st.Mode() & os.ModePerm
	if perm & 0444 == 0444 {
		return true
	}
	fi := st.Sys().(*syscall.Stat_t)
	if perm & 0400 == 0400 && int(fi.Uid) == os.Getuid()  {
		return true
	}
	if perm & 040 == 040 && int(fi.Gid) == os.Getegid() {
		return true
	}
	if perm & 04 == 04 {
		return true
	}
	return false
}

func IsLoopDevice(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		log.WithError(err).With("path", path).Error("cannot stat")
		return false
	}
	if st.Mode() & os.ModeType != os.ModeDevice {
		return false
	}
	fi := st.Sys().(*syscall.Stat_t)
	return fi.Rdev >> 8 == 7
}

// IsExecutable returns whether the given path is a executable file.
func IsExecutable(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		log.WithError(err).With("path", path).Error("cannot stat")
		return false
	}
	mode := st.Mode()
	if !mode.IsRegular() {
		return false
	}
	perm := mode & os.ModePerm
	if perm & 0111 == 0111 {
		return true
	}
	if perm & 0111 == 0 {
		return false
	}
	fi := st.Sys().(*syscall.Stat_t)
	if perm & 0100 == 0100 && int(fi.Uid) == os.Getuid()  {
		return true
	}
	if perm & 010 == 010 && int(fi.Gid) == os.Getegid() {
		return true
	}
	if perm & 01 == 01 {
		return true
	}
	return false
}

// Lchtimes sets the access and modification times tv on path.
// If path refers to a symlink, it is not dereferenced and the timestamps are set on the symlink.
func Lchtimes(filename string, atime, mtime time.Time) error {
	return unix.Lutimes(filename, []unix.Timeval{
		{
			Sec:  atime.Unix(),
			Usec: int64(atime.Nanosecond()),
		},
		{
			Sec:  mtime.Unix(),
			Usec: int64(mtime.Nanosecond()),
		},
	})
}

var randSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// TempFilename generates global unique temporary filename.
func TempFilename(prefix string) string {
	randBytes := make([]byte, 4)
	randSource.Read(randBytes)
	return filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes))
}