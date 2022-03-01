package main

import (
	"context"
	"flag"
	_ "net/http/pprof"
	"os/signal"
	"runtime"
	"syscall"

	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const hashCacheFile = "hashcache.json"

// pair represents two images, their paths and thier element # in the files list
type pair struct {
	I   int
	J   int
	One string
	Two string
}

func init() {
	//	log.SetFormatter(&log.TextFormatter{})
}

func main() {
	var start = time.Now()

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
	flag.StringVar(&rootDir, "dir", "", "directory (abs path)")
	flag.IntVar(&threads, "threads", 1, "number of threads to use")
	flag.Parse()
	if strings.TrimSpace(rootDir) == "" {
		log.Fatal("directory not provided")
	}
	if threads < 0 || threads > runtime.GOMAXPROCS(0) {
		threads = 1
	}

	var ctx, cancel = context.WithCancel(context.Background())
	var pairChan = make(chan pair)
	var files, imageHashCache, dp = setup(ctx, rootDir, threads, pairChan)
	go streamFiles(ctx, files, pairChan)

	log.Info("Started, go to grafana to monitor")

	// wait for all diff workers to finish or we get a shutdown signal
	// whichever comes first
	var workers, graceful = true, true
	for workers && graceful {
		select {
		case <-gracefulShutdown:
			graceful = false
		case <-dp.wait():
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

func setup(ctx context.Context, rootDir string, threads int, pairChan chan pair) ([]string, *hashCache, *DiffPool) {

	var deleteLogger = NewDeleteLogger()

	// list all the files
	files, err := listFiles(rootDir)
	handleErr("listFiles", err)
	log.Infof("Found %d images", len(files))

	// init the image cache
	imageHashCache, err := NewHashCache(hashCacheFile)
	handleErr("NewHashCache", err)
	log.Infof("Loaded %d image hashes from disk cache", len(imageHashCache.Cache))

	// init diff workers
	var dp = NewDiffPool(ctx, threads, pairChan, imageHashCache, deleteLogger)

	// init prom
	go publishStats(imageHashCache)

	// starter stats
	totalComparisons.Set(float64(len(files) * (len(files) - 1)))

	return files, imageHashCache, dp
}
