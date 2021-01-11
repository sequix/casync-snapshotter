// +build linux

package squashfs

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"syscall"

	"github.com/sequix/dequash-snapshotter/pkg/fs"
	"github.com/sequix/dequash-snapshotter/pkg/log"
)

var (
	makeBin    = flag.String("bin-squash", "", "path to mksquashfs binary")
	losetupBin = flag.String("bin-losetup", "", "path to losetup binary")
)

func Init() {
	if len(*makeBin) == 0 {
		bin, err := exec.LookPath("mksquashfs")
		if err != nil {
			log.WithError(err).Fatal("lookup mksquashfs binary failed, you can specify it with flag -bin-squash")
		}
		*makeBin = bin
	}
	if !fs.IsExecutable(*makeBin) {
		log.With("path", *makeBin).Fatal("not a executable")
	}
	if len(*losetupBin) == 0 {
		bin, err := exec.LookPath("losetup")
		if err != nil {
			log.WithError(err).Fatal("lookup losetup binary failed, you can specify it with flag -bin-losetup")
		}
		*losetupBin = bin
	}
	if !fs.IsExecutable(*losetupBin) {
		log.With("path", *losetupBin).Fatal("not a executable")
	}
}

func execute(ctx context.Context, command string, args []string) (stdout, stderr []byte, err error) {
	cmdline := strings.Join(append([]string{command}, args...), " ")
	cmd := exec.Command(command, args...)
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stdout pipe of %q: %w", cmdline, err)
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get stderr pipe of %q: %w", cmdline, err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start command %q: %w", cmdline, err)
	}
	stdout, err = ioutil.ReadAll(outPipe)
	if err != nil {
		log.C(ctx).WithError(err).With("cmd", cmdline).With("read stdout pipe")
	}
	stderr, err = ioutil.ReadAll(errPipe)
	if err != nil {
		log.C(ctx).WithError(err).With("cmd", cmdline).Warn("read stderr pipe")
	}
	if err := cmd.Wait(); err != nil {
		err = fmt.Errorf("exec %q failed: err %w stderr %s, stdout %s",
			cmdline, err, string(stdout), string(stderr))
	}
	return
}

func Make(ctx context.Context, src, dst string, opts ...string) error {
	args := make([]string, 0, len(opts)+2)
	args = append(args, src, dst)
	args = append(args, opts...)
	_, _, err := execute(ctx, *makeBin, args)
	return err
}

// LosetupAttach binds squashFile to the first unused loop device,
// and returns the loop device name.
// If no more loop device available, it will create new loop device.
func LosetupAttach(ctx context.Context, squashFile string) (string, error) {
	args := []string{"--find", "--show", "--nooverlap", "--read-only", squashFile}
	stdout, stderr, err := execute(ctx, *losetupBin, args)
	if len(stderr) != 0 {
		err = fmt.Errorf("losetup attach return stderr: %s", string(stderr))
	}
	return strings.TrimSpace(string(stdout)), err
}

func LosetupDetach(ctx context.Context, loopdev string) error {
	args := []string{"-d", loopdev}
	_, stderr, err := execute(ctx, *losetupBin, args)
	if len(stderr) != 0 {
		err = fmt.Errorf("losetup detach return stderr: %s", string(stderr))
	}
	return err
}

func Mount(src, dst string) (fs.Umount, error) {
	if !fs.IsLoopDevice(src) {
		return nil, fmt.Errorf("expected %q is a loop device", src)
	}
	if fs.IsPathExist(dst) && !fs.IsEmptyDir(dst) {
		return nil, fmt.Errorf("expected %q is an empty directory or not existing", dst)
	}
	um := &umount{
		mntdir: dst,
		errCh:  make(chan error),
	}
	go func() {
		err := syscall.Mount(src, dst, "squashfs", syscall.MS_RDONLY, "")
		if err != nil {
			um.errCh <- fmt.Errorf("mount -rt squashfs %s %s: %w", src, dst, err)
		}
	}()
	return um, nil
}

type umount struct {
	mntdir  string
	errCh   chan error
}

func (u *umount) MountErrChan() chan error {
	return u.errCh
}

func (u *umount) Umount() error {
	if err := fs.UmountForceDetach(u.mntdir); err != nil {
		return fmt.Errorf("unmount %q associated: %w", u.mntdir, err)
	}
	close(u.errCh)
	return nil
}
