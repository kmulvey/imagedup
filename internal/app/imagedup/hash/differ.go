package imagedup

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/kmulvey/goutils"
	"github.com/kmulvey/imagedup/pkg/imagedup/cache"
	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type differ struct {
	ctx               context.Context
	wg                *sync.WaitGroup
	workChan          chan types.Pair
	errors            chan error
	cache             *cache.HashCache
	deleteLogger      *logrus.Logger
	distanceThreshold int
	*stats
}

func newDiffer(ctx context.Context, numWorkers, distanceThreshold int, workChan chan types.Pair, cache *cache.HashCache, deleteLogger *logrus.Logger, stats *stats) *differ {

	if numWorkers <= 0 || numWorkers > runtime.GOMAXPROCS(0)-1 {
		numWorkers = 1
	}

	var dp = &differ{
		ctx:               ctx,
		wg:                new(sync.WaitGroup),
		workChan:          workChan,
		errors:            make(chan error),
		cache:             cache,
		deleteLogger:      deleteLogger,
		distanceThreshold: distanceThreshold,
	}

	var errorChans = make([]chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		dp.wg.Add(1)
		var errors = make(chan error)
		errorChans[i] = errors
		go dp.run(errors)
	}
	dp.errors = goutils.MergeChannels(errorChans...)

	return dp
}

func (dp *differ) wait() chan error {
	return dp.errors
}

func (dp *differ) run(errors chan error) {

	// declare these here to reduce allocations in the loop
	var start time.Time
	var imgCacheOne, imgCacheTwo *cache.ImageCache
	var err error
	var distance int

	for {
		select {
		case <-dp.ctx.Done():
			dp.wg.Done()
			return
		default:
			p, open := <-dp.workChan
			if !open {
				dp.wg.Done()
				return
			}
			start = time.Now()

			imgCacheOne, err = dp.cache.GetHash(p.One)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.One, err)
				continue
			}

			imgCacheTwo, err = dp.cache.GetHash(p.Two)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.Two, err)
				continue
			}

			distance, err = imgCacheOne.ImageHash.Distance(imgCacheTwo.ImageHash)
			if err != nil {
				errors <- fmt.Errorf("GetHash failed for image: %s, err: %w", p.Two, err)
				continue
			}

			if distance < dp.distanceThreshold {
				if (imgCacheOne.Config.Height * imgCacheOne.Config.Width) > (imgCacheTwo.Config.Height * imgCacheTwo.Config.Width) {
					dp.deleteLogger.WithFields(log.Fields{
						"big":   p.One,
						"small": p.Two,
					}).Info("delete")
				} else {
					dp.deleteLogger.WithFields(log.Fields{
						"big":   p.Two,
						"small": p.One,
					}).Info("delete")
				}
			}

			dp.stats.DiffTime.Set(float64(time.Since(start)))
			dp.stats.ComparisonsCompleted.Inc()
		}
	}
}
