package main

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type DiffPool struct {
	ctx          context.Context
	wg           *sync.WaitGroup
	workChan     chan pair
	cache        *hashCache
	deleteLogger *logrus.Logger
}

func NewDiffPool(ctx context.Context, numWorkers int, workChan chan pair, cache *hashCache, deleteLogger *logrus.Logger) *DiffPool {

	var dp = &DiffPool{
		ctx:          ctx,
		wg:           new(sync.WaitGroup),
		workChan:     workChan,
		cache:        cache,
		deleteLogger: deleteLogger,
	}

	for i := 0; i < numWorkers; i++ {
		dp.wg.Add(1)
		go dp.diff()
	}

	return dp
}

func (dp *DiffPool) wait() chan struct{} {
	var c = make(chan struct{})
	go func() {
		dp.wg.Wait()
		close(c)
	}()

	return c
}

func (dp *DiffPool) diff() {

	// declare these here to reduce allocations in the loop
	var start time.Time
	var imgCacheOne, imgCacheTwo *imageCache
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
			handleErr("get hash: "+p.One, err)

			imgCacheTwo, err = dp.cache.GetHash(p.Two)
			handleErr("get hash: "+p.One, err)

			distance, err = imgCacheOne.ImageHash.Distance(imgCacheTwo.ImageHash)
			handleErr("distance", err)

			if distance < 10 {
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

			diffTime.Set(float64(time.Since(start)))
			comparisonsCompleted.Inc()
		}
	}
}
