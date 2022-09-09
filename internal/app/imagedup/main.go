package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/internal/app/imagedup/hash"
	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/sirupsen/logrus"
)

type ImageDup struct {
	context.Context
	*stats
	*hash.Cache
	deleteLogger *logrus.Logger
	*hash.Differ
	images chan types.Pair
}

func NewImageDup(ctx context.Context, promNamespace, hashCacheFile, deleteLogFile string, numWorkers, distanceThreshold int) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.Context = ctx
	id.images = make(chan types.Pair)
	id.stats = newStats(promNamespace)

	id.Cache, err = hash.NewCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	id.deleteLogger, err = newDeleteLogger(deleteLogFile)
	if err != nil {
		return nil, err
	}

	id.Differ = hash.NewDiffer(ctx, numWorkers, distanceThreshold, id.images, id.Cache, id.deleteLogger, promNamespace)

	go id.stats.publishStats(id.Cache)

	return id, nil
}

func (id *ImageDup) Run(files []string) chan error {
	var errors = id.Differ.Run()
	id.streamFiles(files)
	return errors
}

func (id *ImageDup) Shutdown(cacheFile string) error {
	return id.Cache.Persist(cacheFile)
}
