package imagedup

import (
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
	"github.com/prometheus/client_golang/prometheus"
)

// stats are prometheus stats for imagedup
type stats struct {
	PairTotal           prometheus.Counter
	GCTime              prometheus.Gauge
	TotalFiles          prometheus.Gauge
	TotalComparisons    prometheus.Gauge
	ImageCacheBytes     prometheus.Gauge
	ImageCacheNumImages prometheus.Gauge
	FileMapBytes        prometheus.Gauge
	FileMapEntries      prometheus.Gauge
	FileMapHits         prometheus.Counter
	FileMapMisses       prometheus.Counter
	PromNamespace       string
}

// newStats inits all the stats
func newStats(promNamespace string) *stats {
	var s = new(stats)
	s.PromNamespace = promNamespace

	s.GCTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "gc_time_nano",
			Help:      "how long a gc sweep took",
		},
	)
	s.TotalComparisons = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "total_comparisons",
			Help:      "how many comparisons need to be done",
		},
	)
	s.TotalFiles = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "total_files",
			Help:      "how many files we were give to compare",
		},
	)
	s.PairTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "pair_total",
			Help:      "How many pairs we read.",
		},
	)
	s.ImageCacheBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_size_bytes",
			Help:      "disk size of the cache",
		},
	)
	s.ImageCacheNumImages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "image_cache_num_images",
			Help:      "how many images are in the cache",
		},
	)
	s.FileMapBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "file_map_bytes",
			Help:      "size of the file dedup map",
		},
	)
	s.FileMapEntries = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: promNamespace,
			Name:      "file_map_entries",
			Help:      "number of entries in the file dedup map",
		},
	)
	s.FileMapHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "file_map_hits",
			Help:      "number of entries in the file dedup map",
		},
	)
	s.FileMapMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: promNamespace,
			Name:      "file_map_misses",
			Help:      "number of entries in the file dedup map",
		},
	)
	prometheus.MustRegister(s.PairTotal)
	prometheus.MustRegister(s.GCTime)
	prometheus.MustRegister(s.TotalComparisons)
	prometheus.MustRegister(s.TotalFiles)
	prometheus.MustRegister(s.ImageCacheBytes)
	prometheus.MustRegister(s.ImageCacheNumImages)
	prometheus.MustRegister(s.FileMapBytes)
	prometheus.MustRegister(s.FileMapEntries)
	prometheus.MustRegister(s.FileMapHits)
	prometheus.MustRegister(s.FileMapMisses)

	return s
}

// publishStats publishes go GC stats + cache size to prom every 10 seconds
func (s *stats) publishStats(imageCache *hash.Cache, dedupMap map[string]struct{}, dedupPairs bool, bitmapLock *sync.RWMutex) {
	for {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)

		s.GCTime.Set(float64(stats.PauseTotalNs))

		var numImages, cacheBytes = imageCache.Stats()
		s.ImageCacheNumImages.Set(float64(numImages))
		s.ImageCacheBytes.Set(float64(cacheBytes))

		if dedupPairs {
			bitmapLock.Lock()
			s.FileMapEntries.Set(float64(len(dedupMap)))
			s.FileMapBytes.Set(float64(unsafe.Sizeof(dedupMap)))
			bitmapLock.Unlock()
		}

		time.Sleep(10 * time.Second)
	}
}

// unregister removes all the stats
func (s *stats) unregister() {
	prometheus.Unregister(s.PairTotal)
	prometheus.Unregister(s.GCTime)
	prometheus.Unregister(s.TotalComparisons)
	prometheus.Unregister(s.TotalFiles)
	prometheus.Unregister(s.ImageCacheBytes)
	prometheus.Unregister(s.ImageCacheNumImages)
	prometheus.Unregister(s.FileMapBytes)
	prometheus.Unregister(s.FileMapEntries)
	prometheus.Unregister(s.FileMapHits)
	prometheus.Unregister(s.FileMapMisses)
}
