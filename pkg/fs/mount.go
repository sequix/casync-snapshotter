package fs

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// The first directory in `lowerdirs` is the top layer, the last one is the bottom.
// If mount successfully, block until UmountForceDetach() is called, return error if otherwise.
func MountOverlay(lowerdirs []string, upperdir, workdir, mountdir string, indexOff bool) error {
	if len(lowerdirs) == 0 {
		return errors.New("need at least lowerdir")
	}
	for _, ld := range lowerdirs {
		if !IsPathExist(ld) {
			return fmt.Errorf("expected existing lowerdir, not %s", ld)
		}
	}
	if len(upperdir) == 0 {
		return errors.New("need a upperdir")
	}
	if !IsEmptyDir(upperdir) {
		return fmt.Errorf("expected upperdir %s is a empty dir", upperdir)
	}
	if len(mountdir) == 0 {
		return errors.New("need a target dir")
	}
	if !IsEmptyDir(mountdir) {
		return fmt.Errorf("expected target dir %s is a empty dir", mountdir)
	}
	if len(workdir) == 0 {
		return errors.New("need a workdir")
	}
	if !IsEmptyDir(workdir) {
		return fmt.Errorf("expected workdir %s is a empty dir", workdir)
	}
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		strings.Join(lowerdirs, ":"), upperdir, workdir)
	if indexOff {
		opts += ",index=off"
	}
	if err := syscall.Mount("overlay", mountdir, "overlay", 0, opts); err != nil {
		return fmt.Errorf("mount -t overlay -o %s overlay %s: %s", opts, mountdir, err)
	}
	return nil
}

type UmountFunc func()

func UmountForceDetach(dir string) error {
	if len(dir) == 0 {
		return errors.New("unmount need a dir")
	}
	if !IsPathExist(dir) {
		return fmt.Errorf("unmount expected dir %s is existing", dir)
	}
	if err := syscall.Unmount(dir, syscall.MNT_DETACH|syscall.MNT_FORCE); err != nil {
		return fmt.Errorf("unmount %s: %w", dir, err)
	}
	return nil
}

// Figure out whether "index=off" option is recognized by the kernel
// see also: https://github.com/containerd/containerd/pull/4311
func SupportOverlayIndexOff() bool {
	if _, err := os.Stat("/sys/module/overlay/parameters/index"); err == nil {
		return true
	}
	return false
}
