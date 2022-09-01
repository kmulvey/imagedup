package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"regexp"
	"runtime"
	"syscall"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kmulvey/imagedup/internal/app/imagedup/diffpool"
	"github.com/kmulvey/imagedup/internal/app/imagedup/logger"
	"github.com/kmulvey/imagedup/internal/app/imagedup/stream"
	"github.com/kmulvey/imagedup/pkg/imagedup/cache"
	"github.com/kmulvey/imagedup/pkg/types"
	"github.com/kmulvey/path"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const hashCacheFile = "hashcache.json"

func main() {
	var start = time.Now()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	var gracefulShutdown = make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, os.Interrupt, syscall.SIGTERM)

	// prom
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":5000", nil))
	}()

	// get user opts
	var rootDir string
	var threads int
	var distanceThreshold int
	flag.StringVar(&rootDir, "dir", "", "directory (abs path)")
	flag.IntVar(&threads, "threads", 1, "number of threads to use, >1 only useful when rebuilding the cache")
	flag.IntVar(&distanceThreshold, "distance", 10, "max distance for images to be considered the same")
	flag.Parse()
	if strings.TrimSpace(rootDir) == "" {
		log.Fatal("directory not provided")
	}
	if threads <= 0 || threads > runtime.GOMAXPROCS(0) {
		threads = 1
	}

	var ctx, cancel = context.WithCancel(context.Background())
	var pairChan = make(chan types.Pair)
	var files, imageHashCache, dp = setup(ctx, rootDir, threads, distanceThreshold, pairChan)
	go stream.StreamFiles(ctx, files, pairChan)

	log.Info("Started, go to grafana to monitor")

	// wait for all diff workers to finish or we get a shutdown signal
	// whichever comes first
	var workers, graceful = true, true
	for workers && graceful {
		select {
		case <-gracefulShutdown:
			graceful = false
		case <-dp.Wait():
			workers = false
		}
	}

	// shut everything down
	log.Info("Shutting down")
	cancel()
	var err = shutdown(imageHashCache)
	if err != nil {
		log.Fatal("error shutting down", err)
	}

	log.Info("Total time taken: ", time.Since(start))
}

func setup(ctx context.Context, rootDir string, threads, distanceThreshold int, pairChan chan types.Pair) ([]string, *cache.HashCache, *diffpool.DiffPool) {

	var deleteLogger = logger.NewDeleteLogger()

	// list all the files
	var files, err = path.ListFilesWithFilter(rootDir, regexp.MustCompile(".*.jpg$|.*.jpeg$|.*.png$.*.webm$"))
	handleErr("listFiles", err)
	var fileNames = path.OnlyNames(files)
	log.Infof("Found %d images", len(files))

	// init the image cache
	imageHashCache, err := cache.NewHashCache(hashCacheFile)
	handleErr("NewHashCache", err)
	log.Infof("Loaded %d image hashes from disk cache", len(imageHashCache.Cache))

	// init diff workers
	var dp = diffpool.NewDiffPool(ctx, threads, distanceThreshold, pairChan, imageHashCache, deleteLogger)

	// init prom
	go publishStats(imageHashCache)

	// starter stats
	totalComparisons.Set(float64(len(files) * (len(files) - 1)))

	return fileNames, imageHashCache, dp
}

// shutdown gracefully shuts everything down and stores caches for next time
func shutdown(cache *cache.HashCache) error {

	var err = cache.Persist(hashCacheFile)
	if err != nil {
		return err
	}

	return cache.Persist(hashCacheFile)
}

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}
