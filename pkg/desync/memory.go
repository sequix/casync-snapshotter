package desync

import (
	"fmt"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/folbricht/desync"

	"github.com/sequix/dequash-snapshotter/pkg/log"
)

// TODO metrics
type memoryChunkCacheStore struct {
	store desync.WriteStore
	cache *fastcache.Cache
}

func newMemoryCacheStore(maxBytes int, chkStore desync.WriteStore) desync.WriteStore {
	mcs := &memoryChunkCacheStore{
		store: chkStore,
		cache: fastcache.New(maxBytes),
	}
	return mcs
}

func (s *memoryChunkCacheStore) GetChunk(id desync.ChunkID) (*desync.Chunk, error) {
	idk := []byte(id.String())

	// fast path
	chunkBytes := s.cache.GetBig(nil, idk)
	if len(chunkBytes) > 0 {
		return desync.NewChunkFromUncompressed(chunkBytes), nil
	}

	// slow path
	chunk, err := s.store.GetChunk(id)
	if err != nil {
		return nil, fmt.Errorf("get chunk %q: %w", id.String(), err)
	}

	// cache write back
	go func() {
		// TODO chunk uncompressed thread-safe?
		chunkBytes2, err := chunk.Uncompressed()
		if err != nil {
			log.WithError(err).With("chkid", id.String()).Warn("uncompress chunk")
			return
		}
		s.cache.SetBig(idk, chunkBytes2)
	}()
	return chunk, nil
}

func (s *memoryChunkCacheStore) HasChunk(id desync.ChunkID) (bool, error) {
	return s.store.HasChunk(id)
}

func (s *memoryChunkCacheStore) Close() error {
	return s.store.Close()
}

func (s *memoryChunkCacheStore) String() string {
	return "memcache-" + s.store.String()
}

func (s *memoryChunkCacheStore) StoreChunk(c *desync.Chunk) error {
	return s.store.StoreChunk(c)
}