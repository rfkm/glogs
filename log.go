package main

import (
	"bufio"
	"bytes"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

const headerBodySep = ": "

type CommitLog struct {
	hunk string
	repo *Repository
}

type CommitLogChannel chan *CommitLog

func (l *CommitLog) Format(oneline bool) string {
	hunk := strings.Replace(l.hunk, "%rn", l.repo.name, -1)
	hunk = strings.Replace(hunk, "%rp", l.repo.shortPath, -1)
	ret := ""
	if oneline {
		ret += toOneline(hunk)
	} else {
		ret += hunk
	}

	return ret
}

func toOneline(hunk string) string {
	ret := ""
	for _, s := range strings.Split(strings.Replace(hunk, "\n", headerBodySep, 1), "\n") {
		ret += strings.Trim(s, " ") + " "
	}
	return strings.TrimSuffix(ret, headerBodySep)
}

func CombinedGitLogs(repos RepositoryChannel, logFormat string, parallelism int) (out CommitLogChannel) {
	if parallelism < 1 {
		parallelism = 1
	}
	out = make(CommitLogChannel)
	w := make(chan bool, parallelism)
	wg := new(sync.WaitGroup)
	go func() {
		for repo := range repos {
			wg.Add(1)
			w <- true
			go func(repo *Repository) {
				for l := range GitLogs(repo, logFormat) {
					out <- l
				}
				wg.Done()
				<-w
			}(repo)
		}
		wg.Wait()
		close(out)
	}()

	return
}

func GitLogs(repo *Repository, format string) (c CommitLogChannel) {
	c = make(CommitLogChannel)
	path := repo.path
	cmd := exec.Command("git", "log", "--pretty=format:"+format+"%x07")
	cmd.Dir = path
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		cmd.Start()
		scanner := bufio.NewScanner(stdout)
		scanner.Split(splitLogHunk)
		for scanner.Scan() {
			c <- &CommitLog{
				hunk: scanner.Text(),
				repo: repo,
			}
		}
		close(c)
	}()
	return
}

func splitLogHunk(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	var i, j int
	if i = bytes.IndexByte(data, '\x07'); i >= 0 {
		if j = bytes.IndexByte(data[i:], '\n'); j >= 0 {
			return i + j + 1, dropSep(dropCR(data[0 : i+j])), nil
		}
	}
	if atEOF {
		return len(data), dropSep(dropCR(data)), nil
	}
	return 0, nil, nil
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func dropSep(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\x07' {
		return data[0 : len(data)-1]
	}
	return data
}

// Filter

func (l *CommitLog) Match(re *regexp.Regexp) bool {
	return StrMatch(l.hunk, re)
}

func (c CommitLogChannel) Include(patterns []string) (out CommitLogChannel) {
	out = make(CommitLogChannel)
	WrapFilterableChannel(c).applyIncludeFilter(patterns).UnwrapFilterableChannel(out)
	return
}

func (c CommitLogChannel) Exclude(patterns []string) (out CommitLogChannel) {
	out = make(CommitLogChannel)
	WrapFilterableChannel(c).applyExcludeFilter(patterns).UnwrapFilterableChannel(out)
	return
}
