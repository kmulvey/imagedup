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
}

func NewImageDup(ctx context.Context, promNamespace, hashCacheFile, deleteLogFile string, numWorkers, distanceThreshold int) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.Context = ctx
	id.stats = newStats(promNamespace)

	id.Cache, err = hash.NewCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	id.deleteLogger, err = newDeleteLogger(deleteLogFile)
	if err != nil {
		return nil, err
	}

	var workChan = make(chan types.Pair)
	id.Differ = hash.NewDiffer(ctx, numWorkers, distanceThreshold, workChan, id.Cache, id.deleteLogger, promNamespace)

	return id, nil
}

func (id *ImageDup) Errors() error {
	return <-id.Differ.Wait()
}
