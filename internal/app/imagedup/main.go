package imagedup

import (
	"context"
	"sync"

	"github.com/RoaringBitmap/roaring/roaring64"
	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
	"github.com/kmulvey/imagedup/v2/pkg/imagedup/types"
)

type ImageDup struct {
	*stats
	*hash.Cache
	*hash.Differ
	*roaring64.Bitmap
	images     chan types.Pair
	dedupPairs bool
	bitmapLock sync.RWMutex
}

func NewImageDup(promNamespace, hashCacheFile string, numWorkers, distanceThreshold int, dedupPairs bool) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.images = make(chan types.Pair)
	id.stats = newStats(promNamespace)
	id.Bitmap = roaring64.New()
	id.dedupPairs = dedupPairs

	id.Cache, err = hash.NewCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	id.Differ = hash.NewDiffer(numWorkers, distanceThreshold, id.images, id.Cache, promNamespace)

	go id.stats.publishStats(id.Cache, id.Bitmap, dedupPairs, &id.bitmapLock)

	return id, nil
}

func (id *ImageDup) Run(ctx context.Context, files []string) (chan hash.DiffResult, chan error) {
	var results, errors = id.Differ.Run(ctx)
	go id.streamFiles(ctx, files)
	return results, errors
}

func (id *ImageDup) Shutdown() error {
	id.stats.unregister()
	id.Differ.Shutdown()
	return id.Cache.Persist()
}
