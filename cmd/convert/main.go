package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/sequix/casync-snapshotter/pkg/buildinfo"
	"github.com/sequix/casync-snapshotter/pkg/casync"
	"github.com/sequix/casync-snapshotter/pkg/fs"
	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/oci"
	"github.com/sequix/casync-snapshotter/pkg/store/s3"
)

const helpMsg = `usage: casnap-conv [options] <image> <seedImage>
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

var (
	flagTempDir       = flag.String("tmpdir", os.TempDir(), "path to hold all temp files")
	flagUploadThreads = flag.Int("uploadThreads", 8, "upload threads")
)

func main() {
	flag.Parse()
	buildinfo.Init()
	log.Init("casnap-conv")

	if flag.NArg() != 2 {
		fmt.Print(helpMsg)
		os.Exit(1)
	}

	var (
		imageName     = flag.Arg(0)
		seedImageName = flag.Arg(1)
	)

	store, err := s3.New()
	if err != nil {
		log.G.WithError(err).Errorf("init s3 client")
		return
	}
	casync.Init()

	// 1.pull image
	image, err := oci.Pull(imageName)
	if err != nil {
		log.G.WithError(err).Error("pull image")
		return
	}
	log.G.Infof("pulled image %s", imageName)

	// 2.mount image
	mntdir, mntdirClean := fs.TempDir(*flagTempDir, "casnap-mnt-")
	defer mntdirClean()
	mntmpDir, mntmpDirClean := fs.TempDir(*flagTempDir, "casnap-mntmp-")
	defer mntmpDirClean()
	umount, err := oci.Mount(image, mntdir, mntmpDir)
	if err != nil {
		log.G.WithError(err).Error("mount image")
		return
	}
	defer umount()
	log.G.Infof("mounted image to %s", mntdir)

	// 3.make and upload seed image to registry
	seed, err := image.Digest()
	if err != nil {
		log.G.WithError(err).Error("digest image")
		return
	}
	diffId, err := oci.PushSeedImage(seedImageName, seed.Hex)
	if err != nil {
		log.G.WithError(err).Error("push seed image")
		return
	}
	log.G.Infof("pushed image seed")

	// 4.make casync archive
	castrDir, castrDirClean := fs.TempDir(*flagTempDir, "casnap-castr-")
	defer castrDirClean()
	caidxFilepath, caidxClean := fs.TempFile(*flagTempDir, "casnap-caidx-*.caidx")
	defer caidxClean()
	if err := casync.Make(mntdir, castrDir, caidxFilepath); err != nil {
		log.G.WithError(err).Error("make casync archive")
		return
	}
	log.G.Info("made casync archive")

	// 5.upload caidx
	caidxFile, err := ioutil.ReadFile(caidxFilepath)
	if err != nil {
		log.G.WithError(err).Errorf("read caidx file %s", caidxFilepath)
		return
	}
	if err := store.AddChunk("caidx/"+diffId+".caidx", caidxFile); err != nil {
		log.G.WithError(err).Errorf("upload caidx file %s", caidxFilepath)
		return
	}
	log.G.Info("uploaded caidx")

	// 6.upload cacnk
	log.G.Info("uploading cacnk...")
	uploadCh := make(chan string)
	defer close(uploadCh)

	go func() {
		err = filepath.Walk(castrDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.G.WithError(err).Errorf("walk castr dir")
				return err
			}
			if !info.IsDir() {
				uploadCh <- path
			}
			return nil
		})
		if err != nil {
			log.G.WithError(err).Errorf("bad walk")
			return
		}
		for i := 0; i < *flagUploadThreads; i++ {
			uploadCh <- ""
		}
	}()

	uploaderWg := &sync.WaitGroup{}
	uploaderWg.Add(*flagUploadThreads)
	for i := 0; i < *flagUploadThreads; i++ {
		go func() {
			for {
				path := <-uploadCh

				if len(path) == 0 {
					uploaderWg.Done()
					return
				}
				cacnkFile, err := ioutil.ReadFile(path)
				if err != nil {
					log.G.WithError(err).Errorf("read cacnk file %s", path)
					os.Exit(1)
				}
				cacnkKey := filepath.Join("castr", path[len(castrDir):])
				if err := store.AddChunk(cacnkKey, cacnkFile); err != nil {
					log.G.WithError(err).Errorf("upload cacnk file %s", path)
					os.Exit(1)
				}
				log.G.Debugf("uploaded %s", cacnkKey)
			}
		}()
	}
	uploaderWg.Wait()
	log.G.Info("uploaded cacnk")
}
