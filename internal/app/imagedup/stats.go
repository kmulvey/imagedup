package stats

import (
	"runtime"
	"time"

	"github.com/kmulvey/imagedup/internal/app/imagedup/hash"
	"github.com/prometheus/client_golang/prometheus"
)

type Stats struct {
	PromNamespace       string
	PairTotal           prometheus.Counter
	GCTime              prometheus.Gauge
	TotalComparisons    prometheus.Gauge
	ImageCacheSize      prometheus.Gauge
	ImageCacheNumImages prometheus.Gauge
	PairCacheSize       prometheus.Gauge
}

func New(promNamespace string) *Stats {
	var s = new(Stats)
	s.PromNamespace = promNamespace

	s.PairTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "pair_total",
			Help:      "How many pairs we read.",
		},
	)
	s.GCTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "gc_time_nano",
		},
	)
	s.TotalComparisons = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "total_comparisons",
		},
	)
	s.ImageCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_size_bytes",
		},
	)
	s.ImageCacheNumImages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_num_images",
		},
	)
	s.PairCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "pair_cache_size",
		},
	)
	prometheus.MustRegister(s.PairTotal)
	prometheus.MustRegister(s.GCTime)
	prometheus.MustRegister(s.TotalComparisons)
	prometheus.MustRegister(s.ImageCacheSize)
	prometheus.MustRegister(s.ImageCacheNumImages)
	prometheus.MustRegister(s.PairCacheSize)

	return s
}

// publishStats publishes go GC stats + cache size to prom every 10 seconds
func (s *Stats) publishStats(imageCache *hash.Cache) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		s.GCTime.Set(float64(stats.PauseTotalNs))

		s.ImageCacheNumImages.Set(float64(imageCache.NumImages()))

		time.Sleep(10 * time.Second)
	}
}
