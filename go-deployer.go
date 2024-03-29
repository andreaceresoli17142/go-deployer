package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

const (
	defaultPolling = 600 // default 10 sec polling
)

type Repository struct {
	Job     string `json:"job"`
	Name    string `json:"name"`
	Url     string `json:"url"`
	Remote  string `json:"remote"`
	Path    string `json:"path"`
	Polling int    `json:"polling"`
	Force   bool   `json:"force"`
	Script  string `json:"script"`
}

type TomlData struct {
	RepoConfig string `toml:"repoconfig"`
	SshKey     string `toml:"sshkey"`
	SshPw      string `toml:"sshpw"`
}

func notify(s string) {
	exec.Command("notify-send", s).Run()
	fmt.Println(s)
}

func main() {

	var configFile = "config.toml"

	if len(os.Args) == 2 {
		configFile = os.Args[1]
	}

	var config TomlData
	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		fmt.Println("Error reading config file:", err.Error())
		os.Exit(1)
	}

	var publicKeys *ssh.PublicKeys
	privateKeyFile := config.SshKey
	password := config.SshPw
	repoConfig := config.RepoConfig

	if repoConfig == "" {
		repoConfig = "repos.json"
	}

	_, err = os.Stat(privateKeyFile)
	if err != nil {
		fmt.Printf("Error reading file \"%s\": %s\n", privateKeyFile, err.Error())
		os.Exit(1)
	}

	publicKeys, err = ssh.NewPublicKeysFromFile("git", privateKeyFile, password)

	if err != nil {
		fmt.Printf("Error generating publickeys: %s\n", err.Error())
		os.Exit(1)
	}

	var repositories []Repository

	if err := loadJson(repoConfig, &repositories); err != nil {
		fmt.Println(err)
	}

	for _, v := range repositories {
		go startPolling(v, publicKeys)
	}

	cancelChan := make(chan os.Signal, 1)
	// catch SIGETRM or SIGINTERRUPT
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	sig := <-cancelChan
	fmt.Printf("caught: %v\nclosing up shop", sig)

	for _, v := range repositories {
		execJob(v, publicKeys)
	}

	fmt.Println("finished last tasks")
}

func startPolling(repo Repository, sshAuth *ssh.PublicKeys) {

	if repo.Polling == 0 {
		repo.Polling = defaultPolling
	}

	for {
		execJob(repo, sshAuth)
		time.Sleep(time.Second * time.Duration(repo.Polling))
	}
}

func execJob(repo Repository, sshAuth *ssh.PublicKeys) {
	var err error

	switch repo.Job {
	case "pull":
		err = updateIfChanged(sshAuth, repo)
		break
	case "push":
		err = pushIfChanged(sshAuth, repo)
		break
	}
	if err != nil && err != git.NoErrAlreadyUpToDate {
		notify(repo.Name + ": " + err.Error())
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

func updateIfChanged(sshAuth *ssh.PublicKeys, repo Repository) (err error) {

	name := repo.Name
	path := repo.Path
	remoteName := repo.Remote
	force := repo.Force
	script := repo.Script

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
			behind = v.Hash() != localHead.Hash()
			break
		}
	}

	if found && behind {
		var w *git.Worktree
		w, err = local.Worktree()

		if err != nil {
			return
		}

		if force {
			w.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: localHead.Hash()})
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

		if script != "" {
			cmd := exec.Command("sh", script)
			err = cmd.Run()
			if err != nil {
				err = fmt.Errorf("executing script returned an error, %s", err.Error())
				return
			}
		}

		return
	}
	return
}

func pushIfChanged(sshAuth *ssh.PublicKeys, repo Repository) (err error) {

	name := repo.Name
	path := repo.Path
	force := repo.Force
	script := repo.Script

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

	// currently broken
	//_, err = w.Add(".")

	// awful fix
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = w.Filesystem.Root()
	err = cmd.Run()

	if err != nil {
		return
	}

	timeNow := time.Now()
	year, month, day := timeNow.Date()

	hour, minutes := timeNow.Hour(), timeNow.Minute()

	_, err = w.Commit(fmt.Sprintf("go-deployer auto-commit: %d/%d/%d %d:%d", day, month, year, hour, minutes), &git.CommitOptions{})

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

	if script != "" {
		cmd := exec.Command("sh", script)
		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("executing script returned an error, %s", err.Error())
			return
		}
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
