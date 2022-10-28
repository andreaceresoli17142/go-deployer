package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

type repo struct {
	remote string
	Local  string `json:"local"`
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

	// load the rest of the repository data
	// code is scuffed, needs to be rewrtitten
	repo, err := git.PlainOpen(repos[0].Local)

	repoData, err := repo.Head()
	fmt.Println(repoData.Hash())
	// using polling look ar repo hashes to determine if they are up-to-date

}
