package main

import (
	"fmt"
	"os"
	"os/exec"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

const (
	defaultPolling = 10 // 10 sec polling
)

type JobType int

const (
	keepUpdated JobType = iota
	keepPushing JobType = iota
)

type Repository struct {
	Job     JobType `json:"job"`
	Name    string  `json:"name"`
	Url     string  `json:"url"`
	Remote  string  `json:"remote"`
	Path    string  `json:"path"`
	Polling int     `json:"polling"`
}

func notify(s string) {
	exec.Command("notify-send", "started").Run()
}

func main() {

	path := os.Args[1]
	remoteName := os.Args[2]

	var publicKeys *ssh.PublicKeys

	if len(os.Args) >= 4 {
		privateKeyFile := os.Args[3]
		var password string

		if len(os.Args) == 5 {
			password = os.Args[4]
		}

		_, err := os.Stat(privateKeyFile)
		if err != nil {
			fmt.Println("read file %s failed %s\n", privateKeyFile, err.Error())
			return
		}

		// Clone the given repository to the given directory
		publicKeys, err = ssh.NewPublicKeysFromFile("git", privateKeyFile, password)
		if err != nil {
			fmt.Println("generate publickeys failed: %s\n", err.Error())
			return
		}
		fmt.Println(publicKeys)
	}

	if err := updateIfChanged(publicKeys, path, remoteName, true); err != nil {
		fmt.Println(err)
		notify(err.Error())
	}
}

func updateIfChanged(sshAuth *ssh.PublicKeys, path string, remoteName string, force bool) (err error) {

	local, err := git.PlainOpen(path)

	if err != nil {
		return
	}

	localHead, err := local.Head()

	if err != nil {
		return
	}

	remote, err := local.Remote(remoteName)

	if err != nil {
		return
	}

	listAuth := &git.ListOptions{}
	if sshAuth != nil {
		listAuth.Auth = sshAuth
	}

	references, err := remote.List(listAuth)

	if err != nil {
		return
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
		var w *git.Worktree
		fmt.Println("updating repository")
		w, err = local.Worktree()

		if err != nil {
			return
		}

		pullAuth := &git.PullOptions{RemoteName: remoteName, Force: force}
		if sshAuth != nil {
			pullAuth.Auth = sshAuth
		}

		err = w.Pull(pullAuth)

		if err != nil {
			return
		}
		fmt.Println("repository succesfully updated")
		return
	}
	fmt.Println("repository already at latest change")
	return
}
