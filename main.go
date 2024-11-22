package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v66/github"
	"github.com/joho/godotenv"
	"github.com/robfig/cron"
)

type githubUser struct {
	auth   http.BasicAuth
	client *github.Client
}

var githubUsers []githubUser

const dataDir = "/mnt/data"

func init() {
	if err := godotenv.Load(path.Join(dataDir, ".env")); err != nil {
		fmt.Println(path.Join(dataDir, ".env"))

		log.Fatalln("Error loading .env file")
	}

	credentials := strings.Split(os.Getenv("CREDENTIALS_GITHUB"), ",")

	for _, credential := range credentials {
		components := strings.Split(credential, ":")

		if len(components) == 2 {
			githubUsers = append(githubUsers, githubUser{
				auth: http.BasicAuth{
					Username: components[0],
					Password: components[1],
				},
			})
		}
	}
}

func (g *githubUser) backup() error {
	g.client = github.NewClient(nil).WithAuthToken(g.auth.Password)

	opt := &github.RepositoryListByAuthenticatedUserOptions{Type: "all"}
	if repos, _, err := g.client.Repositories.ListByAuthenticatedUser(context.Background(), opt); err != nil {
		return err
	} else {
		for _, repo := range repos {
			// check, wether the repo is already cloned
			repoPath := path.Join(dataDir, *repo.FullName)

			if ok, err := exists(repoPath); err != nil {
				return fmt.Errorf("can't check for existance of %q: %v", repoPath, err)
			} else if !ok {
				if _, err := git.PlainClone(path.Join(repoPath, ".git"), false, &git.CloneOptions{URL: *repo.CloneURL, Mirror: true, Auth: &g.auth}); err != nil {
					return fmt.Errorf("can't clone %q: %v", *repo.CloneURL, err)
				} else {
					fmt.Printf("cloned repo %q\n", *repo.FullName)
				}
			} else if r, err := git.PlainOpen(repoPath); err != nil {
				return fmt.Errorf("can't open repo at %q: %v", repoPath, err)
			} else if err := r.FetchContext(context.Background(), &git.FetchOptions{Auth: &g.auth}); err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					fmt.Printf("repo %q already up to date\n", *repo.FullName)
				} else {
					return fmt.Errorf("can't fetch repo %q: %v", *repo.FullName, err)
				}
			} else {
				fmt.Printf("fetched repo %q\n", *repo.FullName)
			}
		}

		return nil
	}
}

func backupAll() {
	var wg sync.WaitGroup

	for _, user := range githubUsers {
		wg.Add(1)

		go func(user githubUser) {
			if err := user.backup(); err != nil {
				fmt.Println(err.Error())
			}

			wg.Done()
		}(user)
	}

	wg.Wait()
}

func exists(pth string) (bool, error) {
	if _, err := os.Stat(pth); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

func main() {
	// set up cronjob
	c := cron.New()

	var cronInterval string
	var ok bool
	if cronInterval, ok = os.LookupEnv("INTERVAL"); !ok {
		cronInterval = "0 0 0 * * * "
	}

	c.AddFunc(cronInterval, backupAll)

	// run at start
	backupAll()

	c.Start()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
