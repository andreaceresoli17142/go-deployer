package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type repo struct {
	remote string
	local  string `json:"local"`
	branch string
	hash   string
}

func main() {
	fmt.Println("application started")

	// read json file
	repofile, err := os.ReadFile("repos.json")
	if err != nil {
		fmt.Printf("error reading json file: %v\n", err)
	}

	// read all repositories from the json
	var repos []repo
	if err = json.Unmarshal(repofile, &repos); err != nil {
		fmt.Printf("error unmarshaling json: %v\n", err)
	}

	// using polling look ar repo hashes to determine if they are up-to-date

}
