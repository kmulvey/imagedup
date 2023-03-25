package imagedup

import (
	"context"
	"fmt"
	"sync"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
	"github.com/kmulvey/imagedup/v2/pkg/imagedup/types"
)

// ImageDup diffs images in order to find similar/duplicate images
type ImageDup struct {
	*stats
	HashCache *hash.Cache
	*hash.Differ
	dedupCache map[string]struct{}
	images     chan types.Pair
	dedupPairs bool
	bitmapLock sync.RWMutex
}

// NewImageDup is the constructor which sets up everything for diffing but does not actually start diffing, Run() must be called for that.
func NewImageDup(promNamespace, hashCacheFile string, numWorkers, numFiles, distanceThreshold int, dedupPairs bool) (*ImageDup, error) {

	if numFiles < 2 {
		return nil, fmt.Errorf("Skipping because there are only %d files", numFiles)
	}
	var id = new(ImageDup)
	var err error

	id.images = make(chan types.Pair)
	id.stats = newStats(promNamespace)
	id.dedupPairs = dedupPairs
	if dedupPairs {
		id.dedupCache = make(map[string]struct{}, numFiles)
	}

	id.HashCache, err = hash.NewCache(hashCacheFile, promNamespace, numFiles)
	if err != nil {
		return nil, err
	}

	id.Differ = hash.NewDiffer(numWorkers, distanceThreshold, id.images, id.HashCache, promNamespace)

	go id.stats.publishStats(id.HashCache, id.dedupCache, dedupPairs, &id.bitmapLock)

	return id, nil
}

// Run starts the diff workers and feeds them files
func (id *ImageDup) Run(ctx context.Context, files []string) (chan hash.DiffResult, chan error) {
	var results, errors = id.Differ.Run(ctx)
	go id.streamFiles(ctx, files)
	return results, errors
}

// Shutdown unregisters prom stats and writes the image cache to disk. Context cancel must be called to
// kill the differ workers. See nsquared/main.go for an example
func (id *ImageDup) Shutdown() error {
	id.stats.unregister()
	id.Differ.Shutdown()
	return id.HashCache.Persist()
}
