package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sequix/dequash-snapshotter/pkg/desync"
	"github.com/sequix/dequash-snapshotter/pkg/fs"
	"github.com/sequix/dequash-snapshotter/pkg/image"
	"github.com/sequix/dequash-snapshotter/pkg/log"
	"github.com/sequix/dequash-snapshotter/pkg/signal"
	"github.com/sequix/dequash-snapshotter/pkg/squashfs"
)

const helpMsg = `usage: dequash-convert [options] <image> <seedImage>
  [options]
    check details with --help

  <image>     specify the image you want to convert
  <seedImage> specify where the seed image

  both of <image> and <seedImage> support the following format: 

    [registry:|tarball:][http://|https://][<username>:<password>@]<imageReference|tarballPath>
      [registry:|tarball:]         optional, pull image from registry or tarball file, registry by default
      [http://|https://]           optional, skip TLS verification or not for registry, https by default
      [<username>:<password>@]     optional, registry username and password
      <imageReference|tarballPath> mandatory, registry image reference or tarball filepath

    docker.io/library/nginx:1.18.0-alpine
      pull image from registry
    
    http://test:pass@127.0.0.1:5000/nginx:1.18-edited
      pull image from local registry with username 'test' and password 'pass'

    tarball:nginx.tar
      pull image from local file ./nginx.tar, you can use 'crane' or 'docker save' to get a tar file
`

func main() {
	flag.Parse()
	log.Init("convert")
	squashfs.Init()
	desync.Init()
	go func() { <-signal.SetupSignalHandler() }()

	if flag.NArg() != 2 {
		fmt.Print(helpMsg)
		os.Exit(1)
	}
	ctx := log.InjectIntoContext(context.Background())
	imageName := flag.Arg(0)
	seedImageName := flag.Arg(1)

	// 1.pull image
	img, err := image.Pull(imageName)
	if err != nil {
		log.C(ctx).WithError(err).With("image", imageName).Fatal("pull image")
	}
	log.C(ctx).Infof("pulled image %s", imageName)

	// 2.mount image
	mntdir, err := ioutil.TempDir("", "dequash-mnt-")
	if err != nil {
		log.C(ctx).WithError(err).Fatal("create mntdir to mount image")
	}
	umount, err := image.Mount(ctx, img, mntdir)
	if err != nil {
		log.C(ctx).WithError(err).With("image", imageName).Fatal("mount image")
	}
	defer func() {
		if err := umount.Umount(); err != nil {
			log.C(ctx).WithError(err).With("image", imageName).Error("umount image")
		}
		if err := os.RemoveAll(mntdir); err != nil {
			log.C(ctx).WithError(err).With("path", mntdir).Error("remove dir")
		}
	}()
	log.C(ctx).Infof("mounted image to %s", mntdir)

	// 3.mksquashfs
	squashFile := fs.TempFilename("dequash-squash-")
	// N.B. `-nopad` option will cause the final squash file in-mountable.
	squashOpts := []string{"-noI", "-noD", "-noF", "-noX", "-no-fragments", "-no-duplicates"}
	if err := squashfs.Make(ctx, mntdir, squashFile, squashOpts...); err != nil {
		log.C(ctx).WithError(err).Fatal("mksquashfs")
	}
	defer func() {
		if err := os.Remove(squashFile); err != nil {
			log.C(ctx).WithError(err).With("path", squashFile).Error("rm file")
		}
	}()
	log.C(ctx).Infof("made a squashfs at %s", squashFile)

	// 4.push seed image and get diffID of first layer
	seed, err := img.Digest()
	if err != nil {
		log.C(ctx).WithError(err).With("image", imageName).Fatal("digest image")
	}
	indexName, err := image.PushSeedImage(seedImageName, seed.Hex)
	if err != nil {
		log.C(ctx).WithError(err).With("image", seedImageName).Fatal("push seed image")
	}
	log.C(ctx).Infof("pushed image seed")

	// 5.desync
	if err := desync.ChopFile(ctx, squashFile, indexName); err != nil {
		log.WithError(err).Fatal("push desync chunk")
	}
	log.C(ctx).Info("done")
}
