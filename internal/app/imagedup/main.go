package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/pkg/imagedup/cache"
	"github.com/sirupsen/logrus"
)

type ImageDup struct {
	context.Context
	*stats
	*cache.HashCache
	deleteLogger *logrus.Logger
}

func NewImageDup(promNamespace, hashCacheFile, deleteLogFile string) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.stats = newStats(promNamespace)

	id.HashCache, err = cache.NewHashCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	id.deleteLogger, err = newDeleteLogger(deleteLogFile)
	if err != nil {
		return nil, err
	}

	return id, nil
}
