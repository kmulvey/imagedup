package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup"
	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
	"github.com/kmulvey/imagedup/v2/pkg/imagedup/logger"
	"github.com/kmulvey/path"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.szostok.io/version"
	"go.szostok.io/version/printer"
)

func main() {
	var start = time.Now()
	var ctx, cancel = context.WithCancel(context.Background())

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	var gracefulShutdown = make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		s := &http.Server{
			Addr:           ":5000",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		log.Fatal(s.ListenAndServe())
	}()

	dir, cacheFile, outputFile, threads, distanceThreshold, depth, dedupFilePairs := parseFlags()

	// list all the files
	//nolint:gosec
	var files, err = path.List(dir, uint8(depth), false, path.NewRegexEntitiesFilter(imagedup.ImageExtensionRegex))
	handleErr("listFiles", err)

	var fileNames = path.OnlyNames(files)
	log.Infof("Found %d files", len(files))
	if len(files) < 2 {
		log.Fatalf("Skipping %s because there are only %d files", dir, len(files))
	}

	resultsLogger, err := logger.NewDeleteLogger(outputFile)
	handleErr("NewDeleteLogger", err)

	id, err := imagedup.NewImageDup("imagedup", cacheFile, threads, len(files), distanceThreshold, dedupFilePairs)
	handleErr("NewImageDup", err)

	var results, errors = id.Run(ctx, fileNames)
	log.Info("Started, go to grafana to monitor")

	collectResults(results, errors, resultsLogger, gracefulShutdown)

	log.Info("Shutting down")
	cancel()
	if err = id.Shutdown(); err != nil {
		log.Fatal("error shutting down", err)
	}
	if err := resultsLogger.Close(); err != nil {
		log.Error(err)
	}
	log.Info("Total time taken: ", time.Since(start))
}

// parseFlags parses and validates CLI flags, exiting on --help/--version,
// and returns the resolved configuration values.
func parseFlags() (string, string, string, int, int, int, bool) {
	var dir, cacheFile, outputFile string
	var threads, distanceThreshold, depth int
	var dedupFilePairs, help, v bool
	flag.StringVar(&dir, "dir", "", "directory (abs path)")
	flag.StringVar(&cacheFile, "cache-file", "cache.json", "json file to store the image hashes which be different for different input dirs")
	flag.StringVar(&outputFile, "output-file", "delete.json", "json file to store the duplicate pairs, it will be deleted and recreated")
	flag.IntVar(&threads, "threads", 1, "number of threads to use, >1 only useful when rebuilding the cache")
	flag.IntVar(&depth, "depth", 2, "how far down the directory tree to search for files")
	flag.IntVar(&distanceThreshold, "distance", 10, "max distance for images to be considered the same")
	flag.BoolVar(&dedupFilePairs, "dedup-file-pairs", false, "dedup file pairs e.g. if a&b have been compared then dont comprare b&a as it will have the same result. doing this will reduce the time to diff but will also require more memory.")
	flag.BoolVar(&help, "help", false, "print help")
	flag.BoolVar(&v, "version", false, "print version")
	flag.BoolVar(&v, "v", false, "print version")
	flag.Parse()

	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if v {
		var verPrinter = printer.New()
		var info = version.Get()
		if err := verPrinter.PrintInfo(os.Stdout, info); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	if _, err := os.Stat(strings.TrimSpace(dir)); err != nil {
		log.Fatalf("directory %s is not valid, err :%s \n", dir, err.Error())
	}
	if threads <= 0 || threads > runtime.GOMAXPROCS(0) {
		threads = 1
	}
	if filepath.Ext(cacheFile) != ".json" {
		log.Fatal("cache file must have extension .json")
	}
	if filepath.Ext(outputFile) != ".json" {
		log.Fatal("output file must have extension .json")
	}
	return dir, cacheFile, outputFile, threads, distanceThreshold, depth, dedupFilePairs
}

// collectResults drains the result and error channels, logging each entry,
// until both are closed or a shutdown signal is received.
func collectResults(results chan hash.DiffResult, errors chan error, rl *logger.DeleteLogger, gracefulShutdown chan os.Signal) {
CollectionLoop:
	for results != nil || errors != nil {
		select {
		case <-gracefulShutdown:
			break CollectionLoop
		default:
			select {
			case result, open := <-results:
				if !open {
					results = nil
					continue
				}
				if err := rl.LogResult(result); err != nil {
					log.Error(err)
				}
			case err, open := <-errors:
				if !open {
					errors = nil
					continue
				}
				log.Error(err)
			}
		}
	}
}

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}
