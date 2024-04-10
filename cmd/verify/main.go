package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/kmulvey/imagedup/v2/pkg/imagedup/logger"
	"github.com/kmulvey/path"
	log "github.com/sirupsen/logrus"
	"go.szostok.io/version"
	"go.szostok.io/version/printer"
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
		log.Fatal(err)

	}

	for _, deleteFile := range files {

		var dedupedFiles, err = logger.ReadDeleteLogFile(deleteFile.AbsolutePath)
		if err != nil {
			log.Fatal(err)
		}

		for i, pair := range dedupedFiles {

			// we could has already deleted one of them, so just go around
			if !fileExists(pair.Small) {
				fmt.Println(pair.Small, " already deleted")
				continue
			}
			if !fileExists(pair.Big) {
				fmt.Println(pair.Big, " already deleted")
				continue
			}

			if alwaysDelete {
				if err := os.Remove(pair.Small); err != nil {
					log.Fatal(err)
				}
				fmt.Println("deleted", pair.Small)
				continue
			}

			var viewerCmd string
			var goos = runtime.GOOS
			switch goos {
			case "windows":
			case "darwin":
				viewerCmd = "preview"
			case "linux":
				viewerCmd = "eog" // eog -- GNOME Image Viewer 41.1
			default:
				log.Fatalf("unsupported os: %s", goos)
			}
			// open both images with image viewer
			cmdBig := exec.Command(viewerCmd, pair.Big)
			if err := cmdBig.Start(); err != nil {
				log.Fatal(err)
			}

			cmdSmall := exec.Command(viewerCmd, pair.Small)
			if err := cmdSmall.Start(); err != nil {
				log.Fatal(err)
			}

			// ask the user if we should delete
			var del string
			fmt.Printf("[%d/%d]	delete: %s ?", i+1, len(dedupedFiles), pair.Small)
			fmt.Scanln(&del)
			if del == "y" {
				if err := os.Remove(pair.Small); err != nil {
					log.Fatal(err)
				}
				fmt.Println("deleted", pair.Small)
			}
		}
	}
}

// fileExists returns true if the file exists
func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}
