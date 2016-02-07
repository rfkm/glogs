package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Repository struct {
	path      string
	shortPath string
	name      string
}

type RepositoryChannel chan *Repository

func ghqRoots() []string {
	cmd := exec.Command("ghq", "root", "--all")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	cmd.Start()
	defer cmd.Wait()

	scanner := bufio.NewScanner(stdout)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func extractRelPath(roots []string, fullpath string) string {
	for _, root := range roots {
		if strings.HasPrefix(fullpath, root) {
			return strings.TrimPrefix(fullpath, root)
		}
	}

	return fullpath
}

func extractShortPath(roots []string, fullPath string) string {
	rel := extractRelPath(roots, fullPath)
	return strings.TrimPrefix(rel, "/")
}

func extractName(fullPath string) string {
	return filepath.Base(fullPath)
}

func GitReposFromGhq() (c RepositoryChannel) {
	roots := ghqRoots()
	c = make(RepositoryChannel)
	// TODO: Should make the command configurable?
	cmd := exec.Command("ghq", "list", "-p")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	go func() {
		cmd.Start()
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		var repo string
		for scanner.Scan() {
			repo = scanner.Text()
			c <- &Repository{
				path:      repo,
				shortPath: extractShortPath(roots, repo),
				name:      extractName(repo),
			}
		}
		close(c)
	}()
	return
}

func GitReposFromStdin() (c RepositoryChannel) {
	c = make(RepositoryChannel)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		var repo string
		for scanner.Scan() {
			repo = scanner.Text()
			c <- &Repository{
				path:      repo,
				shortPath: extractShortPath(nil, repo),
				name:      extractName(repo),
			}
		}
		close(c)
	}()
	return
}

// Filter

func (r *Repository) Match(re *regexp.Regexp) bool {
	return StrMatch(r.path, re)
}

func (c RepositoryChannel) Include(patterns []string) (out RepositoryChannel) {
	out = make(RepositoryChannel)
	WrapFilterableChannel(c).applyIncludeFilter(patterns).UnwrapFilterableChannel(out)
	return
}

func (c RepositoryChannel) Exclude(patterns []string) (out RepositoryChannel) {
	out = make(RepositoryChannel)
	WrapFilterableChannel(c).applyExcludeFilter(patterns).UnwrapFilterableChannel(out)
	return
}
