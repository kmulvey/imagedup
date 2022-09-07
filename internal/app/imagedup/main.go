package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/pkg/imagedup/cache"
	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/sirupsen/logrus"
)

type ImageDup struct {
	context.Context
	*stats
	*cache.HashCache
	deleteLogger *logrus.Logger
	*differ
}

func NewImageDup(ctx context.Context, promNamespace, hashCacheFile, deleteLogFile string, numWorkers, distanceThreshold int) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.Context = ctx
	id.stats = newStats(promNamespace)

	id.HashCache, err = cache.NewHashCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	id.deleteLogger, err = newDeleteLogger(deleteLogFile)
	if err != nil {
		return nil, err
	}

	var workChan = make(chan types.Pair)
	id.differ = newDiffer(ctx, numWorkers, distanceThreshold, workChan, id.HashCache, id.deleteLogger, id.stats)

	return id, nil
}

func (id *ImageDup) Errors() error {
	return <-id.differ.wait()
}
