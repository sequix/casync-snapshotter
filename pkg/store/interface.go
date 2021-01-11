package store

import (
	"errors"
)

// Interface represents a general api of object store.
type Interface interface {
	AddChunk(key string, src []byte) error
	GetChunk(key string, dst []byte) ([]byte, error)
}

var (
	ErrNotFound = errors.New("no such chunk")
	ErrNotSupport = errors.New("not support such func")
)