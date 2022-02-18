package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
)

type pair struct {
	Big   string
	Small string
}

func main() {
	var alwaysDelete bool
	flag.BoolVar(&alwaysDelete, "always-delete", false, "just take the larger one, always")
	flag.Parse()

	var file, err = os.Open("delete.log")
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var p pair
		var err = json.Unmarshal(scanner.Bytes(), &p)
		if err != nil {
			log.Fatal(err)
		}

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

		// open both images with image viewer "eog -- GNOME Image Viewer 41.1"
		cmd := exec.Command("eog", p.Big)
		cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		cmdS := exec.Command("eog", p.Small)
		cmdS.Run()
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
