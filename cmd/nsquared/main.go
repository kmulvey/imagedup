package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	// _ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kmulvey/imagedup/internal/app/imagedup"
	"github.com/kmulvey/imagedup/pkg/imagedup/logger"
	"github.com/kmulvey/path"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.szostok.io/version"
	"go.szostok.io/version/printer"
)

func main() {
	var start = time.Now()
	var ctx, cancel = context.WithCancel(context.Background())

	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	var gracefulShutdown = make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, os.Interrupt, syscall.SIGTERM)

	// prom
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

	// get user opts
	var rootDir string
	var cacheFile string
	var outputFile string
	var threads int
	var distanceThreshold int
	var dedupFilePairs bool
	var help bool
	var v bool
	flag.StringVar(&rootDir, "dir", "", "directory (abs path)")
	flag.StringVar(&cacheFile, "cache-file", "cache.json", "json file to store the image hashes")
	flag.StringVar(&outputFile, "output-file", "delete.log", "log file to store the duplicate pairs")
	flag.IntVar(&threads, "threads", 1, "number of threads to use, >1 only useful when rebuilding the cache")
	flag.IntVar(&distanceThreshold, "distance", 10, "max distance for images to be considered the same")
	flag.BoolVar(&dedupFilePairs, "dedup-file-pairs", false, "dedup file pairs e.g. if a&b have been compared then dont comprare b&a as it will have the same result. doing this will reduce the time to diff but will also require more memory.")
	flag.BoolVar(&help, "h", false, "print help")
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
	if strings.TrimSpace(rootDir) == "" {
		log.Fatal("directory not provided")
	}
	if threads <= 0 || threads > runtime.GOMAXPROCS(0) {
		threads = 1
	}
	if filepath.Ext(cacheFile) != ".json" {
		log.Fatal("cache file must have extension .json")
	}
	if filepath.Ext(outputFile) != ".log" {
		log.Fatal("output file must have extension .log")
	}

	// list all the files
	var files, err = path.ListFilesWithFilter(rootDir, regexp.MustCompile(".*.jpg$|.*.jpeg$|.*.png$.*.webm$"))
	handleErr("listFiles", err)
	var fileNames = path.OnlyNames(files)
	log.Infof("Found %d files", len(files))

	// start er up
	resultsLogger, err := logger.NewDeleteLogger(outputFile)
	handleErr("NewImageDup", err)
	id, err := imagedup.NewImageDup("imagedup", cacheFile, threads, len(files), distanceThreshold, dedupFilePairs)
	handleErr("NewImageDup", err)

	var results, errors = id.Run(ctx, fileNames)

	log.Info("Started, go to grafana to monitor")

	// wait for all diff workers to finish or we get a shutdown signal
	// whichever comes first
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
				logger.LogResult(resultsLogger, result)
			case err, open := <-errors:
				if !open {
					errors = nil
					continue
				}
				log.Error(err)
			}
		}
	}

	// shut everything down
	log.Info("Shutting down")
	cancel()
	err = id.Shutdown()
	if err != nil {
		log.Fatal("error shutting down", err)
	}

	log.Info("Total time taken: ", time.Since(start))
}

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}
