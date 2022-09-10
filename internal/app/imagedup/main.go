package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/internal/app/imagedup/hash"
	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type ImageDup struct {
	*stats
	*hash.Cache
	deleteLogger *logrus.Logger
	*hash.Differ
	images chan types.Pair
}

func NewImageDup(promNamespace, hashCacheFile, deleteLogFile string, numWorkers, distanceThreshold int) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

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

	id.Differ = hash.NewDiffer(numWorkers, distanceThreshold, id.images, id.Cache, id.deleteLogger, promNamespace)

	go id.stats.publishStats(id.Cache)

	return id, nil
}

func (id *ImageDup) Run(ctx context.Context, files []string) chan error {
	var errors = id.Differ.Run(ctx)
	id.streamFiles(ctx, files)
	return errors
}

func (id *ImageDup) Shutdown() error {
	return id.Cache.Persist()
}

func (id *ImageDup) CollectResults(results chan hash.DiffResult) {
	for result := range results {
		if result.OneArea > result.TwoArea {
			id.deleteLogger.WithFields(log.Fields{
				"big":   result.One,
				"small": result.Two,
			}).Info("delete")
		} else {
			id.deleteLogger.WithFields(log.Fields{
				"big":   result.Two,
				"small": result.One,
			}).Info("delete")
		}
	}
}
