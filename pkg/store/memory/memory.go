package memory

import (
	"flag"

	"github.com/VictoriaMetrics/fastcache"

	"github.com/sequix/casync-snapshotter/pkg/store"
	"github.com/sequix/casync-snapshotter/pkg/util"
)

var (
	flagCacheSize = flag.Int("memcache-size-mib", 1024, "memory cache size in MiB")
)

type MemCache struct {
	fc *fastcache.Cache
}

func New() *MemCache {
	mc := &MemCache{
		fc: fastcache.New(*flagCacheSize * 1024 * 1024),
	}

	// TODO fscache metrics
	//mc.fc.UpdateStats()

	return mc
}

func (m *MemCache) AddChunk(key string, src []byte) error {
	m.fc.SetBig(util.ToUnsafeBytes(key), src)
	return nil
}

func (m *MemCache) GetChunk(key string, dst []byte) ([]byte, error) {
	dst = m.fc.GetBig(dst, util.ToUnsafeBytes(key))
	if len(dst) > 0 {
		return dst, nil
	}
	return dst, store.ErrNotFound
}
