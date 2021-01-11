package fs

import (
	"encoding/hex"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sequix/casync-snapshotter/pkg/log"
)

// IsPathExist returns whether the given path exists.
func IsPathExist(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.G.WithError(err).With("path", path).Error("cannot stat")
	}
	return true
}

// IsEmptyDir returns whether the given path is a empty dir.
func IsEmptyDir(path string) bool {
	d, err := os.Open(path)
	if err != nil {
		log.G.WithError(err).With("path", path).Error("cannot open")
		return false
	}
	ds, err := d.Stat()
	if err != nil {
		log.G.WithError(err).With("path", path).Error("cannot stat")
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
		log.G.WithError(err).With("path", path).Error("cannot readdir")
		return false
	}
	return len(fs) == 0
}

// TempDir behaves exactly like ioutil.TempDir, except
// 1.this func do not return error.
// 2.this func return a func to clean that tempdir, on which you can call defer.
func TempDir(parentDir string, prefix string) (string, func()) {
	if len(parentDir) == 0 {
		parentDir = os.TempDir()
	}
	rst, err := ioutil.TempDir(parentDir, prefix)
	if err != nil {
		log.G.WithError(err).Fatalf("create temp dir under %s with prefix %s", parentDir, prefix)
	}
	return rst, func() {
		if err := os.RemoveAll(rst); err != nil {
			log.G.WithError(err).Errorf("remove dir %s", rst)
		}
	}
}

func TempFile(parentDir string, prefix string) (string, func()) {
	f, err := ioutil.TempFile(parentDir, prefix)
	if err != nil {
		log.G.WithError(err).Fatalf("create temp file under %s with prefix %s", parentDir, prefix)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		log.G.WithError(err).Fatalf("close temp file %s", path)
	}
	return path, func() {
		if err := os.Remove(path); err != nil {
			log.G.WithError(err).Fatalf("remove temp file %s", path)
		}
	}
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
