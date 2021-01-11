package filesystem

import (
	"container/heap"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/gofrs/flock"

	"github.com/sequix/casync-snapshotter/pkg/fs"
	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/util"
)

var (
	flagRoot          = flag.String("fscache-root", "", "path to filesystem cache")
	flagMaxBytes      = flag.Int64("fscache-max-bytes", 1024, "max size of filesystem cache in MiB")
	flagCleanInterval = flag.Duration("fscache-clean-interval", 5*time.Minute, "interval between each clean run")
)

type FsCache struct {
	root          string
	tmp           string
	cleanInterval time.Duration
	maxBytes      int64
	totalBytes    int64
	fisByAtime    *fileInfoHeap
	fisByAtimeMu  sync.RWMutex
	flock         *flock.Flock
}

func New(stop util.BroadcastCh) (*FsCache, error) {
	if err := os.MkdirAll(*flagRoot, 0775); err != nil {
		return nil, fmt.Errorf("create root dir %s: %w", *flagRoot, err)
	}

	var (
		fisByAtime = &fileInfoHeap{}
		totalBytes int64
	)

	err := filepath.Walk(*flagRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			heap.Push(fisByAtime, info)
			totalBytes += info.Size()
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk dir %s: %w", *flagRoot, err)
	}

	fc := &FsCache{
		root:          *flagRoot,
		tmp:           filepath.Join(*flagRoot, ".tmp"),
		cleanInterval: *flagCleanInterval,
		maxBytes:      *flagMaxBytes,
		totalBytes:    totalBytes,
		fisByAtime:    fisByAtime,
		fisByAtimeMu:  sync.RWMutex{},
		flock:         flock.New(filepath.Join(*flagRoot, ".flock")),
	}
	return fc, nil
}

func (f *FsCache) Run(stop util.BroadcastCh) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(f.cleanInterval):
			continue
		}
	}
}

func (f *FsCache) rflock() {
	if err := f.flock.RLock(); err != nil {
		log.G.WithError(err).Fatal("rlock filesystem cache flock")
	}
}

func (f *FsCache) uflock() {
	if err := f.flock.Unlock(); err != nil {
		log.G.WithError(err).Fatal("ulock filesystem cache flock")
	}
}

func (f *FsCache) wflock() {
	if err := f.flock.Lock(); err != nil {
		log.G.WithError(err).Fatal("lock filesystem cache flock")
	}
}

func (f *FsCache) tmpPathFromKey(key string) string {
	return filepath.Join(f.tmp, strings.ReplaceAll(key, string(os.PathSeparator), "_"))
}

func (f *FsCache) pathFromKey(key string) string {
	return filepath.Join(f.root, strings.ReplaceAll(key, string(os.PathSeparator), "_"))
}

func (f *FsCache) AddChunk(key string, src []byte) error {
	path := f.pathFromKey(key)

	if fs.IsPathExist(path) {
		return nil
	}

	tmpPath := f.tmpPathFromKey(key)
	ioutils.AtomicWriteFile()
}

func (f *FsCache) GetChunk(key string, dst []byte) ([]byte, error) {
	panic("implement me")
}

type fileInfoHeap []os.FileInfo

func (f fileInfoHeap) Len() int {
	return len(f)
}

func (f fileInfoHeap) Less(i, j int) bool {
	fis := f[i].Sys().(*syscall.Stat_t)
	fjs := f[j].Sys().(*syscall.Stat_t)
	fia := time.Unix(fis.Atim.Sec, fis.Atim.Nsec)
	fja := time.Unix(fjs.Atim.Sec, fjs.Atim.Nsec)
	return fia.Before(fja)
}

func (f fileInfoHeap) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f *fileInfoHeap) Push(x interface{}) {
	*f = append(*f, x.(os.FileInfo))
}

func (f *fileInfoHeap) Pop() interface{} {
	old := *f
	n := len(old)
	x := old[n-1]
	*f = old[0 : n-1]
	return x
}
