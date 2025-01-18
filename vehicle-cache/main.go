package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
)

func getEnv(key string) string {
	return os.Getenv(key)
}

func clone(repo string, dir string) (*git.Repository, error) {
	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: repo,
	})
	if err != nil {
		return r, err
	}
	return r, nil
}

func pull(repo *git.Repository) error {
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Pull(&git.PullOptions{})
	if err != nil && err.Error() != "already up-to-date" {
		return err
	}
	revision, err := repo.ResolveRevision("HEAD")
	if err != nil {
		return err
	}
	log.Printf("at revision: %s", revision)
	return nil
}

func check_updates(repo *git.Repository) {
	for {
		err := pull(repo)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second * 90)
	}
}

func main() {
	git_repo := os.Getenv("VT_GIT_REPO")
	git_file := os.Getenv("VT_GIT_FILE")
	http_port := os.Getenv("VT_HTTP_PORT")

	git_dir, err := os.MkdirTemp(os.TempDir(), "vehicle")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(git_dir)

	repo, err := clone(git_repo, git_dir)
	if err != nil {
		log.Fatal(err)
	}

	go check_updates(repo)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, fmt.Sprintf("%s/%s", git_dir, git_file))
	})

	fmt.Println(fmt.Sprintf("Server is running on port %s", http_port))
	err = http.ListenAndServe(fmt.Sprintf(":%s", http_port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
