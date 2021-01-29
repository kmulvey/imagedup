package main

import (
	"bufio"
	"encoding/json"
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
