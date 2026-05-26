package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kmulvey/imagedup/v2/pkg/imagedup/logger"
	"github.com/kmulvey/path"
	log "github.com/sirupsen/logrus"
	"go.szostok.io/version"
	"go.szostok.io/version/printer"
)

// ErrUnsupportedViewer is returned when an unsupported image viewer command is requested.
var ErrUnsupportedViewer = errors.New("unsupported viewer command")

const (
	viewerPreview = "preview"
	viewerEOG     = "eog"
)

func main() {
	var alwaysDelete bool
	var deleteFiles path.Entry
	var v bool
	var help bool
	flag.BoolVar(&alwaysDelete, "always-delete", false, "just take the larger one, always")
	flag.Var(&deleteFiles, "delete-files", "json file where duplicate pairs are stored, same file from -cache-file when running nsquared")
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

	var files, err = deleteFiles.Flatten(true)
	if err != nil {
		log.Fatal("error flattening files: ", err)
	}

	for _, deleteFile := range files {
		processDeleteFile(deleteFile.AbsolutePath, alwaysDelete)
	}
}

// processDeleteFile loads a log file and processes every duplicate pair in it.
func processDeleteFile(path string, alwaysDelete bool) {
	var dedupedFiles, err = logger.ReadDeleteLogFile(path)
	if err != nil {
		log.Fatalf("error reading file: %s, err: %s", path, err)
	}

	var viewer = viewerForOS()

	for i, pair := range dedupedFiles {
		processPair(i, len(dedupedFiles), pair, alwaysDelete, viewer)
	}
}

// viewerForOS returns the image viewer command for the current OS.
func viewerForOS() string {
	switch runtime.GOOS {
	case "darwin":
		return viewerPreview
	case "linux":
		return viewerEOG // eog -- GNOME Image Viewer 41.1
	case "windows":
		return ""
	default:
		log.Fatalf("unsupported os: %s", runtime.GOOS)
		return ""
	}
}

// processPair handles a single duplicate pair: skip, auto-delete, or interactive review.
func processPair(idx, total int, pair logger.DeleteEntry, alwaysDelete bool, viewer string) {
	// skip pairs that have already been handled
	if !fileExists(pair.Small) {
		fmt.Printf("%s already deleted", pair.Small)
		return
	}
	if !fileExists(pair.Big) {
		fmt.Printf("%s already deleted", pair.Big)
		return
	}
	if strings.HasSuffix(pair.Small, "-small.jpg") {
		fmt.Printf("%s skipped small", pair.Small)
		return
	}

	if alwaysDelete {
		if err := os.Remove(pair.Small); err != nil {
			log.Fatalf("unable to remove file: %s, err: %s", pair.Small, err)
		}
		log.Infof("deleted %s", pair.Small)
		return
	}

	reviewPairInteractive(idx, total, pair, viewer)
}

// reviewPairInteractive opens both images in a viewer and asks the user whether to delete.
func reviewPairInteractive(idx, total int, pair logger.DeleteEntry, viewer string) {
	largeImageProcess, err := openImage(viewer, pair.Big)
	if err != nil {
		log.Fatal("error opening large image ", err)
	}

	smallImageProcess, err := openImage(viewer, pair.Small)
	if err != nil {
		log.Fatal("error opening small image ", err)
	}

	var del string
	fmt.Printf("[%d/%d]\tdelete: %s ? ", idx+1, total, pair.Small)
	if _, err := fmt.Scanln(&del); err != nil {
		log.Fatalf("unable to read delete input: %s, err: %s", del, err)
	}
	if strings.TrimSpace(del) == "y" {
		if err := os.Remove(pair.Small); err != nil {
			log.Fatal(err)
		}
		log.Infof("deleted %s", pair.Small)
	}

	if err := closeImage(largeImageProcess); err != nil {
		log.Fatal("error closing large image: ", err)
	}
	if err := closeImage(smallImageProcess); err != nil {
		log.Fatal("error closing small image: ", err)
	}
}

// fileExists returns true if the file exists
func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}

func openImage(viewerCmd, imagePath string) (*exec.Cmd, error) {
	// Validate viewerCmd against an allow-list to avoid launching arbitrary subprocesses.
	switch viewerCmd {
	case viewerPreview, viewerEOG:
		// allowed
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedViewer, viewerCmd)
	}

	var imageProcess *exec.Cmd
	switch viewerCmd {
	case viewerPreview:
		// #nosec G204: imagePath is user-provided but comes from program arguments
		// intended for local CLI use; launching a trusted viewer is expected.
		imageProcess = exec.Command(viewerPreview, imagePath)
	case viewerEOG:
		// #nosec G204: imagePath is user-provided but comes from program arguments
		// intended for local CLI use; launching a trusted viewer is expected.
		imageProcess = exec.Command(viewerEOG, imagePath)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedViewer, viewerCmd)
	}

	if err := imageProcess.Start(); err != nil {
		return imageProcess, fmt.Errorf("error opening image %s, err: %w", imagePath, err)
	}

	return imageProcess, nil
}

func closeImage(process *exec.Cmd) error {
	if err := process.Process.Kill(); err != nil {
		return fmt.Errorf("error killing process: %w", err)
	}

	if err := process.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("error waiting for process: %w", err)
		}
	}
	return nil
}
