package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-errors/errors"
	"github.com/integrii/flaggy"
	"github.com/jesseduffield/lazygit/pkg/app"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/env"
)

var (
	commit      string
	version     = "unversioned"
	date        string
	buildSource = "unknown"
)

func startLoging() {
	logFile, err := os.OpenFile("../lazydog.log", (os.O_RDWR | os.O_CREATE | os.O_APPEND), 0666)
	if err != nil {
		log.Fatalf("file=logFile err=%s", err.Error())
	}

	multiLogFile := io.MultiWriter(os.Stdout, logFile)

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(multiLogFile)
}

func main() {
	startLoging()
	flaggy.DefaultParser.ShowVersionWithVersionFlag = false

	repoPath := ""
	flaggy.String(&repoPath, "p", "path", "Path of git repo. (equivalent to --work-tree=<path> --git-dir=<path>/.git/)")

	filterPath := ""
	flaggy.String(&filterPath, "f", "filter", "Path to filter on in `git log -- <path>`. When in filter mode, the commits, reflog, and stash are filtered based on the given path, and some operations are restricted")

	dump := ""
	flaggy.AddPositionalValue(&dump, "gitargs", 1, false, "Todo file")
	flaggy.DefaultParser.PositionalFlags[0].Hidden = true

	versionFlag := false
	flaggy.Bool(&versionFlag, "v", "version", "Print the current version")

	debuggingFlag := false
	flaggy.Bool(&debuggingFlag, "d", "debug", "Run in debug mode with logging (see --logs flag below). Use the LOG_LEVEL env var to set the log level (debug/info/warn/error)")

	logFlag := false
	flaggy.Bool(&logFlag, "l", "logs", "Tail lazygit logs (intended to be used when `lazygit --debug` is called in a separate terminal tab)")

	configFlag := false
	flaggy.Bool(&configFlag, "c", "config", "Print the default config")

	workTree := ""
	flaggy.String(&workTree, "w", "work-tree", "equivalent of the --work-tree git argument")

	gitDir := ""
	flaggy.String(&gitDir, "g", "git-dir", "equivalent of the --git-dir git argument")

	flaggy.Parse()

	log.Println(repoPath)
	log.Println(filterPath)
	log.Println(dump)
	log.Println(logFlag)
	log.Println(debuggingFlag)
	log.Println(gitDir)

	if repoPath != "" {
		if workTree != "" || gitDir != "" {
			log.Fatal("--path option is incompatible with the --work-tree and --git-dir options")
		}

		workTree = repoPath
		gitDir = filepath.Join(repoPath, ".git")
	}

	if workTree != "" {
		env.SetGitWorkTreeEnv(workTree)
	}

	if gitDir != "" {
		env.SetGitDirEnv(gitDir)
	}

	if versionFlag {
		fmt.Printf("commit=%s, build date=%s, build source=%s, version=%s, os=%s, arch=%s\n", commit, date, buildSource, version, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	if configFlag {
		fmt.Printf("%s\n", config.GetDefaultConfig())
		os.Exit(0)
	}

	if logFlag {
		app.TailLogs()
		os.Exit(0)
	}

	if workTree != "" {
		if err := os.Chdir(workTree); err != nil {
			log.Fatal(err.Error())
		}
	}

	appConfig, err := config.NewAppConfig("lazygit", version, commit, date, buildSource, debuggingFlag)
	if err != nil {
		log.Fatal(err.Error())
	}

	app, err := app.NewApp(appConfig, filterPath)

	if err == nil {
		err = app.Run()
	}

	if err != nil {
		if errorMessage, known := app.KnownError(err); known {
			log.Fatal(errorMessage)
		}
		newErr := errors.Wrap(err, 0)
		stackTrace := newErr.ErrorStack()
		app.Log.Error(stackTrace)

		log.Fatal(fmt.Sprintf("%s\n\n%s", app.Tr.SLocalize("ErrorOccurred"), stackTrace))
	}
}
