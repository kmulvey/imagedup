package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	log "github.com/sirupsen/logrus"
)

type pair struct {
	Big   string
	Small string
}

func main() {
	var alwaysDelete bool
	var deleteFile string
	flag.BoolVar(&alwaysDelete, "always-delete", false, "just take the larger one, always")
	flag.StringVar(&deleteFile, "delete-file", "delete.log", "log file where duplicate pairs are stored, same file from -cache-file when running nsquared")
	flag.Parse()

	var file, err = os.Open(deleteFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range dedupFile(file) {

		// we could has already deleted one of them, so just go around
		if !fileExists(p.Small) {
			fmt.Println(p.Small, " already deleted")
			continue
		}
		if !fileExists(p.Big) {
			fmt.Println(p.Big, " already deleted")
			continue
		}

		if alwaysDelete {
			err = os.Remove(p.Small)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("deleted", p.Small)
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
		cmd := exec.Command(viewerCmd, p.Big)
		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}

		cmdS := exec.Command(viewerCmd, p.Small)
		err = cmdS.Run()
		if err != nil {
			log.Fatal(err)
		}

		// ask the user if we should delete
		var del string
		fmt.Print("delete ", p.Small, " ? ")
		fmt.Scanln(&del)
		if del == "y" {
			err = os.Remove(p.Small)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("deleted", p.Small)
		}
	}
}

func fileExists(fileName string) bool {
	if _, err := os.Stat(fileName); err == nil {
		return true
	}
	return false
}

func dedupFile(file *os.File) []pair {
	var scanner = bufio.NewScanner(file)
	var imagePairs []pair

FileLoop:
	for scanner.Scan() {
		var filePair pair
		var err = json.Unmarshal(scanner.Bytes(), &filePair)
		if err != nil {
			log.Fatal(err)
		}

		for _, p := range imagePairs {
			if filePair.Big == p.Big && filePair.Small == p.Small ||
				filePair.Big == p.Small && filePair.Small == p.Big {
				continue FileLoop
			}
		}
		imagePairs = append(imagePairs, filePair)
	}

	return imagePairs
}
