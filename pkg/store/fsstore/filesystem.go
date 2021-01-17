package fsstore

import (
	"flag"
	"time"

	"github.com/sequix/fscache"

	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/store"
	"github.com/sequix/casync-snapshotter/pkg/util"
)

var (
	flagRoot          = flag.String("fscache-root", "", "path to filesystem cache")
	flagMaxBytes      = flag.Int64("fscache-max-bytes", 1024, "max size of filesystem cache in MiB")
	flagCleanInterval = flag.Duration("fscache-clean-interval", 5*time.Minute, "interval between each clean run")
)

type FsCache struct {
	cache fscache.Interface
}

func New(stop util.BroadcastCh) *FsCache {
	c, err := fscache.New(
		fscache.WithGcStopCh(stop),
		fscache.WithGcInterval(*flagCleanInterval),
		fscache.WithCacheDir(*flagRoot),
		fscache.WithMaxBytes(*flagMaxBytes),
	)
	if err != nil {
		log.G.Fatal(err)
	}
	return &FsCache{cache: c}
}

func (f *FsCache) AddChunk(key string, src []byte) error {
	return f.cache.Set(key, src)
}

func (f *FsCache) GetChunk(key string, dst []byte) ([]byte, error) {
	var err error
	dst, err = f.cache.Get(key, dst)
	if err == fscache.ErrNotFound {
		err = store.ErrNotFound
	}
	return dst, err
}
