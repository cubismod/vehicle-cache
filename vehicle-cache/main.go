package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	ristretto_store "github.com/eko/gocache/store/ristretto/v4"
	"github.com/go-git/go-git/v5"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/json"
)

func getEnv(key string) string {
	return os.Getenv(key)
}

func clone(repo string, dir string) (*git.Repository, error) {
	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:   repo,
		Depth: 2,
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

func check_updates(repo *git.Repository, git_dir string, git_file string, cache *cache.Cache[string]) {
	mediatype := "application/json"
	m := minify.New()
	m.AddFuncRegexp(regexp.MustCompile(".+"), json.Minify)
	for {
		err := pull(repo)
		if err != nil {
			log.Fatal(err)
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Minute))
		defer cancel()

		file, err := os.ReadFile(fmt.Sprintf("%s/%s", git_dir, git_file))
		if err != nil {
			log.Fatal(err)
		}

		s, err := m.String(mediatype, string(file))
		if err != nil {
			log.Fatal(err)
		}

		err = cache.Set(ctx, "out", s)
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

	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}

	ristrettoStore := ristretto_store.NewRistretto(ristrettoCache)

	cacheManager := cache.New[string](ristrettoStore)

	go check_updates(repo, git_dir, git_file, cacheManager)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("{'%s': '%s'}", name, value)
			}
		}
		log.Printf("Received request: %s %s from %s\n",
			r.Method, r.URL.Path, r.RemoteAddr)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		val, err := cacheManager.Get(ctx, "out")
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = io.WriteString(w, val)
		if err != nil {
			log.Fatal(err)
		}
	})

	fmt.Println(fmt.Sprintf("Server is running on port %s", http_port))
	err = http.ListenAndServe(fmt.Sprintf(":%s", http_port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
