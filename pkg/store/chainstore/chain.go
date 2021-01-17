package chainstore

import (
	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/store"
)

type Chain struct {
	stores []store.Interface
}

func New(ss ...store.Interface) *Chain {
	return &Chain{
		stores: ss,
	}
}

func (c *Chain) AddChunk(key string, src []byte) error {
	return store.ErrNotSupport
}

func (c *Chain) GetChunk(key string, dst []byte) ([]byte, error) {
	var (
		err     error
		foundAt = -1
	)

	for i, s := range c.stores {
		dst, err = s.GetChunk(key, dst)
		if err == nil {
			foundAt = i
			break
		}
	}

	if foundAt == -1 {
		return dst, store.ErrNotFound
	}

	for i := 0; i < foundAt; i++ {
		go func(s store.Interface) {
			if err := s.AddChunk(key, dst); err != nil && err != store.ErrNotSupport {
				log.G.WithError(err).Errorf("add chunk %s", key)
			}
		}(c.stores[i])
	}
	return dst, nil
}
