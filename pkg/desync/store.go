package desync

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/folbricht/desync"
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/credentials"

	"github.com/sequix/dequash-snapshotter/pkg/log"
)

// Architecture and/or implementation specific integer limits and bit widths.
const (
	maxInt  = 1<<(intBits-1) - 1
	intBits = 1 << (^uint(0)>>32&1 + ^uint(0)>>16&1 + ^uint(0)>>8&1 + 3)
)

var (
	s3AccessKey = flag.String("store-s3-ak", "", "s3 access key, use env S3_ACCESS_KEY by default")
	s3SecretKey = flag.String("store-s3-sk", "", "s3 secret key, use env S3_SECRET_KEY by default")
	s3Region    = flag.String("store-s3-region", "", "s3 region, use env S3_REGION by default")
	s3Endpoint  = flag.String("store-s3-endpoint", "", "s3 endpoint, if you want "+
		"output chunks to S3 bucket A andsubpath B, use s3:https://s3.amazonaws.com/A/B, ")

	localPath = flag.String("store-local-path", "./store", "filesystem store path")

	optThreads    = flag.Int("store-thread", 4, "store threads")
	optSkipVerify = flag.Bool("store-skip-verify", false, "store skip chunk digest verification")
	optChunkSize  = flag.String("store-chunk-size", "16:64:256", "min:avg:max chunk size in kb")

	memCacheSize = flag.String("store-memcache-size", "0Gi", "chunk memory cache size")
)

var (
	chunkStore desync.WriteStore
	indexStore desync.IndexWriteStore
)

func Init() {
	if len(*s3AccessKey) == 0 {
		*s3AccessKey = os.Getenv("S3_ACCESS_KEY")
	}
	if len(*s3SecretKey) == 0 {
		*s3SecretKey = os.Getenv("S3_SECRET_KEY")
	}
	if len(*s3Region) == 0 {
		*s3Region = os.Getenv("S3_REGION")
	}

	var err error
	if len(*s3Endpoint) > 0 {
		chunkStore, err = newS3Store()
		if err != nil {
			log.WithError(err).Fatal("init chunk store")
		}
		indexStore, err = newS3IndexStore()
		if err != nil {
			log.WithError(err).Fatal("init index store")
		}
	} else {
		chunkStore, err = newLocalStore()
		if err != nil {
			log.WithError(err).Fatal("init chunk store")
		}
		indexStore, err = newLocalIndexStore()
		if err != nil {
			log.WithError(err).Fatal("init index store")
		}
	}

	memCacheMaxBytes, err := humanize.ParseBytes(*memCacheSize)
	if err != nil {
		log.WithError(err).Fatalf("invalid memory cache size %q", *memCacheSize)
	}
	if memCacheMaxBytes > 0 {
		if memCacheMaxBytes > maxInt {
			log.Fatal("exceed memory max size %s", humanize.Bytes(maxInt))
		}
		chunkStore = newMemoryCacheStore(int(memCacheMaxBytes), chunkStore)
	}
}

func newS3Store() (desync.WriteStore, error) {
	endpointURL, err := url.Parse(*s3Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint url %s: %w", *s3Endpoint, err)
	}
	cred := credentials.NewStatic(*s3AccessKey, *s3SecretKey, "", credentials.SignatureV4)

	store, err := desync.NewS3Store(endpointURL, cred, *s3Region, desync.StoreOptions{
		N:          *optThreads,
		SkipVerify: *optSkipVerify,
	}, minio.BucketLookupAuto)
	if err != nil {
		return nil, fmt.Errorf("create s3 chunk store: %w", err)
	}
	return store, nil
}

// TODO local store with gc
func newLocalStore() (desync.WriteStore, error) {
	path := filepath.Join(*localPath, "chunk")
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", path, err)
	}
	store, err := desync.NewLocalStore(path, desync.StoreOptions{
		N:          *optThreads,
		SkipVerify: *optSkipVerify,
	})
	if err != nil {
		return nil, fmt.Errorf("create local chunk store: %w", err)
	}
	return store, nil
}

func newS3IndexStore() (desync.IndexWriteStore, error) {
	endpointURL, err := url.Parse(*s3Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint url %s: %w", *s3Endpoint, err)
	}
	cred := credentials.NewStatic(*s3AccessKey, *s3SecretKey, "", credentials.SignatureV4)

	store, err := desync.NewS3IndexStore(endpointURL, cred, *s3Region, desync.StoreOptions{
		N:          *optThreads,
		SkipVerify: *optSkipVerify,
	}, minio.BucketLookupAuto)
	if err != nil {
		return nil, fmt.Errorf("create s3 index store: %w", err)
	}
	return store, nil
}

func newLocalIndexStore() (desync.IndexWriteStore, error) {
	path := filepath.Join(*localPath, "index")
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", path, err)
	}
	store, err := desync.NewLocalIndexStore(path)
	if err != nil {
		return nil, fmt.Errorf("create local index store: %w", err)
	}
	return store, nil
}
