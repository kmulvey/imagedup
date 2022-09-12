package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/internal/app/imagedup/hash"
	"github.com/kmulvey/imagedup/pkg/imagedup/types"
)

type ImageDup struct {
	*stats
	*hash.Cache
	*hash.Differ
	images chan types.Pair
}

func NewImageDup(promNamespace, hashCacheFile string, numWorkers, distanceThreshold int) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.images = make(chan types.Pair)
	id.stats = newStats(promNamespace)

	id.Cache, err = hash.NewCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	id.Differ = hash.NewDiffer(numWorkers, distanceThreshold, id.images, id.Cache, promNamespace)

	go id.stats.publishStats(id.Cache)

	return id, nil
}

func (id *ImageDup) Run(ctx context.Context, files []string) (chan hash.DiffResult, chan error) {
	var results, errors = id.Differ.Run(ctx)
	id.streamFiles(ctx, files)
	return results, errors
}

func (id *ImageDup) Shutdown() error {
	return id.Cache.Persist()
}
