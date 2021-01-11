package main

import (
	"flag"
	"net"
	"os"
	"path/filepath"

	snapshotsapi "github.com/containerd/containerd/api/services/snapshots/v1"
	"github.com/containerd/containerd/contrib/snapshotservice"
	"google.golang.org/grpc"

	"github.com/sequix/dequash-snapshotter/pkg/desync"
	"github.com/sequix/dequash-snapshotter/pkg/log"
	snapshotter "github.com/sequix/dequash-snapshotter/pkg/overlay"
	"github.com/sequix/dequash-snapshotter/pkg/signal"
	"github.com/sequix/dequash-snapshotter/pkg/squashfs"
)

// TODO snapshotter: docker-env demo
// TODO private registry k8s private registry
// TODO unit test, more test
// TODO k8s p2p
// TODO 一个镜像被mount多次，那么desync mount和losetup只需要执行一次
// TODO loop设备最多多少个？

const (
	defaultAddress  = "/run/containerd-dequash-grpc/socket"
	defaultRootDir = "/var/lib/containerd-dequash-grpc"
)

var (
	address  = flag.String("ss-address", defaultAddress, "socket address for the snapshotter's GRPC server")
	rootDir = flag.String("ss-root", defaultRootDir, "path to the root directory for this snapshotter")
	asyncRemove = flag.Bool("ss-asyncRemove", false, "defer snapshotter removal of filesystem content")
)

func main() {
	flag.Parse()
	log.Init("snapshotter")
	desync.Init()
	squashfs.Init()

	// Create snapshotter instance
	rsOpts := []snapshotter.Opt{}
	if *asyncRemove {
		rsOpts = append(rsOpts, snapshotter.AsynchronousRemove)
	}
	rs, err := snapshotter.NewSnapshotter(*rootDir, rsOpts...)
	if err != nil {
		log.WithError(err).Fatal("create dequash snapshotter")
	}
	defer func() {
		if err := rs.Close(); err != nil {
			log.WithError(err).Fatal("close dequash snapshotter")
		}
	}()

	rpc := grpc.NewServer()
	service := snapshotservice.FromSnapshotter(rs)
	snapshotsapi.RegisterSnapshotsServer(rpc, service)

	// Prepare the directory for the socket
	if err := os.MkdirAll(filepath.Dir(*address), 0700); err != nil {
		log.WithError(err).With("path", filepath.Dir(*address)).
			Fatalf("create directory snapshotter root dir")
	}

	// Try to remove the socket file to avoid EADDRINUSE
	if err := os.RemoveAll(*address); err != nil {
		log.WithError(err).With("path", *address).Fatal("remove snapshotter socket")
	}

	// Listen and serve
	l, err := net.Listen("unix", *address)
	if err != nil {
		log.WithError(err).With("path", *address).Fatalf("listen snapshotter grpc socket")
	}
	go func() {
		if err := rpc.Serve(l); err != nil {
			log.WithError(err).With("path", *address).Fatalf("serving grpc via socket")
		}
	}()
	log.Info("dequash snapshotter started")
	<-signal.SetupSignalHandler()
}