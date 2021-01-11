package desync

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/folbricht/desync"
)

func ChopFile(ctx context.Context, file, indexName string) error {
	min, avg, max, err := parseChunkSizeParam(*optChunkSize)
	if err != nil {
		return fmt.Errorf("parse chunk size %q: %w", *optChunkSize, err)
	}
	index, _, err := desync.IndexFromFile(ctx, file, *optThreads, min, avg, max, nil)
	if err != nil {
		return fmt.Errorf("create desync index: %w", err)
	}
	if err := desync.ChopFile(ctx, file, index.Chunks, chunkStore, *optThreads, nil); err != nil {
		return fmt.Errorf("chop file: %w", err)
	}
	if err := indexStore.StoreIndex(indexName, index); err != nil {
		return fmt.Errorf("store index: %w", err)
	}
	return nil
}

func parseChunkSizeParam(s string) (min, avg, max uint64, err error) {
	sizes := strings.Split(s, ":")
	if len(sizes) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid chunk size '%s'", s)
	}
	num, err := strconv.Atoi(sizes[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf( "min chunk size: %w", err)
	}
	min = uint64(num) * 1024
	num, err = strconv.Atoi(sizes[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf( "avg chunk size: %w", err)
	}
	avg = uint64(num) * 1024
	num, err = strconv.Atoi(sizes[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf( "max chunk size: %w", err)
	}
	max = uint64(num) * 1024
	return
}