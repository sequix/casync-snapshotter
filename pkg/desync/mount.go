package desync

import (
	"context"
	"fmt"
	"time"

	"github.com/folbricht/desync"

	"github.com/sequix/dequash-snapshotter/pkg/fs"
	"github.com/sequix/dequash-snapshotter/pkg/log"
)

const (
	// desync-index mount requires a little bit time to populate all files in the destination directory.
	// We list the directory periodically until all files a populated.
	mountPollInterval = 1 * time.Second
	// If files were not populated within this time, return an error.
	mountPollTimeout = 10 * time.Second
	//
	IndexFilename = "squash"
)

func Mount(ctx context.Context, idxName, mntdir string) (fs.Umount, error) {
	if fs.IsPathExist(mntdir) && !fs.IsEmptyDir(mntdir) {
		return nil, fmt.Errorf("expected %q is not existing or an empty dir", mntdir)
	}
	idx, err := indexStore.GetIndex(idxName)
	if err != nil {
		return nil, fmt.Errorf("get index: %w", err)
	}
	um := &umount{
		mntdir: mntdir,
		errCh: make(chan error),
	}
	go func() {
		err := desync.MountIndex(ctx, idx, mntdir, IndexFilename, chunkStore, *optThreads)
		if err != nil {
			um.errCh <- fmt.Errorf("mount desync index: %w", err)
		}
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
			log.C(ctx).With("path", mntdir).Warn("desync-index mount timeout")
			return nil, um.Umount()
		case err := <-um.errCh:
			if err != nil {
				if uerr := um.Umount(); uerr != nil {
					log.With("path", mntdir).WithError(err).Warn("umount a mount-error directory")
				}
				return nil, err
			}
		}
	}
}

type umount struct {
	mntdir string
	errCh chan error
}

func (u *umount) MountErrChan() chan error {
	return u.errCh
}

func (u *umount) Umount() error {
	err := fs.UmountForceDetach(u.mntdir)
	close(u.errCh)
	return err
}
