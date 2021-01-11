package main

import (
	"flag"

	"github.com/sequix/casync-snapshotter/pkg/buildinfo"
	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/util"
)

func main() {
	flag.Parse()
	buildinfo.Init()
	stop := util.SetupSignalHandler()

	log.Init("casync-snapshotter")
	log.G.Info("inited logger")

	stop.Wait()
	log.G.Info("recv stop signal")
}
