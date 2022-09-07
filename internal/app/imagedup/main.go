package imagedup

import (
	"context"

	"github.com/kmulvey/imagedup/pkg/imagedup/cache"
)

type ImageDup struct {
	context.Context
	*stats
	*cache.HashCache
}

func NewImageDup(promNamespace, hashCacheFile string) (*ImageDup, error) {
	var id = new(ImageDup)
	var err error

	id.stats = newStats(promNamespace)

	id.HashCache, err = cache.NewHashCache(hashCacheFile, promNamespace)
	if err != nil {
		return nil, err
	}

	return id, nil
}
