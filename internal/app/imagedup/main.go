package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/internal/app/imagedup/hash"
	"github.com/kmulvey/imagedup/internal/app/imagedup/stats"
	"github.com/kmulvey/imagedup/pkg/imagedup/cache"
	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/sirupsen/logrus"
)

type ImageDup struct {
	context.Context
	*stats.Stats
	*cache.HashCache
	deleteLogger *logrus.Logger
	*hash.Differ
}

func NewImageDup(ctx context.Context, promNamespace, hashCacheFile, deleteLogFile string, numWorkers, distanceThreshold int) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.Context = ctx
	id.Stats = stats.New(promNamespace)

	id.HashCache, err = cache.NewHashCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	id.deleteLogger, err = newDeleteLogger(deleteLogFile)
	if err != nil {
		return nil, err
	}

	var workChan = make(chan types.Pair)
	id.Differ = hash.NewDiffer(ctx, numWorkers, distanceThreshold, workChan, id.HashCache, id.deleteLogger, id.Stats)

	return id, nil
}

func (id *ImageDup) Errors() error {
	return <-id.Differ.Wait()
}
