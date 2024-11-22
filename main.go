package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v66/github"
	"github.com/joho/godotenv"
)

type githubUser struct {
	auth   http.BasicAuth
	client *github.Client
}

var githubUsers []githubUser

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalln("Error loading .env file")
	}

	credentials := strings.Split(os.Getenv("CREDENTIALS"), ",")

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
			repoPath := path.Join("repos", *repo.FullName)

			if ok, err := exists(repoPath); err != nil {
				return err
			} else if !ok {
				if _, err := git.PlainClone(repoPath, false, &git.CloneOptions{URL: *repo.CloneURL, Auth: &g.auth, Mirror: true}); err != nil {
					return err
				}
			} else if r, err := git.PlainOpen(repoPath); err != nil {
				return err
			} else if wt, err := r.Worktree(); err != nil {
			} else {
				if err := wt.Pull(&git.PullOptions{Auth: &g.auth}); err != nil {
					return err
				}
			}
		}

		return nil
	}
}

func main() {
	for _, user := range githubUsers {
		go user.backup()
	}
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
