package main

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const PromNamespace = "imagedup"

var (
	diffTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "diff_time_nano",
			Help:      "How long it takes to diff two images, in nanoseconds.",
		},
	)
	pairTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: PromNamespace,
			Name:      "pair_total",
			Help:      "How many pairs we read.",
		},
	)
	gcTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "gc_time_nano",
		},
	)
	totalComparisons = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "total_comparisons",
		},
	)
	comparisonsCompleted = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "comparisons_completed",
		},
	)
	imageCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "image_cache_size_bytes",
		},
	)
	imageCacheNumImages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "image_cache_num_images",
		},
	)
	pairCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: PromNamespace,
			Name:      "pair_cache_size",
		},
	)
)

func init() {
	prometheus.MustRegister(diffTime)
	prometheus.MustRegister(pairTotal)
	prometheus.MustRegister(gcTime)
	prometheus.MustRegister(totalComparisons)
	prometheus.MustRegister(comparisonsCompleted)
	prometheus.MustRegister(imageCacheSize)
	prometheus.MustRegister(imageCacheNumImages)
	prometheus.MustRegister(pairCacheSize)
}

// publishStats publishes go GC stats + cache size to prom
func publishStats(imageCache *hashCache) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		gcTime.Set(float64(stats.PauseTotalNs))

		imageCacheNumImages.Set(float64(imageCache.NumImages()))

		time.Sleep(10 * time.Second)
	}
}
