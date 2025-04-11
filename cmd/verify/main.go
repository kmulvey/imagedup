package main

import (
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

	var screenWidth, screenHeight = screenWidth()

	for _, deleteFile := range files {

		var dedupedFiles, err = logger.ReadDeleteLogFile(deleteFile.AbsolutePath)
		if err != nil {
			log.Fatalf("error reading file: %s, err: %s", deleteFile.AbsolutePath, err)
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
			if strings.HasSuffix(pair.Small, "-small.jpg") {
				fmt.Println(pair.Small, " skipped small")
				continue
			}

			if alwaysDelete {
				if err := os.Remove(pair.Small); err != nil {
					log.Fatalf("unable to remove file: %s, err: %s", pair.Small, err)
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
			largeImageProcess, err := openImage(viewerCmd, pair.Big, "left", screenWidth, screenHeight)
			if err != nil {
				log.Fatal("error opening large image ", err)
			}

			smallImageProcess, err := openImage(viewerCmd, pair.Small, "right", screenWidth, screenHeight)
			if err != nil {
				log.Fatal("error opening small image ", err)
			}

			// ask the user if we should delete
			var del string
			fmt.Printf("[%d/%d]	delete: %s ?", i+1, len(dedupedFiles), pair.Small)
			if _, err := fmt.Scanln(&del); err != nil {
				log.Fatalf("unale to read delete input: %s, err: %s", del, err)
			}
			if strings.TrimSpace(del) == "y" {
				if err := os.Remove(pair.Small); err != nil {
					log.Fatal(err)
				}
				fmt.Println("deleted", pair.Small)
			}

			// close the viewer
			if err := closeImage(largeImageProcess); err != nil {
				log.Fatal("error closing large image: ", err)
			}
			if err := closeImage(smallImageProcess); err != nil {
				log.Fatal("error closing small image: ", err)
			}
		}
	}
}

// fileExists returns true if the file exists
func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}

func screenWidth() (int, int) {
	// Get screen width (using xrandr to get the screen size)
	screenWidthCmd := exec.Command("xrandr")
	output, err := screenWidthCmd.Output()
	if err != nil {
		log.Fatal("error getting screen dimensions ", err)
	}

	// Find the screen width by parsing the xrandr output (assuming first * indicates active screen)
	var screenWidth, screenHeight int
	lines := strings.Split(string(output), "\n")
	var doesntMatter int
	if len(lines) > 0 {
		if _, err := fmt.Sscanf(strings.TrimSpace(lines[0]), "Screen %d: minimum %d x %d, current %d x %d, maximum %d x %d", &doesntMatter, &doesntMatter, &doesntMatter, &screenWidth, &screenHeight, &doesntMatter, &doesntMatter); err != nil {
			log.Fatal("error parsing xrandr output ", err)
		}
	} else {
		log.Fatal("xrandr output is empty")
	}

	return screenWidth, screenHeight
}

func openImage(viewerCmd, imagePath, orientation string, screenWidth, screenHeight int) (*exec.Cmd, error) {
	imageProcess := exec.Command(viewerCmd, imagePath)
	if err := imageProcess.Start(); err != nil {
		return imageProcess, fmt.Errorf("error opening image %s, err: %w", imagePath, err)
	}

	// Calculate 1/4 of the screen width for each window
	// quarterWidth := screenWidth / 4
	// halfHeight := screenHeight / 2

	// if orientation == "left" {
	//	// Move the first window to the left quarter of the screen
	//	resizeProcess := exec.Command("wmctrl", "-r", "Eye of GNOME", "-e", fmt.Sprintf("0,0,0,%d,%d", quarterWidth, screenHeight))
	//	if err := resizeProcess.Run(); err != nil {
	//		return imageProcess, fmt.Errorf("error resizing image %s, err: %w", imagePath, err)
	//	}
	// } else {
	//	resizeProcess := exec.Command("wmctrl", "-r", "Eye of GNOME", "-e", fmt.Sprintf("0,%d,0,%d,%d", quarterWidth*3, screenHeight, quarterWidth))
	//	if err := resizeProcess.Run(); err != nil {
	//		return imageProcess, fmt.Errorf("error resizing image %s, err: %w", imagePath, err)
	//	}
	// }

	return imageProcess, nil
}

func closeImage(process *exec.Cmd) error {
	if err := process.Process.Kill(); err != nil {
		return err
	}

	if err := process.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return err
		}
	}
	return nil
}
