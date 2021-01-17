package main

import (
	"flag"
	"net/http"
	"strings"
	"sync"

	"github.com/sequix/casync-snapshotter/pkg/buildinfo"
	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/server"
	"github.com/sequix/casync-snapshotter/pkg/store"
	"github.com/sequix/casync-snapshotter/pkg/store/chainstore"
	"github.com/sequix/casync-snapshotter/pkg/store/fsstore"
	"github.com/sequix/casync-snapshotter/pkg/store/memstore"
	"github.com/sequix/casync-snapshotter/pkg/store/p2pstore"
	"github.com/sequix/casync-snapshotter/pkg/store/s3store"
	"github.com/sequix/casync-snapshotter/pkg/util"
)

var (
	cache store.Interface
)

func main() {
	flag.Parse()
	buildinfo.Init()
	log.Init("casnap-cnksvr")
	stop := util.SetupSignalHandler()

	cache = chainstore.New(memstore.New(), fsstore.New(stop), p2pstore.New(), s3store.New())
	log.G.Info("inited cache")

	server.Init()
	server.Register("/", http.HandlerFunc(handler))
	serverRw := util.Run(server.Run)
	log.G.Info("server inited")

	stop.Wait()
	log.G.Info("recv stop signal")

	serverRw.StopAndWait()
	log.G.Info("server finished")
}

var bytesBufferPool = &sync.Pool{}

func getBytesBuffer() []byte {
	v := bytesBufferPool.Get()
	if v == nil {
		return make([]byte, 0, 16 * 1024)
	}
	return v.([]byte)[:0]
}

func putBytesBuffer(v []byte) {
	bytesBufferPool.Put(v)
}

func handler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")

	if len(pathParts) != 3 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	key := pathParts[2]
	if !strings.HasSuffix(key, ".cacnk") {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	rst := getBytesBuffer()
	defer putBytesBuffer(rst)

	rst, err := cache.GetChunk(key, rst)
	if err != nil {
		log.G.WithError(err).Errorf("get chunk %s", key)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	if written, err := w.Write(rst); err != nil {
		log.G.WithError(err).Errorf("write chunk %s , written %d bytes", key, written)
	}
}