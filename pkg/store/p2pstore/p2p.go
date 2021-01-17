package p2pstore

import "github.com/sequix/casync-snapshotter/pkg/store"

type P2pCache struct {

}

func New() *P2pCache {
	return &P2pCache{}
}

func (p *P2pCache) AddChunk(key string, src []byte) error {
	return store.ErrNotSupport
}

func (p *P2pCache) GetChunk(key string, dst []byte) ([]byte, error) {
	// TODO: p2p
	return dst, store.ErrNotSupport
}
