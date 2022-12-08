package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

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
	Force   bool    `json:"force"`
}

func notify(s string) {
	exec.Command("notify-send", s).Run()
}

func main() {

	/*path := os.Args[1]
	remoteName := os.Args[2]*/

	//TODO: fix the use of ssh keys
	var publicKeys *ssh.PublicKeys
	if len(os.Args) >= 2 {
		privateKeyFile := os.Args[1]
		var password string

		if len(os.Args) == 3 {
			password = os.Args[2]
		}

		_, err := os.Stat(privateKeyFile)
		if err != nil {
			fmt.Printf("read file %s failed %s\n", privateKeyFile, err.Error())
			return
		}

		publicKeys, err = ssh.NewPublicKeysFromFile("git", privateKeyFile, password)
		if err != nil {
			fmt.Printf("generate publickeys failed: %s\n", err.Error())
			return
		}
	}

	var repositories []Repository

	if err := loadJson("repos.json", &repositories); err != nil {
		fmt.Println(err)
	}

	//fmt.Println(repositories)

	for _, v := range repositories {
		go startPolling(v, publicKeys)
	}

	// this is just to stop the program from exiting after starting all the goroutines
	for {
		time.Sleep(time.Second * 10000)
	}
}

func startPolling(repo Repository, sshAuth *ssh.PublicKeys) {

	if repo.Polling == 0 {
		repo.Polling = defaultPolling
	}

	var err error

	for {
		switch repo.Job {
		case keepUpdated:
			err = updateIfChanged(sshAuth, repo.Name, repo.Path, repo.Remote, repo.Force)
			break
		case keepPushing:
			err = pushIfChanged(sshAuth, repo.Name, repo.Path, repo.Force)
			break
		}
		if err != nil {
			notify(repo.Name + ": " + err.Error())
		}
		time.Sleep(time.Second * time.Duration(repo.Polling))
	}
}

func hasUnstagedChages(repo *git.Repository) (bool, error) {

	w, err := repo.Worktree()

	if err != nil {
		return false, err
	}

	s, err := w.Status()

	if err != nil {
		return false, err
	}

	return !s.IsClean(), nil
}

func updateIfChanged(sshAuth *ssh.PublicKeys, name string, path string, remoteName string, force bool) (err error) {

	local, err := git.PlainOpen(path)

	if err != nil {
		return
	}

	if !force {
		var unstChanges bool
		unstChanges, err = hasUnstagedChages(local)
		if err != nil {
			return
		}

		if unstChanges {
			return
		}
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

		if !force {
			notify(name + ": successfully pulled")
		}

		return
	}

	return
}

func pushIfChanged(sshAuth *ssh.PublicKeys, name string, path string, force bool) (err error) {

	local, err := git.PlainOpen(path)

	if err != nil {
		return
	}

	unstChange, err := hasUnstagedChages(local)

	if !unstChange || err != nil {
		return
	}

	w, err := local.Worktree()

	if err != nil {
		return
	}

	err = w.AddWithOptions(&git.AddOptions{All: true})

	if err != nil {
		return
	}

	day, month, year := time.Now().Date()

	_, err = w.Commit(fmt.Sprintf("automated commit: %d/%v/%d", day, month, year), &git.CommitOptions{})

	if err != nil {
		return
	}

	pushOpt := &git.PushOptions{Force: force}
	if sshAuth != nil {
		pushOpt.Auth = sshAuth
	}

	err = local.Push(pushOpt)

	if err != nil {
		return
	}

	if !force {
		notify(name + ": successfully pushed")
	}

	return
}

func loadJson[T any](FileName string, inp T) error {
	content, err := os.ReadFile(FileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(content, &inp)
	if err != nil {
		return err
	}

	return nil
}
