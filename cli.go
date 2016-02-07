package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/codegangsta/cli"
)

var GlobalFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "e, exclude-repo",
		Value: nil,
		Usage: "Exclude repositories matching a given pattern.",
	},
	cli.StringSliceFlag{
		Name:  "i, include-repo",
		Value: nil,
		Usage: "Print commit logs of only repositories matching a given pattern. Note that -e patterns take priority over -i patterns.",
	},
	cli.StringSliceFlag{
		Name:  "E, exclude-log",
		Value: nil,
		Usage: "Exclude commit logs matching a given pattern.",
	},
	cli.StringSliceFlag{
		Name:  "I, include-log",
		Value: nil,
		Usage: "Print only commit logs matching a given pattern. Note that -E patterns take priority over -I patterns.",
	},
	cli.StringFlag{
		Name:  "f, format",
		Value: "[%rn] <%an> %B",
		Usage: "Print commit logs in a given format. See \"PRETTY FORMAT\" section of git-log(1) for more details.",
	},
	cli.IntFlag{
		Name:  "p, parallelism",
		Value: 50,
		Usage: "Limit the number of repositories read in parallel.",
	},
	cli.BoolFlag{
		Name:  "1, oneline",
		Usage: "Format each commit message to one line.",
	},
	cli.BoolFlag{
		Name:  "h,help",
		Usage: "Show help.",
	},
}

func ensureGhqExists() {
	err := exec.Command("ghq", "-v").Run()
	if err != nil {
		panic(err)
	}
	return
}

func Action(c *cli.Context) {
	if c.Bool("h") {
		cli.ShowAppHelp(c)
		return
	}

	stat, _ := os.Stdin.Stat()
	var repos RepositoryChannel
	if (stat.Mode() & os.ModeCharDevice) == 0 { // piped
		repos = GitReposFromStdin()
	} else {
		ensureGhqExists()
		repos = GitReposFromGhq()
	}
	repos = repos.Include(c.StringSlice("i")).Exclude(c.StringSlice("e"))
	logs := CombinedGitLogs(repos, c.String("f"), c.Int("p")).Include(c.StringSlice("I")).Exclude(c.StringSlice("E"))

	for l := range logs {
		fmt.Println(l.Format(c.Bool("oneline")))
	}
}

func NewApp() *cli.App {
	cli.VersionFlag = cli.BoolFlag{
		Name:  "v, version",
		Usage: "Print the version.",
	}

	app := cli.NewApp()
	app.Name = Name
	app.Version = Version
	// app.Author = "@rfkm"
	app.Email = ""
	app.Usage = "Command line tool to dump commit logs across repositories managed by ghq"
	app.ArgsUsage = " "

	app.Flags = GlobalFlags
	app.HideHelp = true
	app.Action = Action

	return app
}
