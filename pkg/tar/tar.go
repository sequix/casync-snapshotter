package tar

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sequix/dequash-snapshotter/pkg/fs"
)

func Unpack(tr *tar.Reader, dstDir string) error {
	return unpackHelper(tr, dstDir, lchownNop, lchtimesNop)
}

func UnpackWithOwnerAndTimes(tr *tar.Reader, dstDir string) error {
	return unpackHelper(tr, dstDir, os.Lchown, fs.Lchtimes)
}

type (
	lchown   func(filename string, uid, gid int) error
	lchtimes func(filename string, atime, mtime time.Time) error
)

func lchownNop(_ string, _, _ int) error         { return nil }
func lchtimesNop(_ string, _, _ time.Time) error { return nil }

func unpackHelper(tr *tar.Reader, dstDir string, lchown lchown, lchtimes lchtimes) error {
	for {
		th, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		filename := filepath.Join(dstDir, th.Name)
		switch th.Typeflag {
		case tar.TypeReg:
			f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(th.Mode))
			if err != nil {
				return fmt.Errorf("open %s from layer: %w", filename, err)
			}
			if written, err := io.Copy(f, tr); err != nil {
				return fmt.Errorf("extract %s from layer: written %d, err %w", filename, written, err)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("close %s: %w", filename, err)
			}
		case tar.TypeLink:
			if err := os.Link(filepath.Join(dstDir, th.Linkname), filename); err != nil {
				return fmt.Errorf("make hardlink %s -> %s: %w", th.Name, th.Linkname, err)
			}
		case tar.TypeSymlink:
			if err := os.Symlink(th.Linkname, filename); err != nil {
				return fmt.Errorf("make symlink %s -> %s: %w", th.Name, th.Linkname, err)
			}
		case tar.TypeChar, tar.TypeBlock:
			if err := syscall.Mknod(filename, uint32(th.Mode), int((th.Devmajor<<8)|th.Devminor)); err != nil {
				return fmt.Errorf("mknod %s (%d,%d): %w", th.Name, th.Devmajor, th.Devminor, err)
			}
		case tar.TypeDir:
			if err := os.MkdirAll(filename, os.FileMode(th.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", filename, err)
			}
		case tar.TypeFifo:
			if err := syscall.Mkfifo(filename, uint32(th.Mode)); err != nil {
				return fmt.Errorf("mkfifo %s: %w", filename, err)
			}
		// TODO
		//case tar.TypeXGlobalHeader:
		//case tar.TypeXHeader:
		//case tar.TypeGNUSparse:
		//case tar.TypeGNULongLink:
		//case tar.TypeGNULongName:
		default:
			log.Printf("unknown file type %x for tar entry %s", th.Typeflag, th.Name)
			continue
		}
		// todo test
		if err := lchown(filename, th.Uid, th.Gid); err != nil {
			return fmt.Errorf("chwon %s: %w", filename, err)
		}
		// todo test
		if err := lchtimes(filename, th.AccessTime, th.ModTime); err != nil {
			return fmt.Errorf("change atime & mtime for %s: %w", filename, err)
		}
	}
	return nil
}
