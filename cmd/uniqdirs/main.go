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
	"github.com/kmulvey/imagedup/v2/pkg/imagedup/logger"
	"github.com/kmulvey/path"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.szostok.io/version"
	"go.szostok.io/version/printer"
)

var logExt = ".log"

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
	var threads, distanceThreshold, depth int
	var dedupFilePairs, help, v bool
	flag.StringVar(&rootDir, "dir", "", "directory (abs path)")
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
	if strings.TrimSpace(rootDir) == "" {
		log.Fatal("directory not provided")
	}
	if threads <= 0 || threads > runtime.GOMAXPROCS(0) {
		threads = 1
	}

	// list all the dirs
	dirs, err := path.List(rootDir, uint8(depth), false, path.NewDirEntitiesFilter())
	handleErr("listFiles", err)
	var dirNames = path.OnlyNames(dirs)
	log.Infof("Found %d dirs", len(dirNames))

	for _, dir := range dirNames {
		log.Infof("Starting %s", dir)

		var ctx, cancel = context.WithCancel(context.Background())
		if continueLoop := dedupDir(ctx, cancel, dir, threads, distanceThreshold, depth, dedupFilePairs, gracefulShutdown); !continueLoop {
			break
		}

		// delete emptys
		var logFile, err = os.Stat(filepath.Base(dir) + logExt)
		if err != nil {
			continue
		}
		if logFile.Size() == 0 {
			err = os.RemoveAll(filepath.Base(dir) + logExt)
			handleErr("remove log file", err)
		}
	}
	log.Info("Total time taken: ", time.Since(start))
}

// handleErr is a convience func to log and quit errors, all errors in this app are considered fatal
func handleErr(prefix string, err error) {
	if err != nil {
		log.Fatal(fmt.Errorf("%s: %w", prefix, err))
	}
}

// dedupDir returns a bool representing 'continue' which is usually true except when an os signal is received, then false
func dedupDir(ctx context.Context, cancel context.CancelFunc, dir string, threads, distanceThreshold, depth int, dedupFilePairs bool, gracefulShutdown chan os.Signal) bool {
	// list all the files
	var files, err = path.List(dir, uint8(depth), false, path.NewRegexEntitiesFilter(imagedup.ImageExtensionRegex))
	handleErr("listFiles", err)
	var fileNames = path.OnlyNames(files)
	log.Infof("Found %d files", len(files))
	if len(files) < 2 {
		log.Infof("Skipping %s because there are only %d files", dir, len(files))
		return true
	}
	// start er up

	resultsLogger, err := logger.NewDeleteLogger(filepath.Base(dir) + logExt)
	handleErr("NewImageDup", err)
	id, err := imagedup.NewImageDup("imagedup", filepath.Base(dir)+".json", threads, len(files), distanceThreshold, dedupFilePairs)
	handleErr("NewImageDup", err)

	var results, errors = id.Run(ctx, fileNames)

	// wait for all diff workers to finish or we get a shutdown signal
	// whichever comes first
	for results != nil || errors != nil {
		select {
		case <-gracefulShutdown:
			return false
		default:
			select {
			case result, open := <-results:
				if !open {
					results = nil
					continue
				}
				if err := resultsLogger.LogResult(result); err != nil {
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

	// shut everything down
	cancel()
	err = id.Shutdown()
	if err != nil {
		log.Fatal("error shutting down", err)
	}

	return true
}
