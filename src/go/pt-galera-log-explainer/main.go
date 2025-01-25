package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/percona/percona-toolkit/src/go/pt-galera-log-explainer/regex"
	"github.com/percona/percona-toolkit/src/go/pt-galera-log-explainer/translate"
	"github.com/percona/percona-toolkit/src/go/pt-galera-log-explainer/types"
	"github.com/percona/percona-toolkit/src/go/pt-galera-log-explainer/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	toolname = "pt-galera-log-explainer"
)

// We do not set anything here, these variables are defined by the Makefile
var (
	Build     string //nolint
	GoVersion string //nolint
	Version   string //nolint
	Commit    string //nolint
)

var buildInfo = fmt.Sprintf("%s\nVersion %s\nBuild: %s using %s\nCommit: %s", toolname, Version, Build, GoVersion, Commit)

var CLI struct {
	NoColor               bool
	Since                 *time.Time      `help:"Only list events after this date, format: 2023-01-23T03:53:40Z (RFC3339)"`
	Until                 *time.Time      `help:"Only list events before this date"`
	Verbosity             types.Verbosity `type:"counter" short:"v" default:"0" help:"-v: DebugMySQL (add every mysql info the tool used), -vv: Debug (internal tool debug)"`
	PxcOperator           bool            `default:"false" help:"Analyze logs from Percona PXC operator. Will cause slow performance on non-k8s setups"`
	SkipOperatorDetection bool            `default:"false" help:"Skip auto detection of Percona PXC operator logs"`
	ExcludeRegexes        []string        `help:"Remove regexes from analysis. List regexes using 'pt-galera-log-explainer regex-list'"`
	MergeByDirectory      bool            `help:"Instead of relying on identification, merge contexts and columns by base directory. Very useful when dealing with many small logs organized per directories."`
	SkipMerge             bool            `help:"Disable the ability to merge log files together. Can be used when every nodes have the same wsrep_node_name"`

	List  list  `cmd:""`
	Whois whois `cmd:""`
	//	Sed       sed       `cmd:""`
	Ctx       ctx       `cmd:""`
	RegexList regexList `cmd:""`
	Conflicts conflicts `cmd:""`

	Version kong.VersionFlag

	GrepCmd string `help:"'grep' command path. Could need to be set to 'ggrep' for darwin systems" default:"grep"`

	CustomRegexes map[string]string `help:"Add custom regexes, printed in magenta. Format: (golang regex string)=[optional static message to display]. If the static message is left empty, the captured string will be printed instead. Custom regexes are separated using semi-colon. Example: --custom-regexes=\"Page cleaner took [0-9]*ms to flush [0-9]* pages=;doesn't recommend.*pxc_strict_mode=unsafe query used\""`
}

func main() {
	kongcli := kong.Parse(&CLI,
		kong.Name(toolname),
		kong.Description("An utility to merge and help analyzing Galera logs"),
		kong.UsageOnError(),
		kong.Vars{
			"version": buildInfo,
		},
	)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: CLI.NoColor, FormatTimestamp: func(_ interface{}) string { return "" }})
	initComponentLogger()
	if CLI.Verbosity == types.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	utils.SkipColor = CLI.NoColor

	err := regex.AddCustomRegexes(CLI.CustomRegexes)
	kongcli.FatalIfErrorf(err)

	for _, path := range kongcli.Path {
		if path.Positional != nil && path.Positional.Name == "paths" {
			paths, ok := path.Positional.Target.Interface().([]string)
			if ok && !CLI.PxcOperator && !CLI.SkipOperatorDetection && areOperatorFiles(paths) {
				CLI.PxcOperator = true
				log.Info().Msg("Detected logs coming from Percona XtraDB Cluster Operator, enabling --pxc-operator")
			}
		}
	}

	translate.AssumeIPStable = !CLI.PxcOperator

	err = kongcli.Run()
	kongcli.FatalIfErrorf(err)
}
