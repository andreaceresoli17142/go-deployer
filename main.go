package main

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	_ "github.com/go-git/go-git/v5/_examples"
)

func main() {
	path := os.Args[1]

	local, err := git.PlainOpen(path)

	if err != nil {
		fmt.Println(err)
	}

	localHead, err := local.Head()

	if err != nil {
		fmt.Println(err)
	}

	remote, err := local.Remote("origin")

	if err != nil {
		fmt.Println(err)
	}

	references, err := remote.List(&git.ListOptions{})

	if err != nil {
		fmt.Println(err)
	}

	found, behind := false, false
	_ = found
	_ = behind
	for _, v := range references {
		if v.Name() == localHead.Name() {
			found = true
			behind = v.Hash() == localHead.Hash()
			break
		}
	}

	if found && !behind {
		fmt.Println("updating repository")
		w, err := local.Worktree()

		if err != nil {
			fmt.Println(err)
		}

		err = w.Pull(&git.PullOptions{RemoteName: "origin"})

		if err != nil {
			fmt.Println(err)
		}
		return
	}
	fmt.Println("repository already at latest change")
}
