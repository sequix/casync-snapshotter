package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sequix/dequash-snapshotter/pkg/desync"
	"github.com/sequix/dequash-snapshotter/pkg/fs"
	"github.com/sequix/dequash-snapshotter/pkg/image"
	"github.com/sequix/dequash-snapshotter/pkg/log"
	"github.com/sequix/dequash-snapshotter/pkg/signal"
	"github.com/sequix/dequash-snapshotter/pkg/squashfs"
)

const helpMsg = `usage: dequash-mount [options] <seedImage> <mountDirectory>
  [options]
    check details with --help

  <seedImage>      specify the seed image
  <mountDirectory> mount point

  <seedImage> of them support the following format:

    [http://|https://][<username>:<password>@]<imageReference>
      [http://|https://]       optional, skip TLS verification or not for registry, https by default
      [<username>:<password>@] optional, registry username and password
      <imageReference>         mandatory, registry image reference

    docker.io/library/nginx:1.18.0-alpine
      pull image from registry
    
    http://test:pass@127.0.0.1:5000/nginx:1.18-edited
      pull image from local registry with username 'test' and password 'pass'
`

func main() {
	flag.Parse()
	log.Init("mount")
	squashfs.Init()
	desync.Init()

	if flag.NArg() != 2 {
		fmt.Print(helpMsg)
		os.Exit(1)
	}
	var (
		stopCh        = signal.SetupSignalHandler()
		ctx           = log.InjectIntoContext(context.Background())
		seedImageName = flag.Arg(0)
		mntdir        = flag.Arg(1)
	)
	if fs.IsPathExist(mntdir) && !fs.IsEmptyDir(mntdir) {
		log.C(ctx).With("path", mntdir).Fatal("expected <mountDirectory> is not existing or an empty directory")
	}

	// 1.pull seed image
	indexName, err := image.PullSeedImage(seedImageName)
	if err != nil {
		log.C(ctx).WithError(err).With("image", seedImageName).Fatal("pull seed image")
	}
	log.C(ctx).With("image", seedImageName).Info("pulled seed image")

	// 2.mount desync index
	desyncMntdir, err := ioutil.TempDir("", "dequash-deidx-")
	if err != nil {
		log.C(ctx).WithError(err).Fatal("create temp dir for desync index mount")
	}
	desyncUmount, err := desync.Mount(ctx, indexName, desyncMntdir, )
	if err != nil {
		log.C(ctx).WithError(err).With("path", desyncMntdir).Fatal("mount desync index")
	}
	go func() {
		if err := <-desyncUmount.MountErrChan(); err != nil {
			log.C(ctx).WithError(err).With("path", desyncMntdir).Fatal("mount desync index")
		}
	}()
	defer func() {
		if err := desyncUmount.Umount(); err != nil {
			log.C(ctx).WithError(err).With("path", desyncMntdir).Fatal("umount desync index")
		}
		if err := os.RemoveAll(desyncMntdir); err != nil {
			log.C(ctx).WithError(err).With("path", desyncMntdir).Fatal("rm desync mntdir")
		}
	}()
	log.C(ctx).With("path", desyncMntdir).Info("mounted desync index")

	// 3.attach squashfs to loop device
	squashFile := filepath.Join(desyncMntdir, desync.IndexFilename)
	loopdev, err := squashfs.LosetupAttach(ctx, squashFile)
	if err != nil {
		log.C(ctx).WithError(err).With("path", squashFile).Fatal("attach squashfs to loop device")
	}
	defer func() {
		if err := squashfs.LosetupDetach(ctx, loopdev); err != nil {
			log.C(ctx).WithError(err).With("path", squashFile).Fatal("detach squashfs loop device")
		}
	}()
	log.C(ctx).With("path", squashFile).With("device", loopdev).Info("attach squashfs to loop device")

	// 4.mount squashfs loop device
	squashUmount, err := squashfs.Mount(loopdev, mntdir)
	if err != nil {
		log.C(ctx).WithError(err).With("path", mntdir).Fatal("mount squashfs")
	}
	go func() {
		if err := <-squashUmount.MountErrChan(); err != nil {
			log.C(ctx).WithError(err).With("path", mntdir).Fatal("mount squashfs")
		}
	}()
	defer func() {
		if err := squashUmount.Umount(); err != nil {
			log.C(ctx).WithError(err).With("path", mntdir).Fatal("umount squashfs")
		}
	}()
	log.C(ctx).With("path", mntdir).Info("mounted squashfs")

	<-stopCh
}
