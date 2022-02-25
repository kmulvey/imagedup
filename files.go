package main

import (
	"io/ioutil"
	"path"
	"strings"
)

// listFiles recursivly traverses the root directory and adds every .jpg to a string slice and returns it
func listFiles(root string) ([]string, error) {
	var allFiles []string
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return allFiles, err
	}
	for _, file := range files {
		if file.IsDir() {
			var subFiles, err = listFiles(path.Join(root, file.Name()))
			if err != nil {
				return allFiles, err
			}
			allFiles = append(allFiles, subFiles...)
		} else {
			if strings.HasSuffix(file.Name(), ".jpg") {
				allFiles = append(allFiles, path.Join(root, file.Name()))
			}
		}
	}
	return allFiles, nil
}

func streamFiles(files []string, pairChan chan pair, killChan chan struct{}) {
	for i, one := range files {
		for j, two := range files {
			if i != j {
				// this protects us from getting nil exception when shutting down
				select {
				case _, open := <-killChan:
					if !open {
						close(pairChan)
						return
					}
				default:
					pairChan <- pair{One: one, Two: two, I: i, J: j}
					pairTotal.Inc()
				}
			}
		}
	}
	close(pairChan)
}
