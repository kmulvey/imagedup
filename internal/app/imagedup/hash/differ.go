package hash

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/kmulvey/goutils"
	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type Differ struct {
	ctx                  context.Context
	workChan             chan types.Pair
	errors               chan error
	cache                *Cache
	deleteLogger         *logrus.Logger
	distanceThreshold    int
	diffTime             prometheus.Gauge
	comparisonsCompleted prometheus.Gauge
}

func NewDiffer(ctx context.Context, numWorkers, distanceThreshold int, workChan chan types.Pair, cache *Cache, deleteLogger *logrus.Logger, promNamespace string) *Differ {

	if numWorkers <= 0 || numWorkers > runtime.GOMAXPROCS(0)-1 {
		numWorkers = 1
	}

	var dp = &Differ{
		ctx:               ctx,
		workChan:          workChan,
		errors:            make(chan error),
		cache:             cache,
		deleteLogger:      deleteLogger,
		distanceThreshold: distanceThreshold,
		diffTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: promNamespace,
				Name:      "diff_time_nano",
				Help:      "How long it takes to diff two images, in nanoseconds.",
			}),
		comparisonsCompleted: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: promNamespace,
				Name:      "comparisons_completed",
			}),
	}
	prometheus.MustRegister(dp.diffTime)
	prometheus.MustRegister(dp.comparisonsCompleted)

	var errorChans = make([]chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		var errors = make(chan error)
		errorChans[i] = errors
		go dp.run(errors)
	}

	dp.errors = goutils.MergeChannels(errorChans...)
	return dp
}

func (dp *Differ) Wait() chan error {
	return dp.errors
}

func (dp *Differ) run(errors chan error) {

	// declare these here to reduce allocations in the loop
	var start time.Time
	var imgCacheOne, imgCacheTwo *Image
	var err error
	var distance int

	for {
		select {
		case <-dp.ctx.Done():
			close(errors)
			return
		default:
			p, open := <-dp.workChan
			if !open {
				close(errors)
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

			dp.diffTime.Set(float64(time.Since(start)))
			dp.comparisonsCompleted.Inc()
		}
	}
}
