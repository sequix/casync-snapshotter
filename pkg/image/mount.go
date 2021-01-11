package image

import (
	stdtar "archive/tar"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/sequix/dequash-snapshotter/pkg/fs"
	"github.com/sequix/dequash-snapshotter/pkg/log"
	"github.com/sequix/dequash-snapshotter/pkg/tar"
)

const (
	// OverlayFS mount requires a little bit time to populate all files in the destination directory.
	// We list the directory periodically until all files a populated.
	mountPollInterval = 1 * time.Second
	// If files were not populated within this time, return an error.
	mountPollTimeout = 10 * time.Second
)

// Decompress image layers to a temporary directory without blocking.
func Mount(ctx context.Context, img v1.Image, mntdir string) (fs.Umount, error) {
	tmpdir, err := ioutil.TempDir("", "dequash-tmp-")
	if err != nil {
		return nil, fmt.Errorf("create tmpdir to mount image to dir %s", mntdir)
	}
	lowerRootDir := filepath.Join(tmpdir, "lower")
	if err := os.MkdirAll(lowerRootDir, 0755); err != nil {
		return nil, fmt.Errorf("create lowerRootDir %s: %w", lowerRootDir, err)
	}
	upperdir := filepath.Join(tmpdir, "upper")
	if err := os.MkdirAll(upperdir, 0755); err != nil {
		return nil, fmt.Errorf("create upperdir %s: %w", upperdir, err)
	}
	workdir := filepath.Join(tmpdir, "work")
	if err := os.MkdirAll(workdir, 0755); err != nil {
		return nil, fmt.Errorf("create workdir %s: %w", workdir, err)
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("get layers: %w", err)
	}
	lowerdirs := make([]string, len(layers))
	for i, lyr := range layers {
		dir, err := decompressLayerTo(lyr, lowerRootDir)
		if err != nil {
			return nil, fmt.Errorf("uncompress %dth layer: %w", i+1, err)
		}
		lowerdirs[len(layers)-i-1] = dir
	}
	um := &imageUmount{
		mntdir: mntdir,
		tmpdir: tmpdir,
		errCh:  make(chan error),
	}
	go func() {
		um.errCh <- fs.MountOverlay(lowerdirs, upperdir, workdir, mntdir, fs.SupportOverlayIndexOff())
	}()

	pollTicker := time.NewTicker(mountPollInterval)
	pollTimeout := time.After(mountPollTimeout)
	defer pollTicker.Stop()
	for {
		select {
		case <-pollTicker.C:
			if !fs.IsEmptyDir(mntdir) {
				return um, nil
			}
		case <-pollTimeout:
			log.C(ctx).With("path", mntdir).Warn("overlay mount image layers timeout")
			return nil, um.Umount()
		case err := <-um.errCh:
			if err != nil {
				if uerr := um.Umount(); uerr != nil {
					log.C(ctx).With("path", mntdir).WithError(err).Warn("umount a mount-error directory")
				}
				return nil, err
			}
		}
	}
}

type imageUmount struct {
	mntdir string
	tmpdir string
	errCh chan error
}

func (u *imageUmount) MountErrChan() chan error {
	return u.errCh
}

func (u *imageUmount) Umount() error {
	if err := fs.UmountForceDetach(u.mntdir); err != nil {
		return err
	}
	if err := os.RemoveAll(u.tmpdir); err != nil {
		return fmt.Errorf("rmdir %s: %w", u.tmpdir, err)
	}
	close(u.errCh)
	return nil
}

func decompressLayerTo(lyr v1.Layer, pdir string) (string, error) {
	digest, err := lyr.Digest()
	if err != nil {
		return "", fmt.Errorf("get digest from layer: %w", err)
	}
	hex := digest.Hex
	if len(hex) > 16 {
		hex = hex[:16]
	}
	dir := filepath.Join(pdir, hex)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	lr, err := lyr.Uncompressed()
	if err != nil {
		return "", fmt.Errorf("get layer reader: %w", err)
	}
	defer func() {
		if cerr := lr.Close(); cerr != nil {
			log.WithError(cerr).Warn("close layer reader")
		}
	}()
	if err := tar.UnpackWithOwnerAndTimes(stdtar.NewReader(lr), dir); err != nil {
		return "", fmt.Errorf("unpack layer tar: %w", err)
	}
	return dir, nil
}
